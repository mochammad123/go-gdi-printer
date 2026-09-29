package tray

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"print-service/winapi"
)

var (
	procSendMessageW     = modUser32.NewProc("SendMessageW")
	procGetWindowTextW   = modUser32.NewProc("GetWindowTextW")
	procSetWindowTextW   = modUser32.NewProc("SetWindowTextW")
	procMessageBoxW      = modUser32.NewProc("MessageBoxW")
	procGetSystemMetrics = modUser32.NewProc("GetSystemMetrics")
	procFindWindowW      = modUser32.NewProc("FindWindowW")
	procInvalidateRect   = modUser32.NewProc("InvalidateRect")
	procUpdateWindow     = modUser32.NewProc("UpdateWindow")
	procCreateFontW      = modGdi32Icon.NewProc("CreateFontW")

	modOle32             = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize   = modOle32.NewProc("CoUninitialize")

	modComdlg32          = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileNameW = modComdlg32.NewProc("GetOpenFileNameW")
)

type OPENFILENAMEW struct {
	LStructSize       uint32
	_                 uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	_                 uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	_                 uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

const (
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_EXPLORER      = 0x00080000
)

const (
	WS_VISIBLE       = 0x10000000
	WS_CHILD         = 0x40000000
	WS_BORDER        = 0x00800000
	WS_TABSTOP       = 0x00010000
	WS_OVERLAPPED    = 0x00000000
	WS_CAPTION       = 0x00C00000
	WS_SYSMENU       = 0x00080000
	WS_MINIMIZEBOX   = 0x00020000
	WS_CLIPCHILDREN  = 0x02000000
	WS_CLIPSIBLINGS  = 0x04000000

	CBS_DROPDOWNLIST = 0x0003
	WS_VSCROLL       = 0x00200000
	CB_ADDSTRING     = 0x0143
	CB_SETCURSEL     = 0x014E
	CB_GETCURSEL     = 0x0147
	CB_GETLBTEXT     = 0x0148
	CB_RESETCONTENT  = 0x014B

	ES_LEFT   = 0x0000
	ES_NUMBER = 0x2000

	BS_PUSHBUTTON = 0x00000000
	BS_GROUPBOX   = 0x00000007
	SS_LEFT       = 0x00000000

	WM_SETFONT = 0x0030

	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONERROR       = 0x00000010
	MB_ICONWARNING     = 0x00000030

	COLOR_BTNFACE = 15

	FW_NORMAL = 400
	FW_BOLD   = 700

	ID_GUI_BTN_SET_PRINTER    = 4001
	ID_GUI_BTN_TEST_PRINT     = 4002
	ID_GUI_BTN_SAVE_PORT      = 4003
	ID_GUI_BTN_OPEN_WEB       = 4004
	ID_GUI_BTN_HIDE           = 4005
	ID_GUI_COMBO_PRINTER      = 4006
	ID_GUI_EDIT_PORT          = 4007
	ID_GUI_BTN_OPEN_TEMPLATES = 4008
	ID_GUI_BTN_ADD_TEMPLATE   = 4010
	ID_GUI_BTN_PREVIEW        = 4011
)

type GuiCallbacks struct {
	GetInstalledPrinters func() ([]string, error)
	OnSetDefaultPrinter  func(printerName string) error
	OnTestPrint          func(printerName string) error
	OnSavePort           func(port int) error
	OnOpenWebDashboard   func()
	OnOpenTemplateFolder func()
	OnAddTemplate        func(filePath string) (string, error)
	OnOpenPreview        func()
}

type NativeControlPanel struct {
	Hwnd          uintptr
	HComboPrinter uintptr
	HEditPort     uintptr
	HLabelStatus  uintptr
	HLabelPrinter uintptr
	Port          int
	CurrentPrinter string
	Callbacks     GuiCallbacks
	hFontRegular  uintptr
	hFontBold     uintptr
	mu            sync.Mutex
}

var (
	nativeGuiClassOnce sync.Once
	nativeGuiClassName *uint16
	activeGuiPanel     *NativeControlPanel
	activeGuiMu        sync.RWMutex
)

func registerNativeGuiClass() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	nativeGuiClassName, _ = syscall.UTF16PtrFromString("KnittoPrintServiceGuiWindowClass")

	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		activeGuiMu.RLock()
		gui := activeGuiPanel
		activeGuiMu.RUnlock()

		switch uint32(msg) {
		case WM_COMMAND:
			if gui != nil {
				cmdId := wParam & 0xFFFF
				gui.handleCommand(cmdId)
			}
			return 0

		case WM_CLOSE:
			// Minimalkan / sembunyikan ke system tray, jangan exit proses
			procShowWindow.Call(hwnd, uintptr(SW_HIDE))
			go winapi.TrimWorkingSet()
			return 0
		}

		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	})

	hIcon, _ := LoadDefaultIcon()
	if hIcon == 0 {
		hIcon, _, _ = procLoadIconW.Call(0, uintptr(IDI_APPLICATION))
	}

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   wndProc,
		HInstance:     hInst,
		HIcon:         hIcon,
		HbrBackground: uintptr(COLOR_BTNFACE + 1),
		LpszClassName: nativeGuiClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

