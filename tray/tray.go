package tray

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	modShell32 = syscall.NewLazyDLL("shell32.dll")
	modUser32  = syscall.NewLazyDLL("user32.dll")
	modKernel  = syscall.NewLazyDLL("kernel32.dll")

	procShell_NotifyIconW   = modShell32.NewProc("Shell_NotifyIconW")
	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procPostQuitMessage     = modUser32.NewProc("PostQuitMessage")
	procGetMessageW         = modUser32.NewProc("GetMessageW")
	procTranslateMessage    = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = modUser32.NewProc("DispatchMessageW")
	procPostMessageW        = modUser32.NewProc("PostMessageW")
	procLoadIconW           = modUser32.NewProc("LoadIconW")
	procCreatePopupMenu     = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW         = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = modUser32.NewProc("TrackPopupMenu")
	procDestroyMenu         = modUser32.NewProc("DestroyMenu")
	procGetCursorPos        = modUser32.NewProc("GetCursorPos")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procShowWindow          = modUser32.NewProc("ShowWindow")
	procIsWindowVisible     = modUser32.NewProc("IsWindowVisible")

	procGetModuleHandleW = modKernel.NewProc("GetModuleHandleW")
	procGetConsoleWindow = modKernel.NewProc("GetConsoleWindow")
)

const (
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	NIIF_INFO = 0x00000001

	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_COMMAND       = 0x0111
	WM_LBUTTONUP     = 0x0202
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDBLCLK = 0x0203
	WM_APP           = 0x8000
	WM_TRAYCALLBACK  = WM_APP + 101

	IDI_APPLICATION = 32512

	MF_STRING    = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002
	MF_SEPARATOR = 0x00000800

	TPM_RIGHTBUTTON = 0x0002

	SW_HIDE = 0
	SW_SHOW = 5

	ID_MENU_GUI            = 2000
	ID_MENU_HEADER_STATUS  = 2001
	ID_MENU_HEADER_PRINTER = 2002
	ID_MENU_BROWSER        = 2003
	ID_MENU_OPEN_TEMPLATES = 2004
	ID_MENU_TOGGLE_CONSOLE = 2005
	ID_MENU_ADD_TEMPLATE   = 2006
	ID_MENU_EXIT           = 2007
	ID_MENU_PREVIEW        = 2008
)

type NOTIFYICONDATAW struct {
	CbSize            uint32
	_                 uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	_                 uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	_                 uint32
	GuidItem          [16]byte
	HBalloonIcon      uintptr
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type TrayApp struct {
	Port         int
	PrinterName  string
	Hwnd         uintptr
	ConsoleHwnd  uintptr
	Nid          NOTIFYICONDATAW
	OnExit       func()
	ConsoleShown bool
	Gui          *NativeControlPanel
	Callbacks    GuiCallbacks
	mu           sync.Mutex
	stopChan     chan struct{}
}

var (
	trayClassOnce sync.Once
	trayClassName *uint16
	activeTray    *TrayApp
	activeTrayMu  sync.RWMutex
)

func registerTrayClass() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	trayClassName, _ = syscall.UTF16PtrFromString("PrintServiceTrayWindowClass")

	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		activeTrayMu.RLock()
		app := activeTray
		activeTrayMu.RUnlock()

		switch uint32(msg) {
		case WM_TRAYCALLBACK:
			mouseMsg := lParam & 0xFFFF
			switch mouseMsg {
			case WM_RBUTTONUP:
				if app != nil {
					app.showContextMenu()
				}
			case WM_LBUTTONDBLCLK, WM_LBUTTONUP:
				if app != nil {
					app.ShowControlPanel()
				}
			}
			return 0

		case WM_COMMAND:
			if app != nil {
				cmdId := wParam & 0xFFFF
				switch cmdId {
				case ID_MENU_GUI:
					app.ShowControlPanel()
				case ID_MENU_PREVIEW:
					if app.Callbacks.OnOpenPreview != nil {
						go app.Callbacks.OnOpenPreview()
					}
				case ID_MENU_BROWSER:
					app.openBrowser()
				case ID_MENU_OPEN_TEMPLATES:
					app.openTemplates()
				case ID_MENU_ADD_TEMPLATE:
					if app.Gui != nil {
						app.Gui.onAddTemplateClicked()
					}
				case ID_MENU_TOGGLE_CONSOLE:
					app.toggleConsole()
				case ID_MENU_EXIT:
					if app.OnExit != nil {
						go app.OnExit()
					}
				}
			}
			return 0

		case WM_CLOSE:
			procDestroyWindow.Call(hwnd)
			return 0

		case WM_DESTROY:
			procPostQuitMessage.Call(0)
			return 0
		}

		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	})

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   wndProc,
		HInstance:     hInst,
		LpszClassName: trayClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

