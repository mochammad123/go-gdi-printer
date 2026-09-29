package previewgui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"print-service/engine"
	"print-service/winapi"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modGdi32    = syscall.NewLazyDLL("gdi32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modComdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	modOle32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procPostQuitMessage     = modUser32.NewProc("PostQuitMessage")
	procGetMessageW         = modUser32.NewProc("GetMessageW")
	procTranslateMessage    = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = modUser32.NewProc("DispatchMessageW")
	procShowWindow          = modUser32.NewProc("ShowWindow")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procBringWindowToTop    = modUser32.NewProc("BringWindowToTop")
	procUpdateWindow        = modUser32.NewProc("UpdateWindow")
	procSendMessageW        = modUser32.NewProc("SendMessageW")
	procGetWindowTextW      = modUser32.NewProc("GetWindowTextW")
	procSetWindowTextW      = modUser32.NewProc("SetWindowTextW")
	procMessageBoxW         = modUser32.NewProc("MessageBoxW")
	procGetSystemMetrics    = modUser32.NewProc("GetSystemMetrics")
	procLoadCursorW         = modUser32.NewProc("LoadCursorW")
	procSetCursor           = modUser32.NewProc("SetCursor")
	procBeginPaint          = modUser32.NewProc("BeginPaint")
	procEndPaint            = modUser32.NewProc("EndPaint")
	procGetClientRect       = modUser32.NewProc("GetClientRect")
	procInvalidateRect      = modUser32.NewProc("InvalidateRect")
	procSetCapture          = modUser32.NewProc("SetCapture")
	procReleaseCapture      = modUser32.NewProc("ReleaseCapture")
	procDrawTextW           = modUser32.NewProc("DrawTextW")

	procGetModuleHandleW = modKernel32.NewProc("GetModuleHandleW")

	procCreateSolidBrush       = modGdi32.NewProc("CreateSolidBrush")
	procCreatePen              = modGdi32.NewProc("CreatePen")
	procSelectObject           = modGdi32.NewProc("SelectObject")
	procDeleteObject           = modGdi32.NewProc("DeleteObject")
	procSetBkMode              = modGdi32.NewProc("SetBkMode")
	procSetTextColor           = modGdi32.NewProc("SetTextColor")
	procCreateFontW            = modGdi32.NewProc("CreateFontW")
	procCreateCompatibleDC     = modGdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procBitBlt                 = modGdi32.NewProc("BitBlt")
	procDeleteDC               = modGdi32.NewProc("DeleteDC")
	procStretchDIBits          = modGdi32.NewProc("StretchDIBits")
	procSetStretchBltMode      = modGdi32.NewProc("SetStretchBltMode")
	procSetBrushOrgEx          = modGdi32.NewProc("SetBrushOrgEx")
	procRoundRect              = modGdi32.NewProc("RoundRect")
	procGetStockObject         = modGdi32.NewProc("GetStockObject")

	procGetSaveFileNameW = modComdlg32.NewProc("GetSaveFileNameW")
	procGetOpenFileNameW = modComdlg32.NewProc("GetOpenFileNameW")
	procCoInitializeEx   = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize   = modOle32.NewProc("CoUninitialize")

	procIntersectClipRect = modGdi32.NewProc("IntersectClipRect")
	procSaveDC            = modGdi32.NewProc("SaveDC")
	procRestoreDC         = modGdi32.NewProc("RestoreDC")
	procPolygon           = modGdi32.NewProc("Polygon")
	procRectangle         = modGdi32.NewProc("Rectangle")
	procEllipse           = modGdi32.NewProc("Ellipse")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_CLIPCHILDREN     = 0x02000000
	WS_CLIPSIBLINGS     = 0x04000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000

	BS_OWNERDRAW = 0x0000000B
	ES_LEFT      = 0x0000
	SS_CENTER    = 0x0001
	SS_LEFT      = 0x0000

	WM_DESTROY     = 0x0002
	WM_SIZE        = 0x0005
	WM_PAINT       = 0x000F
	WM_CLOSE       = 0x0010
	WM_ERASEBKGND  = 0x0014
	WM_SETFONT     = 0x0030
	WM_DRAWITEM    = 0x002B
	WM_COMMAND     = 0x0111
	WM_KEYDOWN     = 0x0100
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_MOUSEWHEEL  = 0x020A

	ODS_SELECTED = 0x0001
	ODS_FOCUS    = 0x0010
	ODS_HOTLIGHT = 0x0040

	SW_SHOWNORMAL = 1
	SW_SHOW       = 5
	SW_HIDE       = 0

	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_ICONERROR       = 0x00000010

	DT_CENTER     = 0x00000001
	DT_VCENTER    = 0x00000004
	DT_SINGLELINE = 0x00000020
	DT_LEFT       = 0x00000000

	WHITE_BRUSH = 0
	BLACK_BRUSH = 4

	IDC_ARROW   = 32512
	IDC_HAND    = 32649
	IDC_SIZEALL = 32646

	OFN_OVERWRITEPROMPT = 0x00000002
	OFN_FILEMUSTEXIST   = 0x00001000
	OFN_PATHMUSTEXIST   = 0x00000800
	OFN_EXPLORER        = 0x00080000

	// Menu / Button IDs (FastReport 1:1)
	ID_BTN_PRINT      = 5001
	ID_BTN_OPEN       = 5002
	ID_BTN_SAVE       = 5003
	ID_BTN_FIND       = 5004
	ID_EDIT_FIND      = 5005
	ID_BTN_PREV       = 5006
	ID_LABEL_PAGE     = 5007
	ID_BTN_NEXT       = 5008
	ID_BTN_ZOOM_OUT   = 5009
	ID_BTN_ZOOM_100   = 5010
	ID_BTN_ZOOM_IN    = 5011
	ID_BTN_FIT_WIDTH  = 5012
	ID_BTN_FIT_PAGE   = 5013
	ID_BTN_CLOSE      = 5014
	ID_BTN_FIRST      = 5015
	ID_BTN_LAST       = 5016
)

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

func rgb(r, g, b byte) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

// FastReportPreview represents the modern native Windows desktop preview window
type FastReportPreview struct {
	Hwnd             uintptr
	TargetPrinter    string
	TemplateName     string
	TemplateDir      string
	Template         *engine.DocumentTemplate
	Documents        []engine.DocumentData
	CurrentPageIndex int
	Bitmaps          []*engine.RenderedBitmap

	ZoomScale  float64
	PanOffsetX int32
	PanOffsetY int32

	IsDragging       bool
	DragStartX       int32
	DragStartY       int32
	DragStartOffsetX int32
	DragStartOffsetY int32

	HCursorArrow   uintptr
	HCursorHand    uintptr
	HCursorSizeAll uintptr

	HFontRegular   uintptr
	HFontBold      uintptr
	HFontToolbar   uintptr
	HFontStatus    uintptr
	HFontPageLabel uintptr

	HEditFind   uintptr
	HLabelPage  uintptr
	ButtonTexts map[uint32]string
	DoneChan    chan struct{}

	mu sync.Mutex
}