// NewNativeControlPanel creates the Delphi-style Win32 Form Window
func NewNativeControlPanel(port int, defaultPrinter string, cb GuiCallbacks) (*NativeControlPanel, error) {
	nativeGuiClassOnce.Do(registerNativeGuiClass)

	hInst, _, _ := procGetModuleHandleW.Call(0)
	titleW, _ := syscall.UTF16PtrFromString("Knitto Print Service - Control Panel")

	// Center on screen
	sw, _, _ := procGetSystemMetrics.Call(0)
	sh, _, _ := procGetSystemMetrics.Call(1)
	winW := int32(525)
	winH := int32(480)
	winX := (int32(sw) - winW) / 2
	winY := (int32(sh) - winH) / 2

	style := uint32(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_CLIPCHILDREN | WS_CLIPSIBLINGS)

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(nativeGuiClassName)),
		uintptr(unsafe.Pointer(titleW)),
		uintptr(style),
		uintptr(winX), uintptr(winY),
		uintptr(winW), uintptr(winH),
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("gagal membuat native control panel window: %v", err)
	}

	// Create Segoe UI modern fonts
	fontName, _ := syscall.UTF16PtrFromString("Segoe UI")
	hFontRegular, _, _ := procCreateFontW.Call(
		15, 0, 0, 0,
		uintptr(FW_NORMAL),
		0, 0, 0, 1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(fontName)),
	)
	hFontBold, _, _ := procCreateFontW.Call(
		16, 0, 0, 0,
		uintptr(FW_BOLD),
		0, 0, 0, 1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(fontName)),
	)

	gui := &NativeControlPanel{
		Hwnd:           hwnd,
		Port:           port,
		CurrentPrinter: defaultPrinter,
		Callbacks:      cb,
		hFontRegular:   hFontRegular,
		hFontBold:      hFontBold,
	}

	activeGuiMu.Lock()
	activeGuiPanel = gui
	activeGuiMu.Unlock()

	gui.createControls(hInst)
	gui.RefreshPrinters()

	return gui, nil
}

func (g *NativeControlPanel) createControl(className, text string, style uint32, x, y, w, h int32, id uintptr, hInst uintptr) uintptr {
	clsW, _ := syscall.UTF16PtrFromString(className)
	txtW, _ := syscall.UTF16PtrFromString(text)
	hCtrl, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsW)),
		uintptr(unsafe.Pointer(txtW)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		g.Hwnd,
		id,
		hInst,
		0,
	)
	if hCtrl != 0 && g.hFontRegular != 0 {
		procSendMessageW.Call(hCtrl, WM_SETFONT, g.hFontRegular, 1)
	}
	return hCtrl
}