// StartTray initializes the System Tray icon, Native GUI Window, and message loop
func StartTray(port int, defaultPrinter string, cb GuiCallbacks, onExit func()) (*TrayApp, error) {
	consoleHwnd, _, _ := procGetConsoleWindow.Call()
	consoleShown := true
	if consoleHwnd != 0 {
		vis, _, _ := procIsWindowVisible.Call(consoleHwnd)
		consoleShown = (vis != 0)
	}

	app := &TrayApp{
		Port:         port,
		PrinterName:  defaultPrinter,
		ConsoleHwnd:  consoleHwnd,
		OnExit:       onExit,
		ConsoleShown: consoleShown,
		Callbacks:    cb,
		stopChan:     make(chan struct{}),
	}

	activeTrayMu.Lock()
	activeTray = app
	activeTrayMu.Unlock()

	readyChan := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Pastikan thread terhubung ke desktop interaktif pengguna "Default" (agar icon muncul di taskbar fisik pengguna)
		procOpenDesktopW := modUser32.NewProc("OpenDesktopW")
		procSetThreadDesktop := modUser32.NewProc("SetThreadDesktop")
		defDeskName, _ := syscall.UTF16PtrFromString("Default")
		hDesk, _, _ := procOpenDesktopW.Call(
			uintptr(unsafe.Pointer(defDeskName)),
			0,
			0,
			uintptr(0x01FF), // DESKTOP_ALL_ACCESS
		)
		if hDesk != 0 {
			procSetThreadDesktop.Call(hDesk)
		}

		trayClassOnce.Do(registerTrayClass)

		hInst, _, _ := procGetModuleHandleW.Call(0)
		hwnd, _, err := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(trayClassName)), 0,
			0, 0, 0, 0, 0, 0, 0, hInst, 0,
		)
		if hwnd == 0 {
			readyChan <- fmt.Errorf("gagal membuat tray message window: %v", err)
			return
		}
		app.Hwnd = hwnd

		// Siapkan Tray Icon dari custom apple-touch-icon.png
		hIcon, err := LoadDefaultIcon()
		if err != nil || hIcon == 0 {
			hIcon, _, _ = procLoadIconW.Call(0, uintptr(IDI_APPLICATION))
		}

		app.Nid.CbSize = uint32(unsafe.Sizeof(app.Nid))
		app.Nid.HWnd = hwnd
		app.Nid.UID = 1
		app.Nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP | NIF_INFO
		app.Nid.UCallbackMessage = WM_TRAYCALLBACK
		app.Nid.HIcon = hIcon

		tip := fmt.Sprintf("Print Service (Online :%d)", port)
		copy(app.Nid.SzTip[:], syscall.StringToUTF16(tip))

		// Balloon Notification saat service aktif
		copy(app.Nid.SzInfoTitle[:], syscall.StringToUTF16("Print Service Online"))
		balloonMsg := fmt.Sprintf("Service berjalan di http://localhost:%d\nPrinter: %s\nKlik icon untuk membuka Control Panel", port, defaultPrinter)
		copy(app.Nid.SzInfo[:], syscall.StringToUTF16(balloonMsg))
		app.Nid.DwInfoFlags = NIIF_INFO

		r1, _, _ := procShell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&app.Nid)))
		if r1 == 0 {
			// Fallback: coba tanpa NIF_INFO jika versi Windows lama
			app.Nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
			procShell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&app.Nid)))
		}

		// Siapkan Native Delphi-style Control Panel Window pada OS thread yang sama
		gui, errGui := NewNativeControlPanel(port, defaultPrinter, cb)
		if errGui == nil {
			app.Gui = gui
		}

		readyChan <- nil

		// Message loop (melayani tray window dan native GUI window)
		var msg MSG
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}

		// Hapus icon tray saat keluar
		procShell_NotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&app.Nid)))
		if app.Nid.HIcon != 0 {
			procDestroyIcon.Call(app.Nid.HIcon)
		}
	}()

	if err := <-readyChan; err != nil {
		return nil, err
	}

	return app, nil
}

// ShowControlPanel menampilkan jendela Native Control Panel Windows
func (t *TrayApp) ShowControlPanel() {
	if t.Gui != nil {
		t.Gui.Show()
	}
}

// HideControlPanel menyembunyikan jendela Native Control Panel
func (t *TrayApp) HideControlPanel() {
	if t.Gui != nil {
		t.Gui.Hide()
	}
}

// UpdatePrinter memperbarui informasi printer default di tooltip dan di Native GUI
func (t *TrayApp) UpdatePrinter(printerName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.PrinterName = printerName
	tip := fmt.Sprintf("Print Service (:%d) - %s", t.Port, printerName)
	if len(tip) > 120 {
		tip = tip[:120]
	}
	copy(t.Nid.SzTip[:], syscall.StringToUTF16(tip))
	procShell_NotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.Nid)))

	if t.Gui != nil {
		t.Gui.UpdatePrinter(printerName)
	}
}

// UpdatePort memperbarui informasi port di tooltip tray dan di Native GUI
func (t *TrayApp) UpdatePort(newPort int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Port = newPort
	tip := fmt.Sprintf("Print Service (Online :%d) - %s", newPort, t.PrinterName)
	if len(tip) > 120 {
		tip = tip[:120]
	}
	copy(t.Nid.SzTip[:], syscall.StringToUTF16(tip))
	procShell_NotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.Nid)))

	if t.Gui != nil {
		t.Gui.UpdatePort(newPort)
	}
}

