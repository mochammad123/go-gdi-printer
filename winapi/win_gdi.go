package winapi

import (
	"fmt"
	"image"
	"strings"
	"syscall"
	"unsafe"
)

var (
	ModWinspool = syscall.NewLazyDLL("winspool.drv")
	ModGdi32    = syscall.NewLazyDLL("gdi32.dll")
	ModUser32   = syscall.NewLazyDLL("user32.dll")
	ModKernel32 = syscall.NewLazyDLL("kernel32.dll")

	ProcGetDefaultPrinterW  = ModWinspool.NewProc("GetDefaultPrinterW")
	ProcSetDefaultPrinterW  = ModWinspool.NewProc("SetDefaultPrinterW")
	ProcEnumPrintersW       = ModWinspool.NewProc("EnumPrintersW")
	ProcOpenPrinterW        = ModWinspool.NewProc("OpenPrinterW")
	ProcClosePrinter        = ModWinspool.NewProc("ClosePrinter")
	ProcDocumentPropertiesW = ModWinspool.NewProc("DocumentPropertiesW")
	ProcStartDocPrinterW    = ModWinspool.NewProc("StartDocPrinterW")
	ProcStartPagePrinterW   = ModWinspool.NewProc("StartPagePrinterW")
	ProcWritePrinter        = ModWinspool.NewProc("WritePrinter")
	ProcEndPagePrinterW     = ModWinspool.NewProc("EndPagePrinterW")
	ProcEndDocPrinterW      = ModWinspool.NewProc("EndDocPrinterW")

	ProcCreateDCW          = ModGdi32.NewProc("CreateDCW")
	ProcDeleteDC           = ModGdi32.NewProc("DeleteDC")
	ProcStartDocW          = ModGdi32.NewProc("StartDocW")
	ProcEndDoc             = ModGdi32.NewProc("EndDoc")
	ProcStartPage          = ModGdi32.NewProc("StartPage")
	ProcEndPage            = ModGdi32.NewProc("EndPage")
	ProcGetDeviceCaps      = ModGdi32.NewProc("GetDeviceCaps")
	ProcSetBkMode          = ModGdi32.NewProc("SetBkMode")
	ProcSetBkColor         = ModGdi32.NewProc("SetBkColor")
	ProcSetTextColor       = ModGdi32.NewProc("SetTextColor")
	ProcCreateFontW        = ModGdi32.NewProc("CreateFontW")
	ProcSelectObject       = ModGdi32.NewProc("SelectObject")
	ProcDeleteObject       = ModGdi32.NewProc("DeleteObject")
	ProcTextOutW           = ModGdi32.NewProc("TextOutW")
	ProcSetTextAlign       = ModGdi32.NewProc("SetTextAlign")
	ProcMoveToEx           = ModGdi32.NewProc("MoveToEx")
	ProcLineTo             = ModGdi32.NewProc("LineTo")
	ProcCreatePen          = ModGdi32.NewProc("CreatePen")
	ProcCreateSolidBrush   = ModGdi32.NewProc("CreateSolidBrush")
	ProcGetStockObject     = ModGdi32.NewProc("GetStockObject")
	ProcStretchDIBits      = ModGdi32.NewProc("StretchDIBits")
	ProcCreateCompatibleDC = ModGdi32.NewProc("CreateCompatibleDC")
	ProcCreateDIBSection   = ModGdi32.NewProc("CreateDIBSection")
	ProcRectangle          = ModGdi32.NewProc("Rectangle")
	ProcCreateCompatibleBitmap = ModGdi32.NewProc("CreateCompatibleBitmap")
	ProcBitBlt                 = ModGdi32.NewProc("BitBlt")
	ProcSetStretchBltMode      = ModGdi32.NewProc("SetStretchBltMode")
	ProcSetBrushOrgEx          = ModGdi32.NewProc("SetBrushOrgEx")
	ProcRoundRect              = ModGdi32.NewProc("RoundRect")

	ProcFillRect       = ModUser32.NewProc("FillRect")
	ProcDrawTextW      = ModUser32.NewProc("DrawTextW")
	ProcBeginPaint     = ModUser32.NewProc("BeginPaint")
	ProcEndPaint       = ModUser32.NewProc("EndPaint")
	ProcGetClientRect  = ModUser32.NewProc("GetClientRect")
	ProcInvalidateRect = ModUser32.NewProc("InvalidateRect")
	ProcSetCapture     = ModUser32.NewProc("SetCapture")
	ProcReleaseCapture = ModUser32.NewProc("ReleaseCapture")
	ProcLoadCursorW    = ModUser32.NewProc("LoadCursorW")
	ProcSetCursor      = ModUser32.NewProc("SetCursor")

	ProcCreateMutexW          = ModKernel32.NewProc("CreateMutexW")
	ProcGetLastError          = ModKernel32.NewProc("GetLastError")
	ProcCloseHandle           = ModKernel32.NewProc("CloseHandle")
	ProcGetConsoleProcessList = ModKernel32.NewProc("GetConsoleProcessList")
)