var (
	previewClassOnce sync.Once
	previewClassName *uint16
	activePreviews   = make(map[uintptr]*FastReportPreview)
	activePreviewsMu sync.RWMutex
)

func registerPreviewClass() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	previewClassName, _ = syscall.UTF16PtrFromString("KnittoFastReportPreviewWindow")

	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		activePreviewsMu.RLock()
		p := activePreviews[hwnd]
		activePreviewsMu.RUnlock()

		switch uint32(msg) {
		case WM_COMMAND:
			if p != nil {
				cmdId := uint32(wParam & 0xFFFF)
				p.handleCommand(cmdId)
			}
			return 0

		case WM_DRAWITEM:
			if p != nil {
				dis := (*winapi.DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
				p.drawOwnerDrawButton(dis)
				return 1
			}

		case WM_PAINT:
			if p != nil {
				p.paintWindow(hwnd)
				return 0
			}

		case WM_ERASEBKGND:
			return 1 // Mencegah kedip (flicker-free)

		case WM_SIZE:
			if p != nil {
				procInvalidateRect.Call(hwnd, 0, 0)
			}
			return 0

		case WM_MOUSEWHEEL:
			if p != nil {
				delta := int16(wParam >> 16)
				p.handleMouseWheel(delta)
			}
			return 0

		case WM_LBUTTONDOWN:
			if p != nil {
				x := int32(lParam & 0xFFFF)
				y := int32((lParam >> 16) & 0xFFFF)
				p.handleMouseDown(x, y)
			}
			return 0

		case WM_MOUSEMOVE:
			if p != nil {
				x := int32(lParam & 0xFFFF)
				y := int32((lParam >> 16) & 0xFFFF)
				p.handleMouseMove(x, y)
			}
			return 0

		case WM_LBUTTONUP:
			if p != nil {
				p.handleMouseUp()
			}
			return 0

		case WM_KEYDOWN:
			if p != nil {
				vk := wParam & 0xFF
				p.handleKeyDown(uint32(vk))
			}
			return 0

		case WM_CLOSE:
			procDestroyWindow.Call(hwnd)
			return 0

		case WM_DESTROY:
			activePreviewsMu.Lock()
			delete(activePreviews, hwnd)
			activePreviewsMu.Unlock()
			if p != nil && p.DoneChan != nil {
				select {
				case <-p.DoneChan:
				default:
					close(p.DoneChan)
				}
			}
			procPostQuitMessage.Call(0)
			return 0
		}

		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	})

	hIcon, _, _ := winapi.ModUser32.NewProc("LoadIconW").Call(0, uintptr(32512)) // IDI_APPLICATION
	hCursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))

	hBrBg, _, _ := procGetStockObject.Call(uintptr(BLACK_BRUSH))
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   wndProc,
		HInstance:     hInst,
		HIcon:         hIcon,
		HCursor:       hCursor,
		HbrBackground: hBrBg,
		LpszClassName: previewClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

// ShowFastReportPreview loads the template and documents, renders them, and opens the Native Windows GUI Form
func ShowFastReportPreview(targetPrinter, templateName, templateDir string, docs []engine.DocumentData) (*FastReportPreview, error) {
	if len(docs) == 0 {
		docs = []engine.DocumentData{{}}
	}

	tpl, err := engine.LoadTemplateWithDir(templateName, templateDir)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat template '%s': %w", templateName, err)
	}

	dpi, resolvedPrinter := engine.GetTemplateDPI(targetPrinter)
	bitmaps := make([]*engine.RenderedBitmap, 0, len(docs))
	for _, doc := range docs {
		bmp, errBmp := engine.RenderPreviewBitmap(tpl, doc, dpi)
		if errBmp != nil {
			return nil, fmt.Errorf("gagal merender bitmap preview: %w", errBmp)
		}
		bitmaps = append(bitmaps, bmp)
	}

	preview := &FastReportPreview{
		TargetPrinter:    resolvedPrinter,
		TemplateName:     templateName,
		TemplateDir:      templateDir,
		Template:         tpl,
		Documents:        docs,
		CurrentPageIndex: 0,
		Bitmaps:          bitmaps,
		ZoomScale:        1.0, // 100%
		ButtonTexts:      make(map[uint32]string),
		DoneChan:         make(chan struct{}),
	}

	errChan := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Pastikan thread terhubung ke desktop interaktif pengguna "Default" (layar fisik monitor)
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

		procCoInitializeEx.Call(0, 2)
		defer procCoUninitialize.Call()

		previewClassOnce.Do(registerPreviewClass)

		hInst, _, _ := procGetModuleHandleW.Call(0)
		title := fmt.Sprintf("🖨️ FastReport Print Preview - [%s] (Knitto Windows Native GDI)", templateName)
		titleW, _ := syscall.UTF16PtrFromString(title)

		sw, _, _ := procGetSystemMetrics.Call(0)
		sh, _, _ := procGetSystemMetrics.Call(1)

		winW := int32(1100)
		winH := int32(760)
		if sw > 0 && winW > int32(sw) {
			winW = int32(sw) - 40
		}
		if sh > 0 && winH > int32(sh) {
			winH = int32(sh) - 60
		}
		if winW < 600 {
			winW = 1000
		}
		if winH < 400 {
			winH = 700
		}
		winX := int32(100)
		winY := int32(100)
		if sw > 0 && sh > 0 {
			winX = (int32(sw) - winW) / 2
			winY = (int32(sh) - winH) / 2
		}

		style := uint32(WS_OVERLAPPEDWINDOW | WS_VISIBLE | WS_CLIPCHILDREN | WS_CLIPSIBLINGS)

		hwnd, _, errCreate := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(previewClassName)),
			uintptr(unsafe.Pointer(titleW)),
			uintptr(style),
			uintptr(winX), uintptr(winY),
			uintptr(winW), uintptr(winH),
			0, 0, hInst, 0,
		)
		if hwnd == 0 {
			errChan <- fmt.Errorf("gagal membuat window preview FastReport: %v", errCreate)
			return
		}

		preview.Hwnd = hwnd

		activePreviewsMu.Lock()
		activePreviews[hwnd] = preview
		activePreviewsMu.Unlock()

		preview.initCursorsAndFonts()
		preview.createToolbarControls(hInst)
		preview.updatePageLabel()
		procShowWindow.Call(hwnd, uintptr(SW_SHOWNORMAL))
		procShowWindow.Call(hwnd, uintptr(SW_SHOW))
		procBringWindowToTop.Call(hwnd)
		procSetForegroundWindow.Call(hwnd)
		procUpdateWindow.Call(hwnd)

		errChan <- nil

		// Message Loop untuk Preview Window
		var msg MSG
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()

	if err := <-errChan; err != nil {
		return nil, err
	}

	return preview, nil
}

