package winapi

import (
	"runtime/debug"
	"syscall"
	"time"
)

var (
	ModPsapi                     = syscall.NewLazyDLL("psapi.dll")
	ProcEmptyWorkingSet          = ModPsapi.NewProc("EmptyWorkingSet")
	ProcSetProcessWorkingSetSize = ModKernel32.NewProc("SetProcessWorkingSetSize")
	ProcGetCurrentProcess        = ModKernel32.NewProc("GetCurrentProcess")
)

// TrimWorkingSet asks the Go runtime to release unmapped heap memory back to the OS
// and requests Windows to trim unreferenced pages from the process Working Set.
// This is the classic Delphi/Win32 background service optimization.
func TrimWorkingSet() {
	// 1. Kembalikan unused heap memory Go ke sistem operasi
	debug.FreeOSMemory()

	// 2. Minta Windows Memory Manager untuk membersihkan working set
	hProc, _, _ := ProcGetCurrentProcess.Call()
	if ProcEmptyWorkingSet.Find() == nil {
		ProcEmptyWorkingSet.Call(hProc)
	} else if ProcSetProcessWorkingSetSize.Find() == nil {
		ProcSetProcessWorkingSetSize.Call(hProc, ^uintptr(0), ^uintptr(0))
	}
}

// StartMemoryTrimmer runs a background ticker that trims working set periodically
// when the application is idle in the background.
func StartMemoryTrimmer(interval time.Duration) {
	go func() {
		// Tunggu sampai inisialisasi awal selesai
		time.Sleep(3 * time.Second)
		TrimWorkingSet()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			TrimWorkingSet()
		}
	}()
}
