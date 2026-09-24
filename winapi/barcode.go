package winapi

import (
	"strings"
	"unsafe"
)

var code128Patterns = []string{
	"11011001100", "11001101100", "11001100110", "10010011000", "10010001100", // 0-4
	"10001001100", "10011001000", "10011000100", "10001100100", "11001001000", // 5-9
	"11001000100", "11000100100", "10110011100", "10011011100", "10011001110", // 10-14
	"10111001100", "10011101100", "10011100110", "11001110010", "11001011100", // 15-19
	"11001001110", "11011100100", "11001110100", "11101101110", "11101001100", // 20-24
	"11100101100", "11100100110", "11101100100", "11100110100", "11100110010", // 25-29
	"11011011000", "11011000110", "11000110110", "10100011000", "10001011000", // 30-34
	"10001000110", "10110001000", "10001101000", "10001100010", "11010001000", // 35-39
	"11000101000", "11000100010", "10110111000", "10110001110", "10001101110", // 40-44
	"10111011000", "10111000110", "10001110110", "11101110110", "11010001110", // 45-49
	"11000101110", "11011101000", "11011100010", "11011101110", "11101011000", // 50-54
	"11101000110", "11100010110", "11101101000", "11101100010", "11100011010", // 55-59
	"11101111010", "11001000010", "11110001010", "10100110000", "10100001100", // 60-64
	"10010110000", "10010000110", "10000101100", "10000100110", "10110010000", // 65-69
	"10110000100", "10011010000", "10011000010", "10000110100", "10000110010", // 70-74
	"11000010010", "11001010000", "11110111010", "11000010100", "10001111010", // 75-79
	"10100111100", "10010111100", "10010011110", "10111100100", "10011110100", // 80-84
	"10011110010", "11110100100", "11110010100", "11110010010", "11011011110", // 85-89
	"11011110110", "11110110110", "10101111000", "10100011110", "10001011110", // 90-94
	"10111101000", "10111100010", "11110101000", "11110100010", "10111011110", // 95-99
	"10111101110", "11101011110", "11110101110",                               // 100-102
	"11010000100", "11010010000", "11010011100",                               // 103-105 (Start A/B/C)
	"1100011101011",                                                             // 106 (Stop)
}

const (
	startB = 104
	stop   = 106
)

// EncodeCode128B converts ASCII text to binary string pattern of bars/spaces
func EncodeCode128B(text string) string {
	var sb strings.Builder
	sb.WriteString(code128Patterns[startB])
	checksum := startB

	for i, r := range text {
		val := int(r) - 32
		if val < 0 || val > 94 {
			val = 0
		}
		sb.WriteString(code128Patterns[val])
		checksum += val * (i + 1)
	}

	checkDigit := checksum % 103
	sb.WriteString(code128Patterns[checkDigit])
	sb.WriteString(code128Patterns[stop])

	return sb.String()
}

// DrawBarcode128 renders the barcode sharply using GDI rectangles with exact integer dot widths
func (p *GDIPrinter) DrawBarcode128(xMm, yMm, wMm, hMm float64, text string) {
	p.DrawBarcode128Ex(xMm, yMm, wMm, hMm, text, 0)
}

// DrawBarcode128Ex allows forcing dotWidth (1, 2, 3...) or auto-calculating if forcedDotWidth <= 0
func (p *GDIPrinter) DrawBarcode128Ex(xMm, yMm, wMm, hMm float64, text string, forcedDotWidth int32) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	pattern := EncodeCode128B(text)
	if len(pattern) == 0 {
		return
	}

	startX := p.MmToPxX(xMm)
	startY := p.MmToPxY(yMm)
	endY := p.MmToPxY(yMm + hMm)

	var dotWidth int32
	if forcedDotWidth > 0 {
		dotWidth = forcedDotWidth
	} else {
		targetWidthPx := p.MmToPxX(xMm+wMm) - startX
		dotWidth = int32(float64(targetWidthPx)/float64(len(pattern)) + 0.5)
		if dotWidth < 1 {
			dotWidth = 1
		}
	}

	// Kompensasi Pemuaian Thermal (Bar Width Reduction / BWR):
	// Pada printer thermal (seperti Zebra), panas printhead menyebabkan garis hitam sedikit melebar di kertas.
	// Jika dotWidth >= 2, kurangi 1 pixel dari setiap bar hitam agar celah putih (spasi)
	// otomatis menjadi LEBIH RENGGANG dan tidak pernah tertutup panas printhead.
	compensation := int32(0)
	if dotWidth >= 2 {
		compensation = 1
	}

	// Gambar batang hitam yang menyambung (contiguous bar) sebagai 1 rectangle solid
	currX := startX
	inBar := false
	barStart := currX

	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '1' {
			if !inBar {
				inBar = true
				barStart = currX
			}
		} else {
			if inBar {
				drawRight := currX - compensation
				if drawRight <= barStart {
					drawRight = barStart + 1
				}
				rc := RECT{
					Left:   barStart,
					Top:    startY,
					Right:  drawRight,
					Bottom: endY,
				}
				ProcFillRect.Call(p.HDC, uintptr(unsafe.Pointer(&rc)), p.BlackBr)
				inBar = false
			}
		}
		currX += dotWidth
	}

	if inBar {
		drawRight := currX - compensation
		if drawRight <= barStart {
			drawRight = barStart + 1
		}
		rc := RECT{
			Left:   barStart,
			Top:    startY,
			Right:  drawRight,
			Bottom: endY,
		}
		ProcFillRect.Call(p.HDC, uintptr(unsafe.Pointer(&rc)), p.BlackBr)
	}
}