func (p *FastReportPreview) initCursorsAndFonts() {
	p.HCursorArrow, _, _ = procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	p.HCursorHand, _, _ = procLoadCursorW.Call(0, uintptr(IDC_HAND))
	p.HCursorSizeAll, _, _ = procLoadCursorW.Call(0, uintptr(IDC_SIZEALL))

	segoeW, _ := syscall.UTF16PtrFromString("Segoe UI")
	// Semua font dengan CLEARTYPE_QUALITY (5) agar tajam seperti FastReport
	// Regular 13px — untuk EDIT box dan label
	p.HFontRegular, _, _ = procCreateFontW.Call(13, 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(segoeW)))
	// Bold 13px
	p.HFontBold, _, _ = procCreateFontW.Call(13, 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(segoeW)))
	// Toolbar 12px regular — lebih compact seperti FastReport 6
	p.HFontToolbar, _, _ = procCreateFontW.Call(12, 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(segoeW)))
	// Status bar 11px
	p.HFontStatus, _, _ = procCreateFontW.Call(11, 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(segoeW)))
	// Page label 12px bold
	p.HFontPageLabel, _, _ = procCreateFontW.Call(12, 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(segoeW)))
}

func (p *FastReportPreview) addOwnerDrawButton(id uint32, text string, x, y, w, h int32, hInst uintptr) uintptr {
	p.ButtonTexts[id] = text
	clsW, _ := syscall.UTF16PtrFromString("BUTTON")
	txtW, _ := syscall.UTF16PtrFromString(text)
	style := uint32(WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_OWNERDRAW)

	hBtn, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsW)),
		uintptr(unsafe.Pointer(txtW)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		p.Hwnd,
		uintptr(id),
		hInst,
		0,
	)
	return hBtn
}

func (p *FastReportPreview) createToolbarControls(hInst uintptr) {
	// Toolbar height 38px — persis FastReport 6 compact toolbar
	// 1. Group Dokumen (Print, Open, Save) — PURE ICONS!
	p.addOwnerDrawButton(ID_BTN_PRINT, "", 6, 5, 28, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_OPEN, "", 36, 5, 28, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_SAVE, "", 66, 5, 28, 28, hInst)

	// 2. Group Cari / Find (Icon + Search Box)
	p.addOwnerDrawButton(ID_BTN_FIND, "", 104, 5, 28, 28, hInst)

	editClsW, _ := syscall.UTF16PtrFromString("EDIT")
	editTxtW, _ := syscall.UTF16PtrFromString("")
	editStyle := uint32(WS_CHILD | WS_VISIBLE | WS_TABSTOP | WS_BORDER | ES_LEFT)
	p.HEditFind, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(editClsW)),
		uintptr(unsafe.Pointer(editTxtW)),
		uintptr(editStyle),
		uintptr(136), uintptr(7), uintptr(100), uintptr(24),
		p.Hwnd,
		uintptr(ID_EDIT_FIND),
		hInst,
		0,
	)
	if p.HEditFind != 0 && p.HFontRegular != 0 {
		procSendMessageW.Call(p.HEditFind, WM_SETFONT, p.HFontRegular, 1)
	}

	// 3. Group Paging (|<<, <, label, >, >>|)
	p.addOwnerDrawButton(ID_BTN_FIRST, "", 248, 5, 26, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_PREV, "", 276, 5, 26, 28, hInst)

	statClsW, _ := syscall.UTF16PtrFromString("STATIC")
	statTxtW, _ := syscall.UTF16PtrFromString("1 of 1")
	statStyle := uint32(WS_CHILD | WS_VISIBLE | SS_CENTER)
	p.HLabelPage, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(statClsW)),
		uintptr(unsafe.Pointer(statTxtW)),
		uintptr(statStyle),
		uintptr(304), uintptr(10), uintptr(56), uintptr(18),
		p.Hwnd,
		uintptr(ID_LABEL_PAGE),
		hInst,
		0,
	)
	if p.HLabelPage != 0 && p.HFontPageLabel != 0 {
		procSendMessageW.Call(p.HLabelPage, WM_SETFONT, p.HFontPageLabel, 1)
	}

	p.addOwnerDrawButton(ID_BTN_NEXT, "", 362, 5, 26, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_LAST, "", 390, 5, 26, 28, hInst)

	// 4. Group Zoom (-, 100%, +)
	p.addOwnerDrawButton(ID_BTN_ZOOM_OUT, "", 428, 5, 26, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_ZOOM_100, "100%", 456, 5, 48, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_ZOOM_IN, "", 506, 5, 26, 28, hInst)

	// 5. Group View (Fit Width, Fit Page)
	p.addOwnerDrawButton(ID_BTN_FIT_WIDTH, "", 544, 5, 28, 28, hInst)
	p.addOwnerDrawButton(ID_BTN_FIT_PAGE, "", 574, 5, 28, 28, hInst)

	// 6. Close Button
	p.addOwnerDrawButton(ID_BTN_CLOSE, "Close", 614, 5, 58, 28, hInst)
}

func (p *FastReportPreview) updatePageLabel() {
	if p.HLabelPage != 0 {
		txt := fmt.Sprintf("%d of %d", p.CurrentPageIndex+1, len(p.Documents))
		txtW, _ := syscall.UTF16PtrFromString(txt)
		procSetWindowTextW.Call(p.HLabelPage, uintptr(unsafe.Pointer(txtW)))
		procInvalidateRect.Call(p.HLabelPage, 0, 1)
	}
}

