package tray

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"syscall"
	"unsafe"
)

//go:embed icon.png
var DefaultIconBytes []byte

var (
	modGdi32Icon = syscall.NewLazyDLL("gdi32.dll")
	procCreateBitmap       = modGdi32Icon.NewProc("CreateBitmap")
	procCreateCompatibleDC = modGdi32Icon.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = modGdi32Icon.NewProc("CreateDIBSection")
	procDeleteDC           = modGdi32Icon.NewProc("DeleteDC")
	procDeleteObject       = modGdi32Icon.NewProc("DeleteObject")

	procCreateIconIndirect = modUser32.NewProc("CreateIconIndirect")
	procDestroyIcon        = modUser32.NewProc("DestroyIcon")
)

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
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
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

// LoadDefaultIcon loads and returns the HICON created from embedded icon.png
func LoadDefaultIcon() (uintptr, error) {
	if len(DefaultIconBytes) == 0 {
		return 0, fmt.Errorf("icon.png is empty")
	}
	img, err := png.Decode(bytes.NewReader(DefaultIconBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to decode icon.png: %w", err)
	}
	return CreateHIconFromImage(img)
}

// CreateHIconFromImage converts an image.Image into a 32x32 32-bit ARGB Windows HICON
func CreateHIconFromImage(src image.Image) (uintptr, error) {
	targetW, targetH := 32, 32
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	srcB := src.Bounds()
	sw := srcB.Dx()
	sh := srcB.Dy()

	// High quality area-averaging downscaler for crisp icon
	for y := 0; y < targetH; y++ {
		sy0 := srcB.Min.Y + (y * sh) / targetH
		sy1 := srcB.Min.Y + ((y + 1) * sh) / targetH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}

		for x := 0; x < targetW; x++ {
			sx0 := srcB.Min.X + (x * sw) / targetW
			sx1 := srcB.Min.X + ((x + 1) * sw) / targetW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}

			var sumR, sumG, sumB, sumA uint64
			var count uint64
			for py := sy0; py < sy1; py++ {
				for px := sx0; px < sx1; px++ {
					c := src.At(px, py)
					r, g, b, a := c.RGBA()
					sumR += uint64(r >> 8)
					sumG += uint64(g >> 8)
					sumB += uint64(b >> 8)
					sumA += uint64(a >> 8)
					count++
				}
			}

			if count > 0 {
				dst.SetRGBA(x, y, color.RGBA{
					R: uint8(sumR / count),
					G: uint8(sumG / count),
					B: uint8(sumB / count),
					A: uint8(sumA / count),
				})
			}
		}
	}

	hdcMem, _, _ := procCreateCompatibleDC.Call(0)
	if hdcMem == 0 {
		return 0, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdcMem)

	bmi := BITMAPINFO{}
	bmi.Header.BiSize = uint32(unsafe.Sizeof(bmi.Header))
	bmi.Header.BiWidth = int32(targetW)
	bmi.Header.BiHeight = -int32(targetH) // Top-down DIB
	bmi.Header.BiPlanes = 1
	bmi.Header.BiBitCount = 32
	bmi.Header.BiCompression = 0 // BI_RGB

	var bitsPtr unsafe.Pointer
	hbmColor, _, _ := procCreateDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bmi)),
		0,
		uintptr(unsafe.Pointer(&bitsPtr)),
		0,
		0,
	)
	if hbmColor == 0 || bitsPtr == nil {
		return 0, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hbmColor)

	// Copy pixels to DIBSection memory (BGRA byte format)
	pixelSlice := (*[32 * 32 * 4]byte)(bitsPtr)
	idx := 0
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			c := dst.RGBAAt(x, y)
			pixelSlice[idx+0] = c.B
			pixelSlice[idx+1] = c.G
			pixelSlice[idx+2] = c.R
			pixelSlice[idx+3] = c.A
			idx += 4
		}
	}

	// 1-bit monochrome mask (DWORD-aligned: 32 bits = 4 bytes per row)
	maskBytes := make([]byte, targetH*4)
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			if dst.RGBAAt(x, y).A < 128 {
				byteIdx := y*4 + (x / 8)
				bitIdx := uint(7 - (x % 8))
				maskBytes[byteIdx] |= (1 << bitIdx)
			}
		}
	}

	hbmMask, _, _ := procCreateBitmap.Call(
		uintptr(targetW),
		uintptr(targetH),
		1,
		1,
		uintptr(unsafe.Pointer(&maskBytes[0])),
	)
	if hbmMask == 0 {
		return 0, fmt.Errorf("CreateBitmap mask failed")
	}
	defer procDeleteObject.Call(hbmMask)

	ii := ICONINFO{
		FIcon:    1,
		XHotspot: 0,
		YHotspot: 0,
		HbmMask:  hbmMask,
		HbmColor: hbmColor,
	}

	hIcon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	if hIcon == 0 {
		return 0, fmt.Errorf("CreateIconIndirect failed")
	}

	return hIcon, nil
}