func (g *NativeControlPanel) createControls(hInst uintptr) {
	// 1. Header Area
	hTitle := g.createControl("STATIC", "Knitto Print Service (Windows Native GDI)", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 14, 470, 22, 0, hInst)
	if g.hFontBold != 0 {
		procSendMessageW.Call(hTitle, WM_SETFONT, g.hFontBold, 1)
	}

	statusText := fmt.Sprintf("● Status: ONLINE | Port HTTP: %d", g.Port)
	g.HLabelStatus = g.createControl("STATIC", statusText, WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 38, 470, 20, 0, hInst)

	prnText := fmt.Sprintf("🖨️ Terhubung ke: %s", g.CurrentPrinter)
	g.HLabelPrinter = g.createControl("STATIC", prnText, WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 60, 470, 20, 0, hInst)

	// 2. GroupBox 1: Pengaturan Printer
	hGrp1 := g.createControl("BUTTON", "  Pengaturan Printer Default  ", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 18, 88, 475, 138, 0, hInst)
	if g.hFontBold != 0 {
		procSendMessageW.Call(hGrp1, WM_SETFONT, g.hFontBold, 1)
	}

	g.createControl("STATIC", "Daftar Printer Terpasang di Windows:", WS_CHILD|WS_VISIBLE|SS_LEFT, 32, 114, 440, 18, 0, hInst)

	g.HComboPrinter = g.createControl(
		"COMBOBOX", "",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST,
		32, 134, 445, 200,
		uintptr(ID_GUI_COMBO_PRINTER),
		hInst,
	)

	g.createControl(
		"BUTTON", "💾 Jadikan Default",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		32, 175, 135, 34,
		uintptr(ID_GUI_BTN_SET_PRINTER),
		hInst,
	)

	g.createControl(
		"BUTTON", "📄 Tes Cetak",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		172, 175, 110, 34,
		uintptr(ID_GUI_BTN_TEST_PRINT),
		hInst,
	)

	g.createControl(
		"BUTTON", "👁️ FastReport Preview",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		288, 175, 189, 34,
		uintptr(ID_GUI_BTN_PREVIEW),
		hInst,
	)

	// 3. GroupBox 2: Pengaturan Port Server
	hGrp2 := g.createControl("BUTTON", "  Pengaturan Jaringan & Port Server  ", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 18, 238, 475, 86, 0, hInst)
	if g.hFontBold != 0 {
		procSendMessageW.Call(hGrp2, WM_SETFONT, g.hFontBold, 1)
	}

	g.createControl("STATIC", "Nomor Port HTTP (tersimpan di config.json):", WS_CHILD|WS_VISIBLE|SS_LEFT, 32, 258, 440, 18, 0, hInst)

	portStr := strconv.Itoa(g.Port)
	g.HEditPort = g.createControl(
		"EDIT", portStr,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_NUMBER|ES_LEFT,
		32, 280, 85, 26,
		uintptr(ID_GUI_EDIT_PORT),
		hInst,
	)

	g.createControl(
		"BUTTON", "💾 Simpan",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		125, 279, 80, 28,
		uintptr(ID_GUI_BTN_SAVE_PORT),
		hInst,
	)

	g.createControl("STATIC", "✓ Auto-Start: Selalu aktif di background saat Windows booting", WS_CHILD|WS_VISIBLE|SS_LEFT, 218, 284, 265, 20, 0, hInst)

	// 4. Footer Actions (2 Baris Rapi)
	// Baris 1: Template Management
	g.createControl(
		"BUTTON", "➕ Tambah / Upload Template (.json)",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		18, 338, 232, 36,
		uintptr(ID_GUI_BTN_ADD_TEMPLATE),
		hInst,
	)

	g.createControl(
		"BUTTON", "📂 Buka Folder Template",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		260, 338, 233, 36,
		uintptr(ID_GUI_BTN_OPEN_TEMPLATES),
		hInst,
	)

	// Baris 2: Navigasi Web & Tray
	g.createControl(
		"BUTTON", "🌐 Buka Web Dashboard",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		18, 382, 232, 36,
		uintptr(ID_GUI_BTN_OPEN_WEB),
		hInst,
	)

	g.createControl(
		"BUTTON", "⬇️ Sembunyikan ke Tray",
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		260, 382, 233, 36,
		uintptr(ID_GUI_BTN_HIDE),
		hInst,
	)
}