func drawGdiIcon(hdc uintptr, ctlId uint32, cx, cy int32, isSelected bool) {
	if isSelected {
		cx++
		cy++
	}

	switch ctlId {
	case ID_BTN_PRINT:
		// Printer Body (Grey)
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(40, 40, 40)))
		hBrBody, _, _ := procCreateSolidBrush.Call(uintptr(rgb(205, 210, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBrBody)
		procRoundRect.Call(hdc, uintptr(cx-7), uintptr(cy-2), uintptr(cx+8), uintptr(cy+6), 2, 2)

		// Paper Top
		hBrWhite, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		procSelectObject.Call(hdc, hBrWhite)
		procRectangle.Call(hdc, uintptr(cx-5), uintptr(cy-8), uintptr(cx+5), uintptr(cy-1))

		// Lines on top paper
		hPenLine, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(160, 160, 160)))
		procSelectObject.Call(hdc, hPenLine)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy-6), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy-6))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy-4), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy-4))

		// Paper Bottom
		procSelectObject.Call(hdc, hPen)
		procSelectObject.Call(hdc, hBrWhite)
		procRectangle.Call(hdc, uintptr(cx-5), uintptr(cy+2), uintptr(cx+5), uintptr(cy+8))
		procSelectObject.Call(hdc, hPenLine)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy+4), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy+4))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy+6), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy+6))

		// Green LED
		hBrGreen, _, _ := procCreateSolidBrush.Call(uintptr(rgb(34, 197, 94)))
		procSelectObject.Call(hdc, hBrGreen)
		procEllipse.Call(hdc, uintptr(cx+4), uintptr(cy-1), uintptr(cx+6), uintptr(cy+1))

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBrBody)
		procDeleteObject.Call(hPenLine)
		procDeleteObject.Call(hBrGreen)

	case ID_BTN_OPEN:
		// Folder
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(180, 130, 20)))
		hBrFolder, _, _ := procCreateSolidBrush.Call(uintptr(rgb(245, 180, 40)))
		hBrFront, _, _ := procCreateSolidBrush.Call(uintptr(rgb(255, 205, 70)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBrFolder)

		procRoundRect.Call(hdc, uintptr(cx-7), uintptr(cy-6), uintptr(cx-1), uintptr(cy-2), 2, 2)
		procRoundRect.Call(hdc, uintptr(cx-7), uintptr(cy-4), uintptr(cx+7), uintptr(cy+5), 2, 2)

		hBrWhite, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		procSelectObject.Call(hdc, hBrWhite)
		procRectangle.Call(hdc, uintptr(cx-4), uintptr(cy-6), uintptr(cx+5), uintptr(cy+1))

		procSelectObject.Call(hdc, hBrFront)
		ptFlap := []POINT{
			{cx - 7, cy - 1},
			{cx + 5, cy - 1},
			{cx + 7, cy + 6},
			{cx - 5, cy + 6},
		}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&ptFlap[0])), 4)

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBrFolder)
		procDeleteObject.Call(hBrFront)

	case ID_BTN_SAVE:
		// 3.5" Diskette
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(20, 60, 160)))
		hBrDisk, _, _ := procCreateSolidBrush.Call(uintptr(rgb(37, 99, 235)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBrDisk)

		procRoundRect.Call(hdc, uintptr(cx-7), uintptr(cy-7), uintptr(cx+8), uintptr(cy+8), 2, 2)

		hBrSilver, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 225, 230)))
		procSelectObject.Call(hdc, hBrSilver)
		procRectangle.Call(hdc, uintptr(cx-4), uintptr(cy-7), uintptr(cx+4), uintptr(cy-2))

		hBrBlack, _, _ := procGetStockObject.Call(uintptr(BLACK_BRUSH))
		procSelectObject.Call(hdc, hBrBlack)
		procRectangle.Call(hdc, uintptr(cx-1), uintptr(cy-6), uintptr(cx+2), uintptr(cy-3))

		hBrWhite, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		procSelectObject.Call(hdc, hBrWhite)
		procRectangle.Call(hdc, uintptr(cx-5), uintptr(cy), uintptr(cx+5), uintptr(cy+6))

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBrDisk)
		procDeleteObject.Call(hBrSilver)

	case ID_BTN_FIND:
		// Magnifying Glass
		hPenGlass, _, _ := procCreatePen.Call(0, 2, uintptr(rgb(220, 220, 220)))
		hBrEmpty, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		oldPen, _, _ := procSelectObject.Call(hdc, hPenGlass)
		oldBr, _, _ := procSelectObject.Call(hdc, hBrEmpty)

		procEllipse.Call(hdc, uintptr(cx-6), uintptr(cy-6), uintptr(cx+2), uintptr(cy+2))

		hPenHandle, _, _ := procCreatePen.Call(0, 3, uintptr(rgb(220, 180, 80)))
		procSelectObject.Call(hdc, hPenHandle)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx+1), uintptr(cy+1), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+6), uintptr(cy+6))

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPenGlass)
		procDeleteObject.Call(hPenHandle)

	case ID_BTN_FIRST:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBr, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBr)

		procRectangle.Call(hdc, uintptr(cx-6), uintptr(cy-5), uintptr(cx-4), uintptr(cy+6))
		pt := []POINT{
			{cx + 5, cy - 5},
			{cx + 5, cy + 5},
			{cx - 3, cy},
		}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pt[0])), 3)

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBr)

	case ID_BTN_PREV:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBr, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBr)

		pt := []POINT{
			{cx + 4, cy - 5},
			{cx + 4, cy + 5},
			{cx - 4, cy},
		}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pt[0])), 3)

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBr)

	case ID_BTN_NEXT:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBr, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBr)

		pt := []POINT{
			{cx - 4, cy - 5},
			{cx - 4, cy + 5},
			{cx + 4, cy},
		}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pt[0])), 3)

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBr)

	case ID_BTN_LAST:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBr, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBr)

		pt := []POINT{
			{cx - 5, cy - 5},
			{cx - 5, cy + 5},
			{cx + 3, cy},
		}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pt[0])), 3)
		procRectangle.Call(hdc, uintptr(cx+4), uintptr(cy-5), uintptr(cx+6), uintptr(cy+6))

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBr)

	case ID_BTN_ZOOM_OUT:
		hPen, _, _ := procCreatePen.Call(0, 2, uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-5), uintptr(cy), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+6), uintptr(cy))
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)

	case ID_BTN_ZOOM_IN:
		hPen, _, _ := procCreatePen.Call(0, 2, uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-5), uintptr(cy), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+6), uintptr(cy))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx), uintptr(cy-5), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx), uintptr(cy+6))
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)

	case ID_BTN_FIT_WIDTH:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBr, _, _ := procCreateSolidBrush.Call(uintptr(rgb(220, 220, 220)))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBr)

		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-7), uintptr(cy-6), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx-7), uintptr(cy+7))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx+7), uintptr(cy-6), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+7), uintptr(cy+7))

		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-5), uintptr(cy), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+6), uintptr(cy))

		ptL := []POINT{{cx - 2, cy - 3}, {cx - 2, cy + 3}, {cx - 6, cy}}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&ptL[0])), 3)
		ptR := []POINT{{cx + 2, cy - 3}, {cx + 2, cy + 3}, {cx + 6, cy}}
		procPolygon.Call(hdc, uintptr(unsafe.Pointer(&ptR[0])), 3)

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hBr)

	case ID_BTN_FIT_PAGE:
		hPen, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(220, 220, 220)))
		hBrWhite, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		oldBr, _, _ := procSelectObject.Call(hdc, hBrWhite)

		procRectangle.Call(hdc, uintptr(cx-5), uintptr(cy-7), uintptr(cx+5), uintptr(cy+7))

		hPenLine, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(160, 160, 160)))
		procSelectObject.Call(hdc, hPenLine)
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy-4), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy-4))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy-1), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy-1))
		winapi.ProcMoveToEx.Call(hdc, uintptr(cx-3), uintptr(cy+2), 0)
		winapi.ProcLineTo.Call(hdc, uintptr(cx+3), uintptr(cy+2))

		procSelectObject.Call(hdc, oldBr)
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
		procDeleteObject.Call(hPenLine)
	}
}

