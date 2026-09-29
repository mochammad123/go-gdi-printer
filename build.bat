@echo off
setlocal enabledelayedexpansion
title Knitto Print Service - Build Generator

echo ============================================================
echo   KNITTO PRINT SERVICE - BUILD GENERATOR (GOLANG)
echo ============================================================
echo.
echo PILIH TARGET SISTEM YANG INGIN DI-BUILD:
echo.
echo   [1] Standalone: System Tray + Control Panel (Default)
echo       -^> Ada icon taskbar tray, form Control Panel Windows
echo       -^> Otomatis auto-start saat boot, tanpa CMD hitam
echo       -^> Siap dikemas ke Knitto-Print-Service-v1.0.zip
echo.
echo   [2] Standalone: Silent Daemon (100%% Senyap Tanpa Tray)
echo       -^> Tanpa icon tray dan tanpa CMD hitam (seperti service murni)
echo       -^> Otomatis auto-start saat boot, kelola via Web Dashboard
echo       -^> Siap dikemas ke Knitto-Print-Service-v1.0.zip
echo.
echo   [3] Electron Sidecar (Untuk Aplikasi Pengebalan Desktop)
echo       -^> Otomatis dikompilasi ^& disalin ke folder sidecar/ Electron
echo       -^> Berjalan otomatis di balik layar saat Electron dibuka
echo.
echo   [4] Mode Custom / Lanjutan
echo       -^> Pilih manual mode template (Bundle/Pisah) ^& runtime
echo.
set /p "CHOICE_SYSTEM=Pilihan Anda (1/2/3/4) [Default 1]: "
if "!CHOICE_SYSTEM!"=="" set "CHOICE_SYSTEM=1"

set "CHOICE_BUNDLE=1"
set "CHOICE_TRAY=1"
set "CHOICE_SYNC=1"

if "!CHOICE_SYSTEM!"=="1" (
    set "CHOICE_BUNDLE=1"
    set "CHOICE_TRAY=1"
    set "CHOICE_SYNC=1"
)
if "!CHOICE_SYSTEM!"=="2" (
    set "CHOICE_BUNDLE=1"
    set "CHOICE_TRAY=2"
    set "CHOICE_SYNC=1"
)
if "!CHOICE_SYSTEM!"=="3" (
    set "CHOICE_BUNDLE=1"
    set "CHOICE_TRAY=2"
    set "CHOICE_SYNC=1"
)
if "!CHOICE_SYSTEM!"=="4" (
    echo.
    echo PILIHAN TEMPLATE:
    echo   [1] Full Bundle  : Semua template JSON di-embed ke dalam binary
    echo   [2] Pisah        : File .exe berdiri sendiri + folder template/ di sampingnya
    set /p "CHOICE_BUNDLE=Pilih (1/2) [Default 1]: "
    if "!CHOICE_BUNDLE!"=="" set "CHOICE_BUNDLE=1"

    echo.
    echo PILIHAN RUNTIME:
    echo   [1] System Tray Aktif (Windows GUI)
    echo   [2] Silent Daemon (Tanpa Tray, Tanpa CMD)
    echo   [3] Console / CLI (Jendela Hitam CMD)
    set /p "CHOICE_TRAY=Pilih (1/2/3) [Default 1]: "
    if "!CHOICE_TRAY!"=="" set "CHOICE_TRAY=1"

    echo.
    echo SINKRONISASI KE SIDECAR ELECTRON:
    echo   [1] Ya
    echo   [2] Tidak
    set /p "CHOICE_SYNC=Pilih (1/2) [Default 1]: "
    if "!CHOICE_SYNC!"=="" set "CHOICE_SYNC=1"
)

echo.
echo ============================================================
echo  MEMULAI PROSES BUILD...
echo ============================================================

set "BUILD_TAGS="
set "MODE_LABEL=Full Bundle (Embedded Templates)"

if "!CHOICE_BUNDLE!"=="2" (
    set "BUILD_TAGS=-tags nobundle"
    set "MODE_LABEL=Pisah / External Templates (No-Bundle)"
)

set "LDFLAGS=-s -w -H=windowsgui -X main.DefaultTray=true"
set "TRAY_LABEL=System Tray Aktif"

if "!CHOICE_TRAY!"=="2" (
    set "LDFLAGS=-s -w -H=windowsgui -X main.DefaultTray=false"
    set "TRAY_LABEL=Silent Daemon (100%% Senyap, Tanpa Tray, Tanpa CMD)"
)
if "!CHOICE_TRAY!"=="3" (
    set "LDFLAGS=-s -w -X main.DefaultTray=false"
    set "TRAY_LABEL=Console / CLI (Jendela Hitam CMD)"
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

if not exist "template" (
    mkdir "template"
)

echo [2/3] Memeriksa folder template...
if "!CHOICE_BUNDLE!"=="2" (
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

:: Buat file ZIP otomatis
powershell -Command "Start-Sleep -Milliseconds 500; Compress-Archive -Path 'build\print-service.exe', 'build\config.json', 'build\PANDUAN_USER.txt', 'build\template' -DestinationPath 'build\Knitto-Print-Service-v1.0.zip' -Force" >nul 2>&1

:: Hapus binary sementara di root agar workspace bersih
del /f /q "print-service.exe" >nul 2>&1

echo ✅ Paket rilis berhasil disiapkan di folder 'build\':
echo    - build\print-service.exe
echo    - build\config.json
echo    - build\PANDUAN_USER.txt
echo    - build\template\ (*.json)
echo    - build\Knitto-Print-Service-v1.0.zip (Siap kirim WA/Drive)

echo.
echo ============================================================
echo 🎉 PROSES BUILD SELESAI DENGAN SUKSES!
echo    Mode Template : !MODE_LABEL!
echo    Mode Runtime  : !TRAY_LABEL!
echo    Folder Rilis  : build\ (Tinggal kirim folder ini atau .zip ke user)
echo ============================================================
echo.
pause
