package engine

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"unsafe"

	"print-service/winapi"
)

// RenderedBitmap stores the in-memory 32-bit DIB bitmap rendered by GDI
type RenderedBitmap struct {
	WidthPx  int32
	HeightPx int32
	BMI      winapi.BITMAPINFO
	Pixels   []byte // Top-down 32-bit BGRA pixel data
}

// RenderPreviewBitmap renders a single document into raw BGRA pixel buffer using Windows GDI Memory DC
func RenderPreviewBitmap(tpl *DocumentTemplate, doc DocumentData, dpi int32) (*RenderedBitmap, error) {
	if dpi <= 0 {
		dpi = 203 // Default 203 DPI (8 dots/mm)
	}

	heightMm := tpl.HeightMm
	if tpl.AutoHeight || heightMm <= 0 {
		if dynH := CalculateDocumentHeightMm(tpl, doc); dynH > 0 {
			heightMm = dynH
		}
	}

	wPx := int32(tpl.WidthMm * float64(dpi) / 25.4)
	hPx := int32(heightMm * float64(dpi) / 25.4)
	if wPx <= 0 || hPx <= 0 {
		return nil, fmt.Errorf("ukuran template tidak valid untuk preview: %dx%d px", wPx, hPx)
	}

	// 1. Buat Memory DC
	hdcMem, _, err := winapi.ProcCreateCompatibleDC.Call(0)
	if hdcMem == 0 {
		return nil, fmt.Errorf("gagal membuat Compatible DC: %v", err)
	}
	defer winapi.ProcDeleteDC.Call(hdcMem)

	// 2. Buat 32-bit DIB Section agar kita punya pointer langsung ke buffer pixel
	bmi := winapi.BITMAPINFO{
		BmiHeader: winapi.BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(winapi.BITMAPINFOHEADER{})),
			BiWidth:       wPx,
			BiHeight:      -hPx, // top-down DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: winapi.BI_RGB,
			BiSizeImage:   uint32(wPx * hPx * 4),
		},
	}

	var pvBits unsafe.Pointer
	hBmp, _, err := winapi.ProcCreateDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bmi)),
		winapi.DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pvBits)),
		0,
		0,
	)
	if hBmp == 0 || pvBits == nil {
		return nil, fmt.Errorf("gagal membuat DIBSection untuk preview: %v", err)
	}
	defer winapi.ProcDeleteObject.Call(hBmp)

	// Select DIB ke Memory DC
	oldBmp, _, _ := winapi.ProcSelectObject.Call(hdcMem, hBmp)
	defer winapi.ProcSelectObject.Call(hdcMem, oldBmp)

	// 3. Clear background menjadi putih bersih (kertas)
	totalBytes := int(wPx * hPx * 4)
	rawSlice := unsafe.Slice((*byte)(pvBits), totalBytes)
	for i := 0; i < totalBytes; i += 4 {
		rawSlice[i] = 255   // B
		rawSlice[i+1] = 255 // G
		rawSlice[i+2] = 255 // R
		rawSlice[i+3] = 255 // A
	}

	// Set background mode transparent untuk text
	winapi.ProcSetBkMode.Call(hdcMem, winapi.TRANSPARENT)

	blackBrush, _, _ := winapi.ProcGetStockObject.Call(winapi.BLACK_BRUSH)

	// 4. Bungkus dalam GDIPrinter (sama persis dengan printer fisik)
	gdi := &winapi.GDIPrinter{
		HDC:     hdcMem,
		DpiX:    dpi,
		DpiY:    dpi,
		BlackBr: blackBrush,
	}

	// 5. Render template menggunakan engine GDI yang sama persis
	RenderTemplate(gdi, tpl, doc)

	// 6. Copy pixel buffer agar mandiri dan bisa dipakai ulang tanpa ketergantungan DIB handle
	pixelCopy := make([]byte, totalBytes)
	copy(pixelCopy, rawSlice)

	return &RenderedBitmap{
		WidthPx:  wPx,
		HeightPx: hPx,
		BMI:      bmi,
		Pixels:   pixelCopy,
	}, nil
}

// RenderPreview renders a single document into a base64 PNG data URI string using Windows GDI Memory DC
func RenderPreview(tpl *DocumentTemplate, doc DocumentData, dpi int32) (string, error) {
	bmp, err := RenderPreviewBitmap(tpl, doc, dpi)
	if err != nil {
		return "", err
	}

	totalBytes := len(bmp.Pixels)
	// Konversi BGRA buffer ke Go image.RGBA
	rgbaImg := image.NewRGBA(image.Rect(0, 0, int(bmp.WidthPx), int(bmp.HeightPx)))
	for i := 0; i < totalBytes; i += 4 {
		rgbaImg.Pix[i] = bmp.Pixels[i+2]   // R
		rgbaImg.Pix[i+1] = bmp.Pixels[i+1] // G
		rgbaImg.Pix[i+2] = bmp.Pixels[i]   // B
		rgbaImg.Pix[i+3] = 255            // Alpha opaque
	}

	// Encode ke PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgbaImg); err != nil {
		return "", fmt.Errorf("gagal encode PNG preview: %w", err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GetTemplateDPI resolves the DPI from the target printer or defaults to 203 DPI
func GetTemplateDPI(targetPrinter string) (int32, string) {
	if targetPrinter == "" {
		targetPrinter, _ = winapi.GetDefaultPrinter()
	}
	dpi := int32(203)
	if targetPrinter != "" {
		if gdi, err := winapi.NewGDIPrinter(targetPrinter); err == nil {
			if gdi.DpiX > 0 {
				dpi = gdi.DpiX
			}
			gdi.Close()
		}
	}
	return dpi, targetPrinter
}

// GeneratePreviews renders multiple documents using the specified template and returns an array of Base64 PNGs
func GeneratePreviews(templateName string, documents []DocumentData, targetPrinter string) ([]string, error) {
	return GeneratePreviewsWithDir(templateName, "", documents, targetPrinter)
}

// GeneratePreviewsWithDir renders multiple documents using the specified template and custom directory (e.g. Synology NAS)
func GeneratePreviewsWithDir(templateName, templateDir string, documents []DocumentData, targetPrinter string) ([]string, error) {
	if len(documents) == 0 {
		documents = []DocumentData{{}}
	}

	tpl, err := LoadTemplateWithDir(templateName, templateDir)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat template: %w", err)
	}

	dpi, _ := GetTemplateDPI(targetPrinter)

	results := make([]string, 0, len(documents))
	for _, doc := range documents {
		pngBase64, err := RenderPreview(tpl, doc, dpi)
		if err != nil {
			return nil, err
		}
		results = append(results, pngBase64)
	}

	return results, nil
}
