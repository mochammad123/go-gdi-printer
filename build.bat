@echo off
setlocal enabledelayedexpansion
title Knitto Print Service - Build Generator

echo ============================================================
echo   KNITTO PRINT SERVICE - BUILD GENERATOR (GOLANG)
echo ============================================================

:: Cek flag otomatis dari command line
set "CLI_BUNDLE="
if /i "%~1"=="-bundle" set "CLI_BUNDLE=1"
if /i "%~1"=="-nobundle" set "CLI_BUNDLE=2"
if /i "%~1"=="-external" set "CLI_BUNDLE=2"

:: ------------------------------------------------------------
:: 1. Pilihan Mode Template
:: ------------------------------------------------------------
if not "!CLI_BUNDLE!"=="" (
    set "CHOICE_BUNDLE=!CLI_BUNDLE!"
) else (
    echo.
    echo PILIHAN 1: MODE TEMPLATE
    echo   [1] Full Bundle  : Semua template JSON di-embed ke dalam binary .exe
    echo                      (Single file .exe mandiri, template bawaan tertanam)
    echo   [2] Pisah / Eksternal : File .exe berdiri sendiri + folder template/ di sampingnya
    echo                      (Mudah tambah/edit template JSON tanpa build ulang)
    echo.
    set /p "CHOICE_BUNDLE=Pilih mode template (1/2) [Default 1]: "
    if "!CHOICE_BUNDLE!"=="" set "CHOICE_BUNDLE=1"
)

:: ------------------------------------------------------------
:: 2. Pilihan Mode Tray
:: ------------------------------------------------------------
echo.
echo PILIHAN 2: MODE TRAY RUNTIME
echo   [1] System Tray Aktif : Ada icon taskbar tray + Control Panel Windows (Delphi-style)
echo   [2] Headless / CLI    : Hanya terminal console tanpa GUI (-tray=false)
echo.
set /p "CHOICE_TRAY=Pilih mode tray (1/2) [Default 1]: "
if "!CHOICE_TRAY!"=="" set "CHOICE_TRAY=1"

:: ------------------------------------------------------------
:: 3. Pilihan Sinkronisasi ke Sidecar Electron
:: ------------------------------------------------------------
echo.
echo PILIHAN 3: SINKRONISASI KE SIDECAR ELECTRON
echo   [1] Ya : Salin ke ..\Pengebalan\knitto-manpro-electron\sidecar\
echo   [2] Tidak : Hanya simpan di folder ini (golang\)
echo.
set /p "CHOICE_SYNC=Salin ke sidecar Electron? (1/2) [Default 1]: "
if "!CHOICE_SYNC!"=="" set "CHOICE_SYNC=1"

echo.
echo ============================================================
echo  MEMULAI PROSES BUILD...
echo ============================================================

:: Set build tags berdasarkan pilihan
set "BUILD_TAGS="
set "MODE_LABEL=Full Bundle (Embedded Templates)"

if "!CHOICE_BUNDLE!"=="2" (
    set "BUILD_TAGS=-tags nobundle"
    set "MODE_LABEL=Pisah / External Templates (No-Bundle)"
)

:: Set LDFLAGS (-H=windowsgui untuk pure background daemon tanpa CMD hitam ala Delphi)
set "LDFLAGS=-s -w -H=windowsgui"
if "!CHOICE_TRAY!"=="2" (
    set "LDFLAGS=-s -w"
)

echo [1/3] Menjalankan: go build !BUILD_TAGS! -ldflags="!LDFLAGS!" -o print-service.exe .
go build !BUILD_TAGS! -ldflags="!LDFLAGS!" -o print-service.exe .

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ BUILD GAGAL! Silakan periksa error di atas.
    pause
    exit /b 1
)

echo ✅ Build berhasil! Binary dibuat: print-service.exe

:: Pastikan folder template lokal ada
if not exist "template" (
    mkdir "template"
)

:: Jika pilihan Pisah/Eksternal, pastikan template JSON ada di folder template lokal
echo [2/3] Memeriksa folder template...
if "!CHOICE_BUNDLE!"=="2" (
    echo      Mode Pisah: Memastikan semua template JSON siap di folder template\
    xcopy /Y /Q "template\*.json" "template\" >nul 2>&1
)

:: ------------------------------------------------------------
:: Sinkronisasi ke Electron Sidecar
:: ------------------------------------------------------------
if "!CHOICE_SYNC!"=="1" (
    echo [3/3] Menyinkronkan ke sidecar Electron...
    set "SIDECAR_DIR=..\Pengebalan\knitto-manpro-electron\sidecar"
    
    if not exist "!SIDECAR_DIR!" (
        mkdir "!SIDECAR_DIR!"
    )
    
    copy /Y "print-service.exe" "!SIDECAR_DIR!\print-service.exe" >nul
    copy /Y "config.json" "!SIDECAR_DIR!\config.json" >nul 2>&1
    
    :: Selalu salin folder template ke sidecar jika ada
    if not exist "!SIDECAR_DIR!\template" (
        mkdir "!SIDECAR_DIR!\template"
    )
    xcopy /Y /Q /E "template\*.json" "!SIDECAR_DIR!\template\" >nul 2>&1
    
    echo ✅ Berhasil disalin ke sidecar Electron:
    echo    - !SIDECAR_DIR!\print-service.exe
    echo    - !SIDECAR_DIR!\config.json
    echo    - !SIDECAR_DIR!\template\ (*.json)
) else (
    echo [3/3] Sinkronisasi sidecar dilewati.
)

:: ------------------------------------------------------------
:: Siapkan Folder Distribusi build/ Siap Kirim User
:: ------------------------------------------------------------
echo.
echo [4/4] Menyiapkan paket rilis di folder 'build\'...
if not exist "build" (
    mkdir "build"
)
copy /Y "print-service.exe" "build\print-service.exe" >nul
copy /Y "config.json" "build\config.json" >nul 2>&1
copy /Y "PANDUAN_USER.txt" "build\PANDUAN_USER.txt" >nul 2>&1

if not exist "build\template" (
    mkdir "build\template"
)
xcopy /Y /Q /E "template\*.json" "build\template\" >nul 2>&1

echo ✅ Paket rilis berhasil disiapkan di folder 'build\':
echo    - build\print-service.exe
echo    - build\config.json
echo    - build\PANDUAN_USER.txt
echo    - build\template\ (*.json)

echo.
echo ============================================================
echo 🎉 PROSES BUILD SELESAI DENGAN SUKSES!
echo    Mode Template : !MODE_LABEL!
if "!CHOICE_TRAY!"=="2" (
echo    Mode Tray     : Headless (-tray=false saat dijalankan)
) else (
echo    Mode Tray     : System Tray Aktif (Default)
)
echo    Folder Rilis  : build\ (Tinggal kirim folder ini atau .zip ke user)
echo ============================================================
echo.
pause