// Stop menghapus icon dari tray dan menghentikan message loop
func (t *TrayApp) Stop() {
	if t.Hwnd != 0 {
		procPostMessageW.Call(t.Hwnd, WM_CLOSE, 0, 0)
	}
}

func (t *TrayApp) showContextMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	statusStr := fmt.Sprintf("Print Service (Online :%d)", t.Port)
	statusW, _ := syscall.UTF16PtrFromString(statusStr)
	procAppendMenuW.Call(hMenu, uintptr(MF_DISABLED|MF_GRAYED), uintptr(ID_MENU_HEADER_STATUS), uintptr(unsafe.Pointer(statusW)))

	prnStr := "Printer: " + t.PrinterName
	if len(prnStr) > 36 {
		prnStr = prnStr[:33] + "..."
	}
	prnW, _ := syscall.UTF16PtrFromString(prnStr)
	procAppendMenuW.Call(hMenu, uintptr(MF_DISABLED|MF_GRAYED), uintptr(ID_MENU_HEADER_PRINTER), uintptr(unsafe.Pointer(prnW)))

	// Separator
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)

	// Delphi-style Native Control Panel
	guiW, _ := syscall.UTF16PtrFromString("🖥️ Buka Control Panel (Windows)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_GUI), uintptr(unsafe.Pointer(guiW)))

	// FastReport Native Windows Preview
	prevW, _ := syscall.UTF16PtrFromString("👁️ FastReport Print Preview (Windows)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_PREVIEW), uintptr(unsafe.Pointer(prevW)))

	// Web Dashboard link
	browserW, _ := syscall.UTF16PtrFromString("🌐 Buka Web Dashboard (Browser)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_BROWSER), uintptr(unsafe.Pointer(browserW)))

	// Buka Folder Template
	tmplW, _ := syscall.UTF16PtrFromString("📂 Buka Folder Template")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_OPEN_TEMPLATES), uintptr(unsafe.Pointer(tmplW)))

	// Tambah / Upload Template
	addTmplW, _ := syscall.UTF16PtrFromString("➕ Tambah / Upload Template (.json)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_ADD_TEMPLATE), uintptr(unsafe.Pointer(addTmplW)))

	// Separator
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)

	// Toggle console CMD
	consoleText := "Sembunyikan Terminal"
	if !t.ConsoleShown {
		consoleText = "Tampilkan Terminal"
	}
	consoleW, _ := syscall.UTF16PtrFromString(consoleText)
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_TOGGLE_CONSOLE), uintptr(unsafe.Pointer(consoleW)))

	// Separator
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)

	// Exit
	exitW, _ := syscall.UTF16PtrFromString("Keluar / Stop Service")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_MENU_EXIT), uintptr(unsafe.Pointer(exitW)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(t.Hwnd)
	procTrackPopupMenu.Call(hMenu, uintptr(TPM_RIGHTBUTTON), uintptr(pt.X), uintptr(pt.Y), 0, t.Hwnd, 0)
	procPostMessageW.Call(t.Hwnd, 0, 0, 0)
}

func (t *TrayApp) toggleConsole() {
	if t.ConsoleHwnd == 0 {
		t.ConsoleHwnd, _, _ = procGetConsoleWindow.Call()
	}
	if t.ConsoleHwnd == 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ConsoleShown {
		procShowWindow.Call(t.ConsoleHwnd, uintptr(SW_HIDE))
		t.ConsoleShown = false
	} else {
		procShowWindow.Call(t.ConsoleHwnd, uintptr(SW_SHOW))
		t.ConsoleShown = true
	}
}

// ToggleConsole toggles visibility of the console window
func (t *TrayApp) ToggleConsole() {
	t.toggleConsole()
}

// HideConsole hides the console window
func (t *TrayApp) HideConsole() {
	if t.ConsoleHwnd == 0 {
		t.ConsoleHwnd, _, _ = procGetConsoleWindow.Call()
	}
	if t.ConsoleHwnd == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	procShowWindow.Call(t.ConsoleHwnd, uintptr(SW_HIDE))
	t.ConsoleShown = false
}

// ShowConsole shows the console window
func (t *TrayApp) ShowConsole() {
	if t.ConsoleHwnd == 0 {
		t.ConsoleHwnd, _, _ = procGetConsoleWindow.Call()
	}
	if t.ConsoleHwnd == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	procShowWindow.Call(t.ConsoleHwnd, uintptr(SW_SHOW))
	t.ConsoleShown = true
}

func (t *TrayApp) openBrowser() {
	url := fmt.Sprintf("http://localhost:%d/dashboard", t.Port)
	_ = exec.Command("cmd", "/c", "start", url).Start()
}

func (t *TrayApp) openTemplates() {
	if t.Callbacks.OnOpenTemplateFolder != nil {
		t.Callbacks.OnOpenTemplateFolder()
	}
}