const (
	LOGPIXELSX = 88
	LOGPIXELSY = 90

	TRANSPARENT = 1
	BLACK_BRUSH = 4
	WHITE_BRUSH = 0
	NULL_BRUSH  = 5

	FW_NORMAL = 400
	FW_BOLD   = 700

	PS_SOLID = 0
	PS_DASH  = 1

	TA_LEFT   = 0
	TA_RIGHT  = 2
	TA_CENTER = 6

	DT_TOP         = 0x00000000
	DT_LEFT        = 0x00000000
	DT_CENTER      = 0x00000001
	DT_RIGHT       = 0x00000002
	DT_WORDBREAK   = 0x00000010
	DT_CALCRECT    = 0x00000400
	DT_NOPREFIX    = 0x00000800
	DT_EDITCONTROL = 0x00002000

	DIB_RGB_COLORS = 0
	SRCCOPY        = 0x00CC0020
	BI_RGB         = 0

	PRINTER_ENUM_LOCAL       = 0x00000002
	PRINTER_ENUM_CONNECTIONS = 0x00000004

	DM_ORIENTATION = 0x00000001
	DM_PAPERSIZE   = 0x00000002
	DM_PAPERLENGTH = 0x00000004
	DM_PAPERWIDTH  = 0x00000008

	DMPAPER_USER  = 256
	DM_IN_BUFFER  = 8
	DM_OUT_BUFFER = 2

	WM_PAINT        = 0x000F
	WM_ERASEBKGND   = 0x0014
	WM_SIZE         = 0x0005
	WM_MOUSEMOVE    = 0x0200
	WM_LBUTTONDOWN  = 0x0201
	WM_LBUTTONUP    = 0x0202
	WM_RBUTTONDOWN  = 0x0204
	WM_MOUSEWHEEL   = 0x020A
	WM_DRAWITEM     = 0x002B
	WM_SETCURSOR    = 0x0020
	WM_KEYDOWN      = 0x0100

	BS_OWNERDRAW = 0x0000000B

	ODS_SELECTED = 0x0001
	ODS_GRAYED   = 0x0002
	ODS_DISABLED = 0x0004
	ODS_FOCUS    = 0x0010
	ODS_HOTLIGHT = 0x0040

	IDC_ARROW   = 32512
	IDC_HAND    = 32649
	IDC_SIZEALL = 32646

	STRETCH_HALFTONE = 4

	WS_OVERLAPPEDWINDOW = 0x00CF0000
)

type DOCINFOW struct {
	CbSize       int32
	LpszDocName  *uint16
	LpszOutput   *uint16
	LpszDatatype *uint16
	FwType       uint32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     RECT
	ItemData   uintptr
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type PRINTER_INFO_4 struct {
	PPrinterName *uint16
	PServerName  *uint16
	Attributes   uint32
}

// GetDefaultPrinter returns default Windows printer name
func GetDefaultPrinter() (string, error) {
	var size uint32 = 0
	ProcGetDefaultPrinterW.Call(0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return "", fmt.Errorf("tidak ada printer default yang terpasang")
	}

	buf := make([]uint16, size)
	r1, _, err := ProcGetDefaultPrinterW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r1 == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}

// SetDefaultPrinter sets the Windows system-wide default printer
func SetDefaultPrinter(printerName string) error {
	pName, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}
	r1, _, err := ProcSetDefaultPrinterW.Call(uintptr(unsafe.Pointer(pName)))
	if r1 == 0 {
		return fmt.Errorf("gagal mengatur printer default: %v", err)
	}
	return nil
}
// DOC_INFO_1W structure for StartDocPrinterW
type DOC_INFO_1W struct {
	PDocName    *uint16
	POutputFile *uint16
	PDatatype   *uint16
}

