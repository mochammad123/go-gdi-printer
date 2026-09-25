param (
    [ValidateSet("FullBundle", "External")]
    [string]$Mode = "FullBundle",

    [switch]$Headless,
    [switch]$SkipSync
)

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  KNITTO PRINT SERVICE - BUILD GENERATOR (POWERSHELL)" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# 1. Tentukan tag build
$buildTags = ""
$modeLabel = "Full Bundle (Embedded Templates)"

if ($Mode -eq "External") {
    $buildTags = "-tags nobundle"
    $modeLabel = "Pisah / External Templates (No-Bundle)"
}

Write-Host ""
Write-Host "Mode Template : $modeLabel" -ForegroundColor Yellow
if ($Headless) {
    Write-Host "Mode Runtime  : Headless / CLI (-tray=false)" -ForegroundColor Yellow
} else {
    Write-Host "Mode Runtime  : System Tray Aktif (Windows GUI)" -ForegroundColor Yellow
}

# 2. Build Go binary
Write-Host ""
Write-Host "[1/3] Mengompilasi executable..." -ForegroundColor Green

$ldflags = if ($Headless) { "-s -w" } else { "-s -w -H=windowsgui" }

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

# 3. Pastikan folder template lokal ada
Write-Host ""
Write-Host "[2/3] Memeriksa folder template..." -ForegroundColor Green
if (!(Test-Path "template")) {
    New-Item -ItemType Directory -Path "template" | Out-Null
}

# 4. Sinkronisasi ke Electron Sidecar
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

# 5. Siapkan folder distribusi build/ untuk dikirim ke user
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

Write-Host "Paket rilis berhasil disiapkan di: $distDir" -ForegroundColor Green
Write-Host "   - print-service.exe (Full Bundle, Auto-Install)"
Write-Host "   - config.json"
Write-Host "   - PANDUAN_USER.txt"
Write-Host "   - template\ (*.json)"
Write-Host "   - Knitto-Print-Service-v1.0.zip (Siap kirim WA/Drive)" -ForegroundColor Yellow

$fullPath = $fileInfo.FullName
Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "SELESAI! Folder paket siap dikirim berada di:" -ForegroundColor Cyan
Write-Host "$distDir" -ForegroundColor Yellow
Write-Host "============================================================" -ForegroundColor Cyan

