param (
    [ValidateSet("Tray", "Silent", "Electron", "Custom", "")]
    [string]$Target = "",

    [ValidateSet("FullBundle", "External")]
    [string]$Mode = "FullBundle",

    [ValidateSet("Tray", "SilentDaemon", "Console", "")]
    [string]$Runtime = "",

    [switch]$SkipSync
)

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  KNITTO PRINT SERVICE - BUILD GENERATOR (POWERSHELL)" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# Menu interaktif jika Target dan Runtime tidak ditentukan lewat parameter CLI
if ([string]::IsNullOrWhiteSpace($Target) -and [string]::IsNullOrWhiteSpace($Runtime)) {
    Write-Host ""
    Write-Host "PILIH TARGET SISTEM YANG INGIN DI-BUILD:" -ForegroundColor Yellow
    Write-Host "  [1] Standalone: System Tray + Control Panel (Default)" -ForegroundColor White
    Write-Host "      -> Ada icon taskbar tray, form Control Panel Windows" -ForegroundColor Gray
    Write-Host "      -> Otomatis auto-start saat boot, tanpa CMD hitam" -ForegroundColor Gray
    Write-Host "      -> Siap dikemas ke Knitto-Print-Service-v1.0.zip" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  [2] Standalone: Silent Daemon (100% Senyap Tanpa Tray)" -ForegroundColor White
    Write-Host "      -> Tanpa icon tray dan tanpa CMD hitam (seperti service murni)" -ForegroundColor Gray
    Write-Host "      -> Otomatis auto-start saat boot, kelola via Web Dashboard" -ForegroundColor Gray
    Write-Host "      -> Siap dikemas ke Knitto-Print-Service-v1.0.zip" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  [3] Electron Sidecar (Untuk Aplikasi Pengebalan Desktop)" -ForegroundColor White
    Write-Host "      -> Otomatis dikompilasi & disalin ke folder sidecar/ Electron" -ForegroundColor Gray
    Write-Host "      -> Berjalan otomatis di balik layar saat Electron dibuka" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  [4] Mode Custom / Lanjutan" -ForegroundColor White
    Write-Host "      -> Pilih manual mode template (Bundle/Pisah) & runtime" -ForegroundColor Gray
    Write-Host ""
    $choice = Read-Host "Pilihan Anda (1/2/3/4) [Default 1]"
    if ([string]::IsNullOrWhiteSpace($choice)) { $choice = "1" }

    switch ($choice) {
        "1" { $Target = "Tray" }
        "2" { $Target = "Silent" }
        "3" { $Target = "Electron" }
        "4" { $Target = "Custom" }
        default { $Target = "Tray" }
    }
}

# Terjemahkan pilihan Target ke Mode dan Runtime
if ($Target -eq "Tray") {
    $Mode = "FullBundle"
    $Runtime = "Tray"
} elseif ($Target -eq "Silent") {
    $Mode = "FullBundle"
    $Runtime = "SilentDaemon"
} elseif ($Target -eq "Electron") {
    $Mode = "FullBundle"
    $Runtime = "SilentDaemon"
} elseif ($Target -eq "Custom" -and [string]::IsNullOrWhiteSpace($Runtime)) {
    Write-Host ""
    Write-Host "PILIHAN MODE RUNTIME:" -ForegroundColor Yellow
    Write-Host "  [1] System Tray Aktif" -ForegroundColor White
    Write-Host "  [2] Silent Daemon (Tanpa Tray)" -ForegroundColor White
    Write-Host "  [3] Console / CLI (Jendela Hitam CMD)" -ForegroundColor White
    $cRun = Read-Host "Pilih (1/2/3) [Default 1]"
    switch ($cRun) {
        "2" { $Runtime = "SilentDaemon" }
        "3" { $Runtime = "Console" }
        default { $Runtime = "Tray" }
    }
}

if ([string]::IsNullOrWhiteSpace($Runtime)) {
    $Runtime = "Tray"
}

# 1. Tentukan tag build
$buildTags = ""
$modeLabel = "Full Bundle (Embedded Templates)"

if ($Mode -eq "External") {
    $buildTags = "-tags nobundle"
    $modeLabel = "Pisah / External Templates (No-Bundle)"
}

# 2. Tentukan mode runtime & ldflags
$ldflags = "-s -w -H=windowsgui -X main.DefaultTray=true"
$runtimeLabel = "System Tray Aktif (Windows GUI)"

if ($Runtime -eq "SilentDaemon") {
    $ldflags = "-s -w -H=windowsgui -X main.DefaultTray=false"
    $runtimeLabel = "Silent Daemon (100% Senyap, Tanpa Tray, Tanpa CMD)"
} elseif ($Runtime -eq "Console") {
    $ldflags = "-s -w -X main.DefaultTray=false"
    $runtimeLabel = "Console / CLI (Jendela Hitam CMD untuk Debugging)"
}