// WriteRawPrinter sends raw bytes directly to the Windows print spooler
func WriteRawPrinter(printerName string, docName string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	pNameW, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}
	var hPrinter uintptr
	r, _, err := ProcOpenPrinterW.Call(
		uintptr(unsafe.Pointer(pNameW)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r == 0 || hPrinter == 0 {
		return fmt.Errorf("gagal membuka printer untuk raw write (%s): %v", printerName, err)
	}
	defer ProcClosePrinter.Call(hPrinter)

	pDocNameW, _ := syscall.UTF16PtrFromString(docName)
	pDataTypeW, _ := syscall.UTF16PtrFromString("RAW")
	di := DOC_INFO_1W{
		PDocName:    pDocNameW,
		POutputFile: nil,
		PDatatype:   pDataTypeW,
	}
	jobId, _, err := ProcStartDocPrinterW.Call(hPrinter, 1, uintptr(unsafe.Pointer(&di)))
	if jobId == 0 {
		return fmt.Errorf("StartDocPrinter gagal: %v", err)
	}
	defer ProcEndDocPrinterW.Call(hPrinter)

	ProcStartPagePrinterW.Call(hPrinter)
	var written uint32
	ProcWritePrinter.Call(
		hPrinter,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	ProcEndPagePrinterW.Call(hPrinter)
	return nil
}

// CutPaper sends ESC/POS feed & cut commands to receipt / dot matrix printers
func CutPaper(printerName string) error {
	// ESC d 3 (feed 3 baris melewati pisau) + GS V 1 (partial cut, menyisakan 1 titik agar kertas rapi)
	cmd := []byte{0x1b, 0x64, 0x03, 0x1d, 0x56, 0x01}
	return WriteRawPrinter(printerName, "AutoCut", cmd)
}


// ListInstalledPrinters returns a list of installed printers
func ListInstalledPrinters() ([]string, error) {
	var bytesNeeded, numPrinters uint32
	flags := uint32(PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS)

	ProcEnumPrintersW.Call(
		uintptr(flags),
		0,
		4,
		0,
		0,
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&numPrinters)),
	)

	if bytesNeeded == 0 {
		return []string{}, nil
	}

	buf := make([]byte, bytesNeeded)
	r1, _, err := ProcEnumPrintersW.Call(
		uintptr(flags),
		0,
		4,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(bytesNeeded),
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&numPrinters)),
	)
	if r1 == 0 {
		return nil, err
	}

	printers := make([]string, 0, numPrinters)
	pSlice := (*[1024]PRINTER_INFO_4)(unsafe.Pointer(&buf[0]))[:numPrinters:numPrinters]
	for _, p := range pSlice {
		printers = append(printers, syscall.UTF16ToString((*[512]uint16)(unsafe.Pointer(p.PPrinterName))[:]))
	}
	return printers, nil
}

// GDIPrinter manages raw GDI Device Context for sharp printing
type GDIPrinter struct {
	HDC     uintptr
	DpiX    int32
	DpiY    int32
	BlackBr uintptr
}

// NewGDIPrinter creates a Printer Device Context with default driver settings
func NewGDIPrinter(printerName string) (*GDIPrinter, error) {
	return NewGDIPrinterWithPaperSize(printerName, 0, 0)
}