// RefreshPrinters updates the printer dropdown with Windows installed printers
func (g *NativeControlPanel) RefreshPrinters() {
	if g.HComboPrinter == 0 || g.Callbacks.GetInstalledPrinters == nil {
		return
	}

	printers, err := g.Callbacks.GetInstalledPrinters()
	if err != nil {
		return
	}

	procSendMessageW.Call(g.HComboPrinter, CB_RESETCONTENT, 0, 0)

	var selectedIndex int = -1
	for i, p := range printers {
		pW, _ := syscall.UTF16PtrFromString(p)
		procSendMessageW.Call(g.HComboPrinter, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(pW)))
		if strings.EqualFold(p, g.CurrentPrinter) {
			selectedIndex = i
		}
	}

	if selectedIndex >= 0 {
		procSendMessageW.Call(g.HComboPrinter, CB_SETCURSEL, uintptr(selectedIndex), 0)
	} else if len(printers) > 0 {
		procSendMessageW.Call(g.HComboPrinter, CB_SETCURSEL, 0, 0)
	}
}

// UpdatePrinter updates current printer info in GUI
func (g *NativeControlPanel) UpdatePrinter(printerName string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.CurrentPrinter = printerName

	if g.HLabelPrinter != 0 {
		txt := fmt.Sprintf("🖨️ Terhubung ke: %s", printerName)
		txtW, _ := syscall.UTF16PtrFromString(txt)
		procSetWindowTextW.Call(g.HLabelPrinter, uintptr(unsafe.Pointer(txtW)))
		procInvalidateRect.Call(g.HLabelPrinter, 0, 1)
		procUpdateWindow.Call(g.HLabelPrinter)
	}
	g.RefreshPrinters()
}

// UpdatePort updates current port info in GUI
func (g *NativeControlPanel) UpdatePort(newPort int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Port = newPort

	if g.HLabelStatus != 0 {
		txt := fmt.Sprintf("● Status: ONLINE | Port HTTP: %d", newPort)
		txtW, _ := syscall.UTF16PtrFromString(txt)
		procSetWindowTextW.Call(g.HLabelStatus, uintptr(unsafe.Pointer(txtW)))
		procInvalidateRect.Call(g.HLabelStatus, 0, 1)
		procUpdateWindow.Call(g.HLabelStatus)
	}
	if g.HEditPort != 0 {
		portStr := strconv.Itoa(newPort)
		portW, _ := syscall.UTF16PtrFromString(portStr)
		procSetWindowTextW.Call(g.HEditPort, uintptr(unsafe.Pointer(portW)))
	}
}

// Show displays the window and brings it to front
func (g *NativeControlPanel) Show() {
	if g.Hwnd != 0 {
		g.RefreshPrinters()
		procShowWindow.Call(g.Hwnd, uintptr(SW_SHOW))
		procSetForegroundWindow.Call(g.Hwnd)
	}
}

// Hide minimizes/hides the window back to system tray
func (g *NativeControlPanel) Hide() {
	if g.Hwnd != 0 {
		procShowWindow.Call(g.Hwnd, uintptr(SW_HIDE))
		go winapi.TrimWorkingSet()
	}
}