func (p *FastReportPreview) drawOwnerDrawButton(dis *winapi.DRAWITEMSTRUCT) {
	rc := dis.RcItem
	text := p.ButtonTexts[dis.CtlID]

	isSelected := (dis.ItemState & ODS_SELECTED) != 0
	isHot := (dis.ItemState & ODS_HOTLIGHT) != 0

	// ─── Warna ala FastReport 6 / Office 2016 Industrial Grey ─────────────────
	var bgColor, borderColor uint32
	textColor := rgb(240, 240, 240)

	switch dis.CtlID {
	case ID_BTN_PRINT:
		if isSelected {
			bgColor = rgb(0, 84, 153)
			borderColor = rgb(0, 108, 190)
		} else if isHot {
			bgColor = rgb(28, 120, 200)
			borderColor = rgb(0, 108, 190)
		} else {
			bgColor = rgb(0, 114, 198)
			borderColor = rgb(0, 90, 165)
		}

	case ID_BTN_CLOSE:
		if isSelected {
			bgColor = rgb(160, 30, 30)
			borderColor = rgb(190, 50, 50)
		} else if isHot {
			bgColor = rgb(196, 43, 28)
			borderColor = rgb(210, 60, 40)
		} else {
			bgColor = rgb(76, 76, 76)
			borderColor = rgb(100, 100, 100)
		}
		textColor = rgb(255, 255, 255)

	default:
		if isSelected {
			bgColor = rgb(55, 55, 55)
			borderColor = rgb(90, 90, 90)
		} else if isHot {
			bgColor = rgb(72, 72, 72)
			borderColor = rgb(105, 105, 105)
		} else {
			bgColor = rgb(62, 62, 62)
			borderColor = rgb(80, 80, 80)
		}
	}

	hBrush, _, _ := procCreateSolidBrush.Call(uintptr(bgColor))
	hPen, _, _ := procCreatePen.Call(0, 1, uintptr(borderColor))
	oldBr, _, _ := procSelectObject.Call(dis.HDC, hBrush)
	oldPen, _, _ := procSelectObject.Call(dis.HDC, hPen)
	procRoundRect.Call(dis.HDC, uintptr(rc.Left), uintptr(rc.Top), uintptr(rc.Right), uintptr(rc.Bottom), 3, 3)
	procSelectObject.Call(dis.HDC, oldPen)
	procSelectObject.Call(dis.HDC, oldBr)
	procDeleteObject.Call(hBrush)
	procDeleteObject.Call(hPen)

	cx := (rc.Left + rc.Right) / 2
	cy := (rc.Top + rc.Bottom) / 2

	if dis.CtlID == ID_BTN_ZOOM_100 || dis.CtlID == ID_BTN_CLOSE {
		procSetBkMode.Call(dis.HDC, uintptr(winapi.TRANSPARENT))
		procSetTextColor.Call(dis.HDC, uintptr(textColor))
		oldFont, _, _ := procSelectObject.Call(dis.HDC, p.HFontToolbar)
		tW, _ := syscall.UTF16FromString(text)
		rcText := winapi.RECT{Left: rc.Left + 2, Top: rc.Top, Right: rc.Right - 2, Bottom: rc.Bottom}
		procDrawTextW.Call(
			dis.HDC,
			uintptr(unsafe.Pointer(&tW[0])),
			uintptr(len(tW)-1),
			uintptr(unsafe.Pointer(&rcText)),
			uintptr(DT_CENTER|DT_VCENTER|DT_SINGLELINE),
		)
		procSelectObject.Call(dis.HDC, oldFont)
	} else {
		drawGdiIcon(dis.HDC, dis.CtlID, cx, cy, isSelected)
	}
}

func (p *FastReportPreview) getCurrentBitmap() *engine.RenderedBitmap {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.Bitmaps) == 0 {
		return nil
	}
	if p.CurrentPageIndex < 0 {
		p.CurrentPageIndex = 0
	}
	if p.CurrentPageIndex >= len(p.Bitmaps) {
		p.CurrentPageIndex = len(p.Bitmaps) - 1
	}
	return p.Bitmaps[p.CurrentPageIndex]
}