// NewGDIPrinterWithPaperSize creates a Printer Device Context with explicit paper width and height (in mm)
func NewGDIPrinterWithPaperSize(printerName string, widthMm, heightMm float64) (*GDIPrinter, error) {
	pNameW, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return nil, err
	}

	var hdc uintptr

	// Configure DEVMODEW if width and height are provided
	if widthMm > 0 && heightMm > 0 {
		var hPrinter uintptr
		rOpen, _, _ := ProcOpenPrinterW.Call(
			uintptr(unsafe.Pointer(pNameW)),
			uintptr(unsafe.Pointer(&hPrinter)),
			0,
		)

		if rOpen != 0 && hPrinter != 0 {
			defer ProcClosePrinter.Call(hPrinter)

			// Get DEVMODE size
			size, _, _ := ProcDocumentPropertiesW.Call(
				0,
				hPrinter,
				uintptr(unsafe.Pointer(pNameW)),
				0,
				0,
				0,
			)

			if int32(size) >= 84 {
				buf := make([]byte, size)
				// Fetch current DEVMODE
				rProp, _, _ := ProcDocumentPropertiesW.Call(
					0,
					hPrinter,
					uintptr(unsafe.Pointer(pNameW)),
					uintptr(unsafe.Pointer(&buf[0])),
					0,
					uintptr(DM_OUT_BUFFER),
				)

				if int32(rProp) >= 0 {
					pFields := (*uint32)(unsafe.Pointer(&buf[72]))
					pPaperSize := (*int16)(unsafe.Pointer(&buf[78]))
					pPaperLength := (*int16)(unsafe.Pointer(&buf[80]))
					pPaperWidth := (*int16)(unsafe.Pointer(&buf[82]))

					*pFields |= (DM_PAPERSIZE | DM_PAPERLENGTH | DM_PAPERWIDTH)
					*pPaperSize = DMPAPER_USER           // 256 (User Defined)
					*pPaperLength = int16(heightMm * 10) // in tenths of a mm (e.g. 30mm -> 300)
					*pPaperWidth = int16(widthMm * 10)   // in tenths of a mm (e.g. 80mm -> 800)

					// Validate modified DEVMODE
					ProcDocumentPropertiesW.Call(
						0,
						hPrinter,
						uintptr(unsafe.Pointer(pNameW)),
						uintptr(unsafe.Pointer(&buf[0])),
						uintptr(unsafe.Pointer(&buf[0])),
						uintptr(DM_IN_BUFFER|DM_OUT_BUFFER),
					)

					// Create DC with custom DEVMODE
					resHdc, _, _ := ProcCreateDCW.Call(
						0,
						uintptr(unsafe.Pointer(pNameW)),
						0,
						uintptr(unsafe.Pointer(&buf[0])),
					)
					hdc = resHdc
				}
			}
		}
	}

	// Fallback to default CreateDCW if hdc is still 0
	if hdc == 0 {
		resHdc, _, _ := ProcCreateDCW.Call(0, uintptr(unsafe.Pointer(pNameW)), 0, 0)
		hdc = resHdc
	}

	if hdc == 0 {
		return nil, fmt.Errorf("gagal membuka printer DC: %s", printerName)
	}

	dpiX, _, _ := ProcGetDeviceCaps.Call(hdc, LOGPIXELSX)
	dpiY, _, _ := ProcGetDeviceCaps.Call(hdc, LOGPIXELSY)
	blackBrush, _, _ := ProcGetStockObject.Call(BLACK_BRUSH)

	// Set text background to transparent
	ProcSetBkMode.Call(hdc, TRANSPARENT)

	return &GDIPrinter{
		HDC:     hdc,
		DpiX:    int32(dpiX),
		DpiY:    int32(dpiY),
		BlackBr: blackBrush,
	}, nil
}

func (p *GDIPrinter) Close() {
	if p.HDC != 0 {
		ProcDeleteDC.Call(p.HDC)
		p.HDC = 0
	}
}

// Convert millimeters to device pixels based on printer DPI
func (p *GDIPrinter) MmToPxX(mm float64) int32 {
	return int32(mm * float64(p.DpiX) / 25.4)
}

func (p *GDIPrinter) MmToPxY(mm float64) int32 {
	return int32(mm * float64(p.DpiY) / 25.4)
}

func (p *GDIPrinter) PtToFontHeight(pt float64) int32 {
	return -int32(pt * float64(p.DpiY) / 72.0)
}

func (p *GDIPrinter) StartDoc(docName string) error {
	docNameW, _ := syscall.UTF16PtrFromString(docName)
	di := DOCINFOW{
		CbSize:      int32(unsafe.Sizeof(DOCINFOW{})),
		LpszDocName: docNameW,
	}
	r1, _, err := ProcStartDocW.Call(p.HDC, uintptr(unsafe.Pointer(&di)))
	if int32(r1) <= 0 {
		return fmt.Errorf("StartDoc gagal: %v", err)
	}
	return nil
}

func (p *GDIPrinter) EndDoc() {
	ProcEndDoc.Call(p.HDC)
}

func (p *GDIPrinter) StartPage() {
	ProcStartPage.Call(p.HDC)
}

func (p *GDIPrinter) EndPage() {
	ProcEndPage.Call(p.HDC)
}