Write-Host ""
Write-Host "Mode Template : $modeLabel" -ForegroundColor Yellow
Write-Host "Mode Runtime  : $runtimeLabel" -ForegroundColor Yellow

# 3. Build Go binary
Write-Host ""
Write-Host "[1/3] Mengompilasi executable..." -ForegroundColor Green

if ($Mode -eq "External") {
    go build -tags nobundle -ldflags="$ldflags" -o print-service.exe .
} else {
    go build -ldflags="$ldflags" -o print-service.exe .
}

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Build gagal dengan kode error: $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}

$fileInfo = Get-Item "print-service.exe"
$sizeMb = [math]::Round($fileInfo.Length / 1MB, 2)
$fileBytes = $fileInfo.Length
Write-Host "Kompilasi sukses! Ukuran binary: $sizeMb MB ($fileBytes bytes)" -ForegroundColor Green

# 4. Pastikan folder template lokal ada
Write-Host ""
Write-Host "[2/3] Memeriksa folder template..." -ForegroundColor Green
if (!(Test-Path "template")) {
    New-Item -ItemType Directory -Path "template" | Out-Null
}

# 5. Sinkronisasi ke Electron Sidecar
if (!$SkipSync) {
    Write-Host ""
    Write-Host "[3/3] Menyinkronkan ke folder sidecar Electron..." -ForegroundColor Green
    $sidecarDir = Resolve-Path "..\Pengebalan\knitto-manpro-electron\sidecar" -ErrorAction SilentlyContinue

    if (!$sidecarDir) {
        $targetPath = Join-Path (Get-Location).Path "..\Pengebalan\knitto-manpro-electron\sidecar"
        New-Item -ItemType Directory -Path $targetPath -Force | Out-Null
        $sidecarDir = Resolve-Path $targetPath
    }

    Copy-Item "print-service.exe" -Destination (Join-Path $sidecarDir "print-service.exe") -Force
    if (Test-Path "config.json") {
        Copy-Item "config.json" -Destination (Join-Path $sidecarDir "config.json") -Force
    }

    $sidecarTmpl = Join-Path $sidecarDir "template"
    if (!(Test-Path $sidecarTmpl)) {
        New-Item -ItemType Directory -Path $sidecarTmpl -Force | Out-Null
    }
    Copy-Item "template\*.json" -Destination $sidecarTmpl -Force

    Write-Host "Berhasil disinkronkan ke: $sidecarDir" -ForegroundColor Green
    Write-Host "   - print-service.exe"
    Write-Host "   - config.json"
    Write-Host "   - template\ (*.json)"
} else {
    Write-Host ""
    Write-Host "[3/3] Sinkronisasi sidecar dilewati (-SkipSync)." -ForegroundColor Gray
}

# 6. Siapkan folder distribusi build/ untuk dikirim ke user
Write-Host ""
Write-Host "[4/4] Menyiapkan paket rilis di folder 'build\'..." -ForegroundColor Green
$distDir = Join-Path (Get-Location).Path "build"
if (!(Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir -Force | Out-Null
}

Copy-Item "print-service.exe" -Destination (Join-Path $distDir "print-service.exe") -Force
if (Test-Path "config.json") {
    Copy-Item "config.json" -Destination (Join-Path $distDir "config.json") -Force
}
if (Test-Path "PANDUAN_USER.txt") {
    Copy-Item "PANDUAN_USER.txt" -Destination (Join-Path $distDir "PANDUAN_USER.txt") -Force
}

$distTmpl = Join-Path $distDir "template"
if (!(Test-Path $distTmpl)) {
    New-Item -ItemType Directory -Path $distTmpl -Force | Out-Null
}
Copy-Item "template\*.json" -Destination $distTmpl -Force

# Buat file ZIP siap kirim
$zipFile = Join-Path $distDir "Knitto-Print-Service-v1.0.zip"
if (Test-Path $zipFile) {
    Remove-Item $zipFile -Force
}
Start-Sleep -Milliseconds 600
Compress-Archive -Path "$distDir\print-service.exe", "$distDir\config.json", "$distDir\PANDUAN_USER.txt", "$distDir\template" -DestinationPath $zipFile -Force

# Hapus binary sementara di root folder agar git tetap bersih
Remove-Item "print-service.exe" -Force -ErrorAction SilentlyContinue

Write-Host "Paket rilis berhasil disiapkan di: $distDir" -ForegroundColor Green
Write-Host "   - print-service.exe ($modeLabel, Auto-Install)"
Write-Host "   - config.json"
Write-Host "   - PANDUAN_USER.txt"
Write-Host "   - template\ (*.json)"
Write-Host "   - Knitto-Print-Service-v1.0.zip (Siap kirim WA/Drive)" -ForegroundColor Yellow

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "SELESAI! Paket rilis siap dipakai di: $distDir" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