func (g *NativeControlPanel) handleCommand(cmdId uintptr) {
	switch cmdId {
	case ID_GUI_BTN_SET_PRINTER:
		g.onSetPrinterClicked()
	case ID_GUI_BTN_TEST_PRINT:
		g.onTestPrintClicked()
	case ID_GUI_BTN_PREVIEW:
		if g.Callbacks.OnOpenPreview != nil {
			go g.Callbacks.OnOpenPreview()
		}
	case ID_GUI_BTN_SAVE_PORT:
		g.onSavePortClicked()
	case ID_GUI_BTN_ADD_TEMPLATE:
		g.onAddTemplateClicked()
	case ID_GUI_BTN_OPEN_TEMPLATES:
		if g.Callbacks.OnOpenTemplateFolder != nil {
			g.Callbacks.OnOpenTemplateFolder()
		}
	case ID_GUI_BTN_OPEN_WEB:
		if g.Callbacks.OnOpenWebDashboard != nil {
			g.Callbacks.OnOpenWebDashboard()
		}
	case ID_GUI_BTN_HIDE:
		g.Hide()
	}
}

func buildJsonFilter() []uint16 {
	var filter []uint16
	filter = append(filter, syscall.StringToUTF16("File Template JSON (*.json)")...)
	filter = append(filter, syscall.StringToUTF16("*.json")...)
	filter = append(filter, syscall.StringToUTF16("Semua File (*.*)")...)
	filter = append(filter, syscall.StringToUTF16("*.*")...)
	filter = append(filter, 0)
	return filter
}

func (g *NativeControlPanel) onAddTemplateClicked() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in onAddTemplateClicked: %v\n", r)
			}
		}()

		procCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
		defer procCoUninitialize.Call()

		filter := buildJsonFilter()
		fileBuf := make([]uint16, 1024)
		titleW, _ := syscall.UTF16PtrFromString("Pilih File Template JSON untuk Ditambahkan")
		defExtW, _ := syscall.UTF16PtrFromString("json")

		var ofn OPENFILENAMEW
		ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
		if g != nil {
			ofn.HwndOwner = g.Hwnd
		}
		ofn.LpstrFilter = &filter[0]
		ofn.LpstrFile = &fileBuf[0]
		ofn.NMaxFile = 1024
		ofn.LpstrTitle = titleW
		ofn.LpstrDefExt = defExtW
		ofn.Flags = OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER

		r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
		if r == 0 {
			return // User membatalkan dialog
		}

		selectedPath := strings.TrimSpace(syscall.UTF16ToString(fileBuf))
		if selectedPath == "" {
			return
		}

		if g != nil && g.Callbacks.OnAddTemplate != nil {
			destPath, err := g.Callbacks.OnAddTemplate(selectedPath)
			if err != nil {
				g.showMessageBox(fmt.Sprintf("Gagal menambahkan template:\n%v", err), "Error Tambah Template", MB_OK|MB_ICONERROR)
				return
			}
			baseName := filepath.Base(destPath)
			g.showMessageBox(fmt.Sprintf("✅ Berhasil menambahkan template baru!\n\nNama File: %s\nLokasi: %s\n\nTemplate langsung aktif dan siap digunakan untuk cetak.", baseName, destPath), "Upload Template Sukses", MB_OK|MB_ICONINFORMATION)
		}
	}()
}

func (g *NativeControlPanel) onSetPrinterClicked() {
	idx, _, _ := procSendMessageW.Call(g.HComboPrinter, CB_GETCURSEL, 0, 0)
	if int32(idx) < 0 {
		g.showMessageBox("Pilih salah satu printer terlebih dahulu!", "Peringatan", MB_OK|MB_ICONWARNING)
		return
	}

	buf := make([]uint16, 256)
	procSendMessageW.Call(g.HComboPrinter, CB_GETLBTEXT, idx, uintptr(unsafe.Pointer(&buf[0])))
	selectedPrinter := syscall.UTF16ToString(buf)

	if g.Callbacks.OnSetDefaultPrinter != nil {
		err := g.Callbacks.OnSetDefaultPrinter(selectedPrinter)
		if err != nil {
			g.showMessageBox(fmt.Sprintf("Gagal mengubah printer: %v", err), "Error", MB_OK|MB_ICONERROR)
			return
		}
		g.UpdatePrinter(selectedPrinter)
		g.showMessageBox(fmt.Sprintf("Berhasil mengubah default printer ke:\n%s", selectedPrinter), "Sukses", MB_OK|MB_ICONINFORMATION)
	}
}