func (p *GDIPrinter) DrawText(xMm, yMm float64, text string, hFont uintptr, align string) {
	if hFont != 0 {
		ProcSelectObject.Call(p.HDC, hFont)
	}

	alignMode := uintptr(TA_LEFT)
	switch strings.ToLower(strings.TrimSpace(align)) {
	case "right":
		alignMode = uintptr(TA_RIGHT)
	case "center":
		alignMode = uintptr(TA_CENTER)
	}
	ProcSetTextAlign.Call(p.HDC, alignMode)

	tW, _ := syscall.UTF16FromString(text)
	x := p.MmToPxX(xMm)
	y := p.MmToPxY(yMm)
	ProcTextOutW.Call(p.HDC, uintptr(x), uintptr(y), uintptr(unsafe.Pointer(&tW[0])), uintptr(len(tW)-1))

	// Reset alignment kembali ke TA_LEFT
	ProcSetTextAlign.Call(p.HDC, uintptr(TA_LEFT))
}

// MeasureTextBoxPx calculates the height needed to display text inside width wPx using hFont
func (p *GDIPrinter) MeasureTextBoxPx(text string, hFont uintptr, wPx int32) int32 {
	if wPx <= 0 {
		wPx = 1
	}
	oldFont, _, _ := ProcSelectObject.Call(p.HDC, hFont)
	defer ProcSelectObject.Call(p.HDC, oldFont)

	rc := RECT{
		Left:   0,
		Top:    0,
		Right:  wPx,
		Bottom: 0,
	}
	tW, _ := syscall.UTF16FromString(text)
	flags := uintptr(DT_WORDBREAK | DT_EDITCONTROL | DT_NOPREFIX | DT_CALCRECT)
	ProcDrawTextW.Call(
		p.HDC,
		uintptr(unsafe.Pointer(&tW[0])),
		uintptr(len(tW)-1),
		uintptr(unsafe.Pointer(&rc)),
		flags,
	)
	return rc.Bottom - rc.Top
}

// DrawTextBox draws multiline text inside rect rc with word-wrapping and alignment
func (p *GDIPrinter) DrawTextBox(text string, hFont uintptr, rc RECT, align string) {
	if hFont != 0 {
		oldFont, _, _ := ProcSelectObject.Call(p.HDC, hFont)
		defer ProcSelectObject.Call(p.HDC, oldFont)
	}

	flags := uintptr(DT_WORDBREAK | DT_EDITCONTROL | DT_NOPREFIX)
	switch strings.ToLower(strings.TrimSpace(align)) {
	case "center":
		flags |= uintptr(DT_CENTER)
	case "right":
		flags |= uintptr(DT_RIGHT)
	default:
		flags |= uintptr(DT_LEFT)
	}

	tW, _ := syscall.UTF16FromString(text)
	ProcDrawTextW.Call(
		p.HDC,
		uintptr(unsafe.Pointer(&tW[0])),
		uintptr(len(tW)-1),
		uintptr(unsafe.Pointer(&rc)),
		flags,
	)
}

func (p *GDIPrinter) FillRectMm(xMm, yMm, wMm, hMm float64) {
	rc := RECT{
		Left:   p.MmToPxX(xMm),
		Top:    p.MmToPxY(yMm),
		Right:  p.MmToPxX(xMm + wMm),
		Bottom: p.MmToPxY(yMm + hMm),
	}
	ProcFillRect.Call(p.HDC, uintptr(unsafe.Pointer(&rc)), p.BlackBr)
}

func (p *GDIPrinter) DrawLine(x1Mm, y1Mm, x2Mm, y2Mm float64, isDashed bool) {
	penStyle := int32(PS_SOLID)
	if isDashed {
		penStyle = PS_DASH
	}
	hPen, _, _ := ProcCreatePen.Call(uintptr(penStyle), 1, 0)
	oldPen, _, _ := ProcSelectObject.Call(p.HDC, hPen)

	x1 := p.MmToPxX(x1Mm)
	y1 := p.MmToPxY(y1Mm)
	x2 := p.MmToPxX(x2Mm)
	y2 := p.MmToPxY(y2Mm)

	ProcMoveToEx.Call(p.HDC, uintptr(x1), uintptr(y1), 0)
	ProcLineTo.Call(p.HDC, uintptr(x2), uintptr(y2))

	ProcSelectObject.Call(p.HDC, oldPen)
	ProcDeleteObject.Call(hPen)
}