func (p *FastReportPreview) paintWindow(hwnd uintptr) {
	var ps winapi.PAINTSTRUCT
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var rcClient winapi.RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rcClient)))
	w := rcClient.Right - rcClient.Left
	h := rcClient.Bottom - rcClient.Top
	if w <= 0 || h <= 0 {
		return
	}

	// Double buffering — tidak flicker
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdc)
	if hdcMem == 0 {
		return
	}
	defer procDeleteDC.Call(hdcMem)

	hbmMem, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	if hbmMem == 0 {
		return
	}
	defer procDeleteObject.Call(hbmMem)

	oldBmp, _, _ := procSelectObject.Call(hdcMem, hbmMem)
	defer procSelectObject.Call(hdcMem, oldBmp)

	const tbH int32 = 38  // Toolbar height 38px — persis FastReport compact toolbar
	const stH int32 = 22  // Status bar height 22px

	// 1. ── Toolbar Background (Industrial Grey #3C3C3C seperti FastReport/VS) ──
	hBrTop, _, _ := procCreateSolidBrush.Call(uintptr(rgb(60, 60, 60)))
	rcTop := winapi.RECT{Left: 0, Top: 0, Right: w, Bottom: tbH}
	winapi.ProcFillRect.Call(hdcMem, uintptr(unsafe.Pointer(&rcTop)), hBrTop)
	procDeleteObject.Call(hBrTop)

	// Garis bawah toolbar (tipis terang — depth separator)
	hPenSep, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(45, 45, 45)))
	oldPen, _, _ := procSelectObject.Call(hdcMem, hPenSep)
	winapi.ProcMoveToEx.Call(hdcMem, 0, uintptr(tbH-1), 0)
	winapi.ProcLineTo.Call(hdcMem, uintptr(w), uintptr(tbH-1))

	// Garis terang di atas toolbar (highlight 1px ala Office)
	procSelectObject.Call(hdcMem, oldPen)
	procDeleteObject.Call(hPenSep)
	hPenHi, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(80, 80, 80)))
	procSelectObject.Call(hdcMem, hPenHi)
	winapi.ProcMoveToEx.Call(hdcMem, 0, 0, 0)
	winapi.ProcLineTo.Call(hdcMem, uintptr(w), 0)

	// Garis pemisah grup (vertical separator tipis ala FastReport)
	hPenGrp, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(95, 95, 95)))
	procSelectObject.Call(hdcMem, hPenGrp)
	drawSeparator := func(x int32) {
		winapi.ProcMoveToEx.Call(hdcMem, uintptr(x), 6, 0)
		winapi.ProcLineTo.Call(hdcMem, uintptr(x), uintptr(tbH-6))
	}
	drawSeparator(98)   // setelah Save
	drawSeparator(240)  // setelah Search
	drawSeparator(420)  // setelah Paging
	drawSeparator(538)  // setelah Zoom
	drawSeparator(606)  // setelah View Modes
	procSelectObject.Call(hdcMem, oldPen)
	procDeleteObject.Call(hPenGrp)
	procDeleteObject.Call(hPenHi)

	// 2. ── Canvas Background (Grey #808080 ala FastReport default) ──────────────
	hBrCanvas, _, _ := procCreateSolidBrush.Call(uintptr(rgb(128, 128, 128)))
	rcCanvas := winapi.RECT{Left: 0, Top: tbH, Right: w, Bottom: h - stH}
	winapi.ProcFillRect.Call(hdcMem, uintptr(unsafe.Pointer(&rcCanvas)), hBrCanvas)
	procDeleteObject.Call(hBrCanvas)

	// 3. ── Status Bar (sama warna toolbar) ────────────────────────────────────
	hBrStatus, _, _ := procCreateSolidBrush.Call(uintptr(rgb(60, 60, 60)))
	rcStatus := winapi.RECT{Left: 0, Top: h - stH, Right: w, Bottom: h}
	winapi.ProcFillRect.Call(hdcMem, uintptr(unsafe.Pointer(&rcStatus)), hBrStatus)
	procDeleteObject.Call(hBrStatus)

	// Garis batas atas status bar
	hPenSt, _, _ := procCreatePen.Call(0, 1, uintptr(rgb(45, 45, 45)))
	oldPen2, _, _ := procSelectObject.Call(hdcMem, hPenSt)
	winapi.ProcMoveToEx.Call(hdcMem, 0, uintptr(h-stH), 0)
	winapi.ProcLineTo.Call(hdcMem, uintptr(w), uintptr(h-stH))
	procSelectObject.Call(hdcMem, oldPen2)
	procDeleteObject.Call(hPenSt)

	// 4. ── Render Kertas Dokumen (Floating Paper + Shadow) ─────────────────────
	// KUNCI: Clip region agar kertas dokumen TIDAK PERNAH tembus/bocor ke Toolbar (y < tbH) maupun Status Bar (y > h - stH)
	savedDC, _, _ := procSaveDC.Call(hdcMem)
	procIntersectClipRect.Call(hdcMem, 0, uintptr(tbH), uintptr(w), uintptr(h-stH))

	bmp := p.getCurrentBitmap()
	if bmp != nil && bmp.WidthPx > 0 && bmp.HeightPx > 0 {
		zoom := p.ZoomScale
		if zoom <= 0.05 {
			zoom = 0.05
		}

		paperW := int32(float64(bmp.WidthPx) * zoom)
		paperH := int32(float64(bmp.HeightPx) * zoom)

		canvasH := (h - stH) - tbH
		paperX := (w-paperW)/2 + p.PanOffsetX
		var paperY int32
		if paperH < canvasH {
			paperY = tbH + (canvasH-paperH)/2 + p.PanOffsetY
		} else {
			paperY = tbH + 16 + p.PanOffsetY
		}

		// A. Drop Shadow — tipis 4px seperti FastReport (satu layer, soft)
		shClr := rgb(60, 60, 60)
		hBrSh, _, _ := procCreateSolidBrush.Call(uintptr(shClr))
		rcSh := winapi.RECT{
			Left:   paperX + 4,
			Top:    paperY + 4,
			Right:  paperX + paperW + 4,
			Bottom: paperY + paperH + 4,
		}
		winapi.ProcFillRect.Call(hdcMem, uintptr(unsafe.Pointer(&rcSh)), hBrSh)
		procDeleteObject.Call(hBrSh)

		// B. Lembar kertas putih bersih
		hBrPaper, _, _ := procGetStockObject.Call(uintptr(WHITE_BRUSH))
		rcPaper := winapi.RECT{
			Left:   paperX,
			Top:    paperY,
			Right:  paperX + paperW,
			Bottom: paperY + paperH,
		}
		winapi.ProcFillRect.Call(hdcMem, uintptr(unsafe.Pointer(&rcPaper)), hBrPaper)

		// C. Render Bitmap (HALFTONE = anti-aliased scaling)
		procSetStretchBltMode.Call(hdcMem, uintptr(winapi.STRETCH_HALFTONE))
		procSetBrushOrgEx.Call(hdcMem, 0, 0, 0)
		procStretchDIBits.Call(
			hdcMem,
			uintptr(paperX),
			uintptr(paperY),
			uintptr(paperW),
			uintptr(paperH),
			0,
			0,
			uintptr(bmp.WidthPx),
			uintptr(bmp.HeightPx),
			uintptr(unsafe.Pointer(&bmp.Pixels[0])),
			uintptr(unsafe.Pointer(&bmp.BMI)),
			uintptr(winapi.DIB_RGB_COLORS),
			uintptr(winapi.SRCCOPY),
		)
	}

	procRestoreDC.Call(hdcMem, savedDC)

	// 5. ── Status Bar Text ──────────────────────────────────────────────────────
	procSetBkMode.Call(hdcMem, uintptr(winapi.TRANSPARENT))
	procSetTextColor.Call(hdcMem, uintptr(rgb(200, 200, 200)))
	oldFont, _, _ := procSelectObject.Call(hdcMem, p.HFontStatus)

	var docInfo string
	if p.Template != nil && bmp != nil {
		docInfo = fmt.Sprintf("Printer: %s  |  Paper: %.1f x %.1f mm (%d x %d px)  |  Zoom: %d%%  |  Page: %d of %d  |  203 DPI Native GDI",
			p.TargetPrinter,
			p.Template.WidthMm, p.Template.HeightMm,
			bmp.WidthPx, bmp.HeightPx,
			int(p.ZoomScale*100),
			p.CurrentPageIndex+1, len(p.Documents),
		)
	} else {
		docInfo = fmt.Sprintf("Printer: %s  |  Zoom: %d%%  |  Page: %d of %d",
			p.TargetPrinter, int(p.ZoomScale*100), p.CurrentPageIndex+1, len(p.Documents))
	}

	rcStatusText := winapi.RECT{Left: 10, Top: h - stH + 3, Right: w - 10, Bottom: h - 2}
	sW, _ := syscall.UTF16FromString(docInfo)
	procDrawTextW.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&sW[0])),
		uintptr(len(sW)-1),
		uintptr(unsafe.Pointer(&rcStatusText)),
		uintptr(DT_LEFT|DT_VCENTER|DT_SINGLELINE),
	)
	procSelectObject.Call(hdcMem, oldFont)

	// 6. ── Blit ke layar (Flicker-Free) ──────────────────────────────────────
	procBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), hdcMem, 0, 0, uintptr(winapi.SRCCOPY))
}

func (p *FastReportPreview) handleMouseWheel(delta int16) {
	if delta > 0 {
		p.ZoomScale *= 1.15
		if p.ZoomScale > 4.0 {
			p.ZoomScale = 4.0
		}
	} else if delta < 0 {
		p.ZoomScale /= 1.15
		if p.ZoomScale < 0.25 {
			p.ZoomScale = 0.25
		}
	}
	procInvalidateRect.Call(p.Hwnd, 0, 0)
}