func (g *NativeControlPanel) onTestPrintClicked() {
	idx, _, _ := procSendMessageW.Call(g.HComboPrinter, CB_GETCURSEL, 0, 0)
	targetPrinter := g.CurrentPrinter
	if int32(idx) >= 0 {
		buf := make([]uint16, 256)
		procSendMessageW.Call(g.HComboPrinter, CB_GETLBTEXT, idx, uintptr(unsafe.Pointer(&buf[0])))
		targetPrinter = syscall.UTF16ToString(buf)
	}

	if g.Callbacks.OnTestPrint != nil {
		go func() {
			err := g.Callbacks.OnTestPrint(targetPrinter)
			if err != nil {
				g.showMessageBox(fmt.Sprintf("Gagal mencetak dokumen tes: %v", err), "Error Cetak", MB_OK|MB_ICONERROR)
			} else {
				g.showMessageBox(fmt.Sprintf("Perintah tes cetak berhasil dikirim ke:\n%s", targetPrinter), "Cetak Berhasil", MB_OK|MB_ICONINFORMATION)
			}
		}()
	}
}

func (g *NativeControlPanel) onSavePortClicked() {
	buf := make([]uint16, 32)
	procGetWindowTextW.Call(g.HEditPort, uintptr(unsafe.Pointer(&buf[0])), 32)
	portStr := syscall.UTF16ToString(buf)

	portVal, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || portVal < 1024 || portVal > 65535 {
		g.showMessageBox("Nomor port harus berupa angka antara 1024 dan 65535!", "Input Salah", MB_OK|MB_ICONWARNING)
		return
	}

	if portVal == g.Port {
		g.showMessageBox(fmt.Sprintf("Server sudah berjalan pada port %d.", portVal), "Informasi", MB_OK|MB_ICONINFORMATION)
		return
	}

	if g.Callbacks.OnSavePort != nil {
		err := g.Callbacks.OnSavePort(portVal)
		if err != nil {
			g.showMessageBox(fmt.Sprintf("Gagal mengubah port: %v", err), "Error", MB_OK|MB_ICONERROR)
			return
		}
		g.UpdatePort(portVal)
		g.showMessageBox(fmt.Sprintf("Port berhasil diubah ke %d!\n\nServer HTTP dan status form langsung aktif di port %d.", portVal, portVal), "Port Berhasil Diubah", MB_OK|MB_ICONINFORMATION)
	}
}

func (g *NativeControlPanel) showMessageBox(text, caption string, uType uint32) {
	txtW, _ := syscall.UTF16PtrFromString(text)
	capW, _ := syscall.UTF16PtrFromString(caption)
	procMessageBoxW.Call(g.Hwnd, uintptr(unsafe.Pointer(txtW)), uintptr(unsafe.Pointer(capW)), uintptr(uType))
}

// BringNativeGuiToFront finds and activates any existing GUI window on the desktop
func BringNativeGuiToFront() bool {
	clsW, _ := syscall.UTF16PtrFromString("KnittoPrintServiceGuiWindowClass")
	titleW, _ := syscall.UTF16PtrFromString("Knitto Print Service - Control Panel")
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(clsW)), uintptr(unsafe.Pointer(titleW)))
	if hwnd == 0 {
		hwnd, _, _ = procFindWindowW.Call(uintptr(unsafe.Pointer(clsW)), 0)
	}
	if hwnd != 0 {
		procShowWindow.Call(hwnd, uintptr(SW_SHOW))
		procSetForegroundWindow.Call(hwnd)
		return true
	}
	return false
}
