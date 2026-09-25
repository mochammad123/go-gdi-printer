package winapi

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	ModAdvapi32          = syscall.NewLazyDLL("advapi32.dll")
	ProcRegOpenKeyExW     = ModAdvapi32.NewProc("RegOpenKeyExW")
	ProcRegSetValueExW    = ModAdvapi32.NewProc("RegSetValueExW")
	ProcRegDeleteValueW   = ModAdvapi32.NewProc("RegDeleteValueW")
	ProcRegQueryValueExW  = ModAdvapi32.NewProc("RegQueryValueExW")
	ProcRegCloseKey       = ModAdvapi32.NewProc("RegCloseKey")
	ProcAttachConsole     = ModKernel32.NewProc("AttachConsole")
	ProcGetStdHandle      = ModKernel32.NewProc("GetStdHandle")
)

const (
	HKEY_CURRENT_USER     = 0x80000001
	KEY_READ              = 0x20019
	KEY_WRITE             = 0x20006
	REG_SZ                = 1
	ATTACH_PARENT_PROCESS = ^uintptr(0) // -1
	AutoStartRegistryKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	AutoStartValueName    = "KnittoPrintService"
)

// AttachParentConsole attaches stdout/stderr to parent console or pipe if invoked from terminal (for -H=windowsgui)
func AttachParentConsole() bool {
	ProcAttachConsole.Call(ATTACH_PARENT_PROCESS)
	var stdOut int32 = syscall.STD_OUTPUT_HANDLE
	hOut, _, _ := ProcGetStdHandle.Call(uintptr(stdOut))
	if hOut != 0 && hOut != ^uintptr(0) {
		os.Stdout = os.NewFile(hOut, "/dev/stdout")
	}
	var stdErr int32 = syscall.STD_ERROR_HANDLE
	hErr, _, _ := ProcGetStdHandle.Call(uintptr(stdErr))
	if hErr != 0 && hErr != ^uintptr(0) {
		os.Stderr = os.NewFile(hErr, "/dev/stderr")
	}
	return true
}

// IsAutoStartEnabled checks if the service is set to run automatically on Windows startup
func IsAutoStartEnabled() bool {
	subKey, _ := syscall.UTF16PtrFromString(AutoStartRegistryKey)
	var hKey uintptr
	r1, _, _ := ProcRegOpenKeyExW.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(subKey)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return false
	}
	defer ProcRegCloseKey.Call(hKey)

	valName, _ := syscall.UTF16PtrFromString(AutoStartValueName)
	var valType uint32
	var bufSize uint32 = 1024
	buf := make([]uint16, 512)

	r1, _, _ = ProcRegQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valName)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufSize)),
	)
	return r1 == 0
}

// SetAutoStart adds or removes the executable from Windows Startup registry
func SetAutoStart(enable bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	return SetAutoStartCustom(exePath, enable)
}

// SetAutoStartCustom adds or removes a specific executable path from Windows Startup registry
func SetAutoStartCustom(exePath string, enable bool) error {
	subKey, _ := syscall.UTF16PtrFromString(AutoStartRegistryKey)
	var hKey uintptr
	r1, _, errSys := ProcRegOpenKeyExW.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(subKey)),
		0,
		KEY_WRITE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return fmt.Errorf("gagal membuka registry key Run: %v (code: %d)", errSys, r1)
	}
	defer ProcRegCloseKey.Call(hKey)

	valName, _ := syscall.UTF16PtrFromString(AutoStartValueName)

	if enable {
		valStr, _ := syscall.UTF16FromString(fmt.Sprintf("\"%s\"", exePath))
		r1, _, errSys = ProcRegSetValueExW.Call(
			hKey,
			uintptr(unsafe.Pointer(valName)),
			0,
			REG_SZ,
			uintptr(unsafe.Pointer(&valStr[0])),
			uintptr(len(valStr)*2),
		)
		if r1 != 0 {
			return fmt.Errorf("gagal mendaftarkan auto-start di registry: %v (code: %d)", errSys, r1)
		}
	} else {
		ProcRegDeleteValueW.Call(hKey, uintptr(unsafe.Pointer(valName)))
	}
	return nil
}