func (p *GDIPrinter) CreateFont(name string, ptSize float64, bold bool) uintptr {
	nameW, _ := syscall.UTF16PtrFromString(name)
	weight := int32(FW_NORMAL)
	if bold {
		weight = FW_BOLD
	}
	h := p.PtToFontHeight(ptSize)

	// Untuk printer Dot Matrix (resolusi rendah <= 180 DPI seperti TM-U220):
	// Gunakan NONANTIALIASED_QUALITY (3) agar hasil cetak 100% hitam solid murni tanpa semut abu-abu.
	// Untuk printer thermal (203+ DPI), gunakan ANTIALIASED_QUALITY (4).
	quality := uintptr(4)
	if p.DpiX <= 180 || p.DpiY <= 180 {
		quality = 3 // NONANTIALIASED_QUALITY
	}

	hFont, _, _ := ProcCreateFontW.Call(
		uintptr(h),
		0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		1,       // DEFAULT_CHARSET
		0, 0, quality, 0,
		uintptr(unsafe.Pointer(nameW)),
	)
	return hFont
}

func (p *GDIPrinter) DeleteFont(hFont uintptr) {
	if hFont != 0 {
		ProcDeleteObject.Call(hFont)
	}
}

// DrawImage renders an image.Image to the printer DC scaled to millimeters (wMm x hMm)
func (p *GDIPrinter) DrawImage(img image.Image, xMm, yMm, wMm, hMm float64) error {
	if img == nil {
		return fmt.Errorf("gambar bernilai nil")
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return fmt.Errorf("dimensi gambar tidak valid: %dx%d", w, h)
	}

	// Konversi pixel image ke 32-bit BGRA (Windows DIB format)
	// Alpha blending dengan warna putih kertas agar gambar transparan (PNG) tidak mencetak kotak hitam
	buf := make([]byte, w*h*4)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA() // 0..0xffff
			if a == 0 {
				buf[idx] = 255   // B
				buf[idx+1] = 255 // G
				buf[idx+2] = 255 // R
				buf[idx+3] = 0
			} else {
				alpha := float64(a) / 65535.0
				r8 := byte(float64(r>>8)*alpha + 255.0*(1.0-alpha))
				g8 := byte(float64(g>>8)*alpha + 255.0*(1.0-alpha))
				b8 := byte(float64(b>>8)*alpha + 255.0*(1.0-alpha))

				buf[idx] = b8
				buf[idx+1] = g8
				buf[idx+2] = r8
				buf[idx+3] = 0
			}
			idx += 4
		}
	}

	bmi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(w),
			BiHeight:      -int32(h), // Nilai negatif menandakan top-down DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
			BiSizeImage:   uint32(len(buf)),
		},
	}

	xDest := p.MmToPxX(xMm)
	yDest := p.MmToPxY(yMm)
	destW := p.MmToPxX(wMm)
	destH := p.MmToPxY(hMm)

	r1, _, err := ProcStretchDIBits.Call(
		p.HDC,
		uintptr(xDest),
		uintptr(yDest),
		uintptr(destW),
		uintptr(destH),
		0,
		0,
		uintptr(w),
		uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		SRCCOPY,
	)
	if int32(r1) <= 0 {
		return fmt.Errorf("StretchDIBits gagal: %v", err)
	}
	return nil
}

const ERROR_ALREADY_EXISTS = 183

// AcquireSingleInstance attempts to acquire a named Windows mutex.
// Returns (handle, isAlreadyRunning, error)
func AcquireSingleInstance(name string) (uintptr, bool, error) {
	utf16Name, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, false, err
	}
	hMutex, _, callErr := ProcCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16Name)))
	if hMutex == 0 {
		return 0, false, fmt.Errorf("gagal membuat mutex sistem: %v", callErr)
	}
	if errno, ok := callErr.(syscall.Errno); ok && errno == ERROR_ALREADY_EXISTS {
		return hMutex, true, nil
	}
	return hMutex, false, nil
}

// ReleaseMutexHandle releases the named mutex
func ReleaseMutexHandle(hMutex uintptr) {
	if hMutex != 0 {
		ProcCloseHandle.Call(hMutex)
	}
}

// IsLaunchedFromExplorer checks if the console was created exclusively for this process (e.g. double click)
func IsLaunchedFromExplorer() bool {
	var pids [2]uint32
	count, _, _ := ProcGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	return count == 1
}

