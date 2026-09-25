package winapi

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	ProcMessageBoxW = ModUser32.NewProc("MessageBoxW")
)

const (
	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONERROR       = 0x00000010
)

// GetPermanentAppDir returns %LOCALAPPDATA%\Knitto\PrintService
func GetPermanentAppDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			localAppData = filepath.Join(userProfile, "AppData", "Local")
		} else {
			localAppData = "C:\\Knitto"
		}
	}
	return filepath.Join(localAppData, "Knitto", "PrintService")
}

// IsRunningFromPermanentDir checks if the current process is running from permanent dir or sidecar
func IsRunningFromPermanentDir() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	cleanExe := strings.ToLower(filepath.Clean(exe))
	cleanTargetDir := strings.ToLower(filepath.Clean(GetPermanentAppDir()))

	// If inside permanent dir or inside sidecar dir, consider it already permanent
	if strings.HasPrefix(cleanExe, cleanTargetDir) || strings.Contains(cleanExe, "sidecar") {
		return true
	}
	return false
}

// CreateWindowsShortcut creates a .lnk file using PowerShell WScript.Shell
func CreateWindowsShortcut(targetExe, shortcutPath, description string) error {
	dir := filepath.Dir(shortcutPath)
	_ = os.MkdirAll(dir, 0755)

	targetDir := filepath.Dir(targetExe)
	psCmd := fmt.Sprintf(`& { $ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.WorkingDirectory = '%s'; $s.Description = '%s'; $s.Save() }`,
		strings.ReplaceAll(shortcutPath, "'", "''"),
		strings.ReplaceAll(targetExe, "'", "''"),
		strings.ReplaceAll(targetDir, "'", "''"),
		strings.ReplaceAll(description, "'", "''"),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psCmd)
	return cmd.Run()
}

// DeployToPermanentDirectory copies files to %LOCALAPPDATA%\Knitto\PrintService, sets up shortcuts & registry
func DeployToPermanentDirectory() (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("gagal mendapatkan path executable saat ini: %w", err)
	}

	targetDir := GetPermanentAppDir()
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat folder instalasi '%s': %w", targetDir, err)
	}

	targetExe := filepath.Join(targetDir, "print-service.exe")

	// 1. Salin print-service.exe
	if !strings.EqualFold(filepath.Clean(currentExe), filepath.Clean(targetExe)) {
		if err := copyFile(currentExe, targetExe); err != nil {
			return "", fmt.Errorf("gagal menyalin executable ke '%s': %w", targetExe, err)
		}
	}

	currentDir := filepath.Dir(currentExe)

	// 2. Salin config.json jika ada, atau buat default
	srcCfg := filepath.Join(currentDir, "config.json")
	targetCfg := filepath.Join(targetDir, "config.json")
	if _, err := os.Stat(srcCfg); err == nil {
		_ = copyFile(srcCfg, targetCfg)
	} else if _, err := os.Stat(targetCfg); os.IsNotExist(err) {
		defaultCfg := []byte("{\n  \"port\": 8080,\n  \"default_printer\": \"\"\n}\n")
		_ = os.WriteFile(targetCfg, defaultCfg, 0644)
	}

	// 3. Pastikan folder template di target ada dan salin template yang ada
	targetTmplDir := filepath.Join(targetDir, "template")
	_ = os.MkdirAll(targetTmplDir, 0755)

	srcTmplDir := filepath.Join(currentDir, "template")
	if entries, err := os.ReadDir(srcTmplDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				srcFile := filepath.Join(srcTmplDir, e.Name())
				destFile := filepath.Join(targetTmplDir, e.Name())
				_ = copyFile(srcFile, destFile)
			}
		}
	}

	// 5. Buat Shortcut di Desktop
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		desktopPath := filepath.Join(userProfile, "Desktop", "Knitto Print Service.lnk")
		_ = CreateWindowsShortcut(targetExe, desktopPath, "Knitto Print Service")
	}

	// 6. Buat Shortcut di Folder Startup (shell:startup)
	appData := os.Getenv("APPDATA")
	if appData != "" {
		startupPath := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Knitto Print Service.lnk")
		_ = CreateWindowsShortcut(targetExe, startupPath, "Knitto Print Service (Otomatis saat Booting)")

		startMenuPath := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Knitto Print Service.lnk")
		_ = CreateWindowsShortcut(targetExe, startMenuPath, "Knitto Print Service")
	}

	// 7. Daftarkan ke Registry Run Windows menggunakan targetExe permanen
	_ = SetAutoStartCustom(targetExe, true)

	return targetExe, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ShowDeploySuccessDialog displays a friendly Win32 notification dialog
func ShowDeploySuccessDialog(installedDir string) {
	titleW, _ := syscall.UTF16PtrFromString("Knitto Print Service - Berhasil Dipasang")
	msg := fmt.Sprintf("✅ Knitto Print Service berhasil dipasang di komputer Anda!\n\n"+
		"📁 Lokasi Master Penyimpanan:\n%s\n\n"+
		"• Shortcut telah dibuat di Desktop & Startup Windows.\n"+
		"• Layanan cetak otomatis aktif di System Tray taskbar.\n"+
		"• File installer di folder ini sekarang aman untuk Anda hapus.", installedDir)
	msgW, _ := syscall.UTF16PtrFromString(msg)

	ProcMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgW)),
		uintptr(unsafe.Pointer(titleW)),
		uintptr(MB_OK|MB_ICONINFORMATION),
	)
}

// EnsureShortcuts ensures Desktop and Startup shortcuts exist
func EnsureShortcuts() {
	targetExe := filepath.Join(GetPermanentAppDir(), "print-service.exe")
	if _, err := os.Stat(targetExe); err != nil {
		targetExe, _ = os.Executable()
	}

	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		desktopPath := filepath.Join(userProfile, "Desktop", "Knitto Print Service.lnk")
		if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
			_ = CreateWindowsShortcut(targetExe, desktopPath, "Knitto Print Service")
		}
	}

	appData := os.Getenv("APPDATA")
	if appData != "" {
		startupPath := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Knitto Print Service.lnk")
		if _, err := os.Stat(startupPath); os.IsNotExist(err) {
			_ = CreateWindowsShortcut(targetExe, startupPath, "Knitto Print Service (Otomatis saat Booting)")
		}

		startMenuPath := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Knitto Print Service.lnk")
		if _, err := os.Stat(startMenuPath); os.IsNotExist(err) {
			_ = CreateWindowsShortcut(targetExe, startMenuPath, "Knitto Print Service")
		}
	}
}