func (p *FastReportPreview) handleMouseDown(x, y int32) {
	// Hanya drag jika klik di area kanvas (di bawah toolbar 38px dan di atas status bar 22px)
	var rcClient winapi.RECT
	procGetClientRect.Call(p.Hwnd, uintptr(unsafe.Pointer(&rcClient)))
	h := rcClient.Bottom - rcClient.Top

	if y >= 38 && y < h-22 {
		p.IsDragging = true
		p.DragStartX = x
		p.DragStartY = y
		p.DragStartOffsetX = p.PanOffsetX
		p.DragStartOffsetY = p.PanOffsetY
		procSetCapture.Call(p.Hwnd)
		procSetCursor.Call(p.HCursorSizeAll)
	}
}

func (p *FastReportPreview) handleMouseMove(x, y int32) {
	if p.IsDragging {
		p.PanOffsetX = p.DragStartOffsetX + (x - p.DragStartX)
		p.PanOffsetY = p.DragStartOffsetY + (y - p.DragStartY)
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) handleMouseUp() {
	if p.IsDragging {
		p.IsDragging = false
		procReleaseCapture.Call()
		procSetCursor.Call(p.HCursorArrow)
	}
}

func (p *FastReportPreview) handleKeyDown(vk uint32) {
	switch vk {
	case 27: // ESC -> Tutup
		procDestroyWindow.Call(p.Hwnd)
	case 37: // Panah Kiri -> Halaman Sebelumnya
		p.prevPage()
	case 39: // Panah Kanan -> Halaman Selanjutnya
		p.nextPage()
	case 38: // Panah Atas -> Pan Up
		p.PanOffsetY += 40
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	case 40: // Panah Bawah -> Pan Down
		p.PanOffsetY -= 40
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) firstPage() {
	if p.CurrentPageIndex > 0 {
		p.CurrentPageIndex = 0
		p.PanOffsetY = 0
		p.updatePageLabel()
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) lastPage() {
	lastIdx := len(p.Documents) - 1
	if lastIdx < 0 {
		lastIdx = 0
	}
	if p.CurrentPageIndex < lastIdx {
		p.CurrentPageIndex = lastIdx
		p.PanOffsetY = 0
		p.updatePageLabel()
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) prevPage() {
	if p.CurrentPageIndex > 0 {
		p.CurrentPageIndex--
		p.PanOffsetY = 0
		p.updatePageLabel()
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) nextPage() {
	if p.CurrentPageIndex < len(p.Documents)-1 {
		p.CurrentPageIndex++
		p.PanOffsetY = 0
		p.updatePageLabel()
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) handleCommand(cmdId uint32) {
	switch cmdId {
	case ID_BTN_PRINT:
		p.onPrintClicked()
	case ID_BTN_SAVE:
		p.onSaveClicked()
	case ID_BTN_OPEN:
		p.onOpenClicked()
	case ID_BTN_FIND:
		p.onFindClicked()
	case ID_BTN_FIRST:
		p.firstPage()
	case ID_BTN_PREV:
		p.prevPage()
	case ID_BTN_NEXT:
		p.nextPage()
	case ID_BTN_LAST:
		p.lastPage()
	case ID_BTN_ZOOM_OUT:
		p.ZoomScale /= 1.25
		if p.ZoomScale < 0.25 {
			p.ZoomScale = 0.25
		}
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	case ID_BTN_ZOOM_IN:
		p.ZoomScale *= 1.25
		if p.ZoomScale > 4.0 {
			p.ZoomScale = 4.0
		}
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	case ID_BTN_ZOOM_100:
		p.ZoomScale = 1.0
		p.PanOffsetX = 0
		p.PanOffsetY = 0
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	case ID_BTN_FIT_WIDTH:
		p.fitWidth()
	case ID_BTN_FIT_PAGE:
		p.fitPage()
	case ID_BTN_CLOSE:
		procDestroyWindow.Call(p.Hwnd)
	}
}

func (p *FastReportPreview) fitWidth() {
	bmp := p.getCurrentBitmap()
	if bmp == nil || bmp.WidthPx <= 0 {
		return
	}
	var rcClient winapi.RECT
	procGetClientRect.Call(p.Hwnd, uintptr(unsafe.Pointer(&rcClient)))
	availW := float64(rcClient.Right-rcClient.Left) - 80.0
	if availW > 0 {
		p.ZoomScale = availW / float64(bmp.WidthPx)
		p.PanOffsetX = 0
		p.PanOffsetY = 0
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) fitPage() {
	bmp := p.getCurrentBitmap()
	if bmp == nil || bmp.HeightPx <= 0 {
		return
	}
	var rcClient winapi.RECT
	procGetClientRect.Call(p.Hwnd, uintptr(unsafe.Pointer(&rcClient)))
	// 38px toolbar + 22px status bar + 32px padding
	availH := float64(rcClient.Bottom-rcClient.Top) - 38.0 - 22.0 - 32.0
	if availH > 0 {
		p.ZoomScale = availH / float64(bmp.HeightPx)
		p.PanOffsetX = 0
		p.PanOffsetY = 0
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}
}

func (p *FastReportPreview) showMsg(text, caption string, uType uint32) {
	tW, _ := syscall.UTF16PtrFromString(text)
	cW, _ := syscall.UTF16PtrFromString(caption)
	procMessageBoxW.Call(p.Hwnd, uintptr(unsafe.Pointer(tW)), uintptr(unsafe.Pointer(cW)), uintptr(uType))
}

func (p *FastReportPreview) onPrintClicked() {
	go func() {
		doc := p.Documents[p.CurrentPageIndex]
		err := engine.PrintDocumentsWithDir(p.TargetPrinter, p.TemplateName, p.TemplateDir, []engine.DocumentData{doc})
		if err != nil {
			p.showMsg(fmt.Sprintf("Gagal mencetak dokumen:\n%v", err), "Error Cetak", MB_OK|MB_ICONERROR)
		} else {
			p.showMsg(fmt.Sprintf("✅ Dokumen berhasil dikirim ke printer:\n%s\n\nTemplate: %s\nHalaman: %d dari %d",
				p.TargetPrinter, p.TemplateName, p.CurrentPageIndex+1, len(p.Documents)),
				"Cetak Berhasil", MB_OK|MB_ICONINFORMATION)
		}
	}()
}

func (p *FastReportPreview) onSaveClicked() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		bmp := p.getCurrentBitmap()
		if bmp == nil {
			return
		}

		filter := []uint16{}
		filter = append(filter, syscall.StringToUTF16("PNG Image (*.png)")...)
		filter = append(filter, syscall.StringToUTF16("*.png")...)
		filter = append(filter, syscall.StringToUTF16("Semua File (*.*)")...)
		filter = append(filter, syscall.StringToUTF16("*.*")...)
		filter = append(filter, 0)

		defaultName := fmt.Sprintf("%s_hal%d.png", p.TemplateName, p.CurrentPageIndex+1)
		fileBuf := make([]uint16, 1024)
		copy(fileBuf, syscall.StringToUTF16(defaultName))

		titleW, _ := syscall.UTF16PtrFromString("Simpan Struk Kasir / Dokumen sebagai Gambar PNG")
		defExtW, _ := syscall.UTF16PtrFromString("png")

		var ofn OPENFILENAMEW
		ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
		ofn.HwndOwner = p.Hwnd
		ofn.LpstrFilter = &filter[0]
		ofn.LpstrFile = &fileBuf[0]
		ofn.NMaxFile = 1024
		ofn.LpstrTitle = titleW
		ofn.LpstrDefExt = defExtW
		ofn.Flags = OFN_OVERWRITEPROMPT | OFN_PATHMUSTEXIST | OFN_EXPLORER

		r, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
		if r == 0 {
			return // Dibatalkan oleh pengguna
		}

		savePath := strings.TrimSpace(syscall.UTF16ToString(fileBuf))
		if savePath == "" {
			return
		}

		// Konversi BGRA buffer ke RGBA
		rgba := image.NewRGBA(image.Rect(0, 0, int(bmp.WidthPx), int(bmp.HeightPx)))
		total := len(bmp.Pixels)
		for i := 0; i < total; i += 4 {
			rgba.Pix[i] = bmp.Pixels[i+2]   // R
			rgba.Pix[i+1] = bmp.Pixels[i+1] // G
			rgba.Pix[i+2] = bmp.Pixels[i]   // B
			rgba.Pix[i+3] = 255            // Opaque
		}

		f, err := os.Create(savePath)
		if err != nil {
			p.showMsg(fmt.Sprintf("Gagal membuat file:\n%v", err), "Error Simpan", MB_OK|MB_ICONERROR)
			return
		}
		defer f.Close()

		if err := png.Encode(f, rgba); err != nil {
			p.showMsg(fmt.Sprintf("Gagal menyimpan PNG:\n%v", err), "Error Simpan", MB_OK|MB_ICONERROR)
			return
		}

		p.showMsg(fmt.Sprintf("✅ Gambar struk kasir berhasil disimpan!\n\nLokasi: %s\nResolusi: %d x %d px (203 DPI)",
			savePath, bmp.WidthPx, bmp.HeightPx), "Simpan Berhasil", MB_OK|MB_ICONINFORMATION)
	}()
}

func (p *FastReportPreview) onOpenClicked() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		filter := []uint16{}
		filter = append(filter, syscall.StringToUTF16("File Template JSON (*.json)")...)
		filter = append(filter, syscall.StringToUTF16("*.json")...)
		filter = append(filter, syscall.StringToUTF16("Semua File (*.*)")...)
		filter = append(filter, syscall.StringToUTF16("*.*")...)
		filter = append(filter, 0)

		fileBuf := make([]uint16, 1024)
		titleW, _ := syscall.UTF16PtrFromString("Buka File Template JSON Lainnya")
		defExtW, _ := syscall.UTF16PtrFromString("json")

		var ofn OPENFILENAMEW
		ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
		ofn.HwndOwner = p.Hwnd
		ofn.LpstrFilter = &filter[0]
		ofn.LpstrFile = &fileBuf[0]
		ofn.NMaxFile = 1024
		ofn.LpstrTitle = titleW
		ofn.LpstrDefExt = defExtW
		ofn.Flags = OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER

		r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
		if r == 0 {
			return
		}

		selectedPath := strings.TrimSpace(syscall.UTF16ToString(fileBuf))
		if selectedPath == "" {
			return
		}

		newDir := filepath.Dir(selectedPath)
		newTplName := strings.TrimSuffix(filepath.Base(selectedPath), filepath.Ext(selectedPath))

		newTpl, err := engine.LoadTemplateWithDir(newTplName, newDir)
		if err != nil {
			p.showMsg(fmt.Sprintf("Gagal membaca template:\n%v", err), "Error Buka", MB_OK|MB_ICONERROR)
			return
		}

		dpi, _ := engine.GetTemplateDPI(p.TargetPrinter)
		newBitmaps := make([]*engine.RenderedBitmap, 0, len(p.Documents))
		for _, doc := range p.Documents {
			bmp, errBmp := engine.RenderPreviewBitmap(newTpl, doc, dpi)
			if errBmp != nil {
				p.showMsg(fmt.Sprintf("Gagal render preview template baru:\n%v", errBmp), "Error Render", MB_OK|MB_ICONERROR)
				return
			}
			newBitmaps = append(newBitmaps, bmp)
		}

		p.mu.Lock()
		p.Template = newTpl
		p.TemplateName = newTplName
		p.TemplateDir = newDir
		p.Bitmaps = newBitmaps
		p.CurrentPageIndex = 0
		p.PanOffsetX = 0
		p.PanOffsetY = 0
		p.mu.Unlock()

		newTitle := fmt.Sprintf("🖨️ FastReport Print Preview - [%s] (Knitto Windows Native GDI)", newTplName)
		newTitleW, _ := syscall.UTF16PtrFromString(newTitle)
		procSetWindowTextW.Call(p.Hwnd, uintptr(unsafe.Pointer(newTitleW)))

		p.updatePageLabel()
		procInvalidateRect.Call(p.Hwnd, 0, 0)
	}()
}

func (p *FastReportPreview) onFindClicked() {
	if p.HEditFind == 0 {
		return
	}
	buf := make([]uint16, 256)
	procGetWindowTextW.Call(p.HEditFind, uintptr(unsafe.Pointer(&buf[0])), 256)
	searchKeyword := strings.TrimSpace(syscall.UTF16ToString(buf))
	if searchKeyword == "" {
		p.showMsg("Masukkan kata kunci pencarian pada kotak di atas!", "Pencarian Teks", MB_OK|MB_ICONWARNING)
		return
	}

	doc := p.Documents[p.CurrentPageIndex]
	matchCount := 0
	lowerKeyword := strings.ToLower(searchKeyword)

	for _, v := range doc {
		switch val := v.(type) {
		case string:
			if strings.Contains(strings.ToLower(val), lowerKeyword) {
				matchCount++
			}
		case []interface{}:
			for _, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					for _, itemVal := range itemMap {
						if s, ok := itemVal.(string); ok && strings.Contains(strings.ToLower(s), lowerKeyword) {
							matchCount++
						}
					}
				}
			}
		case []map[string]interface{}:
			for _, itemMap := range val {
				for _, itemVal := range itemMap {
					if s, ok := itemVal.(string); ok && strings.Contains(strings.ToLower(s), lowerKeyword) {
						matchCount++
					}
				}
			}
		}
	}

	if matchCount > 0 {
		p.showMsg(fmt.Sprintf("🔍 Ditemukan %d kecocokan untuk teks \"%s\" pada halaman ini.", matchCount, searchKeyword), "Hasil Pencarian", MB_OK|MB_ICONINFORMATION)
	} else {
		p.showMsg(fmt.Sprintf("Teks \"%s\" tidak ditemukan pada halaman ini.", searchKeyword), "Hasil Pencarian", MB_OK|MB_ICONINFORMATION)
	}
}
