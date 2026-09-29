package engine

import (
	"fmt"
	"strings"
	"time"

	"print-service/winapi"
)

// DocumentData holds key-value pairs for dynamic variable interpolation
type DocumentData map[string]interface{}

// LabelData is an alias for backward compatibility
type LabelData = DocumentData

// PrintRequest defines the API payload for /api/print and /api/preview
type PrintRequest struct {
	// Format snake_case (Konsisten & Utama)
	PrinterNameSnake  string         `json:"printer_name,omitempty"`
	TemplateNameSnake string         `json:"template_name,omitempty"`
	TemplateDirSnake  string         `json:"template_dir,omitempty"`
	DataSnake         []DocumentData `json:"data,omitempty"`

	// Backward compatibility: PascalCase
	PrinterName  string         `json:"PrinterName,omitempty"`
	TemplateName string         `json:"TemplateName,omitempty"`
	Template     string         `json:"Template,omitempty"`
	TemplateDir  string         `json:"TemplateDir,omitempty"`
	Data         []DocumentData `json:"Data,omitempty"`
}

// GetPrinterName returns the target printer name
func (r *PrintRequest) GetPrinterName() string {
	if s := strings.TrimSpace(r.PrinterNameSnake); s != "" {
		return s
	}
	return strings.TrimSpace(r.PrinterName)
}

// GetTemplateName returns the template name
func (r *PrintRequest) GetTemplateName() string {
	if s := strings.TrimSpace(r.TemplateNameSnake); s != "" {
		return s
	}
	if s := strings.TrimSpace(r.TemplateName); s != "" {
		return s
	}
	return strings.TrimSpace(r.Template)
}

// GetTemplateDir returns the template directory override
func (r *PrintRequest) GetTemplateDir() string {
	if s := strings.TrimSpace(r.TemplateDirSnake); s != "" {
		return s
	}
	return strings.TrimSpace(r.TemplateDir)
}

// GetData returns the array of document data
func (r *PrintRequest) GetData() []DocumentData {
	if len(r.DataSnake) > 0 {
		return r.DataSnake
	}
	return r.Data
}

// PrintDocuments prints one or more documents using the specified template to the printer
func PrintDocuments(targetPrinter string, templateName string, documents []DocumentData) error {
	return PrintDocumentsWithDir(targetPrinter, templateName, "", documents)
}

// PrintDocumentsWithDir prints documents with an optional template directory override
func PrintDocumentsWithDir(targetPrinter string, templateName string, templateDir string, documents []DocumentData) error {
	if len(documents) == 0 {
		return fmt.Errorf("tidak ada data dokumen yang dicetak")
	}

	tpl, err := LoadTemplateWithDir(templateName, templateDir)
	if err != nil {
		return fmt.Errorf("gagal memuat template: %w", err)
	}

	if targetPrinter == "" {
		def, err := winapi.GetDefaultPrinter()
		if err != nil {
			return fmt.Errorf("gagal mendapatkan printer default: %w", err)
		}
		targetPrinter = def
	}

	// Cek apakah template menggunakan pengulangan dinamis (band / table)
	hasLoopBand := false
	for _, el := range tpl.Elements {
		if el.Type == "band" || el.Type == "table" {
			hasLoopBand = true
			break
		}
	}

	upperPrinter := strings.ToUpper(targetPrinter)
	isReceiptOrDotMatrix := strings.Contains(upperPrinter, "TM-") ||
		strings.Contains(upperPrinter, "RECEIPT") ||
		strings.Contains(upperPrinter, "POS") ||
		strings.Contains(upperPrinter, "U220") ||
		strings.Contains(upperPrinter, "T82")

	// Hitung tinggi kertas yang dibutuhkan:
	// Untuk struk kasir / dot matrix, atau jika memiliki Area Data Berulang (Band), atau AutoHeight aktif:
	// Tinggi kertas OTOMATIS memotong pas di akhir data (tidak akan membuang kertas kosong walaupun di kanvas diset 297 mm).
	// Untuk label stiker barcode tetap (misal 80x50 mm), tetap menggunakan ukuran pas stiker (tpl.HeightMm).
	paperHeightMm := tpl.HeightMm
	for _, doc := range documents {
		neededH := CalculateDocumentHeightMm(tpl, doc)
		if hasLoopBand || tpl.AutoHeight || isReceiptOrDotMatrix || tpl.HeightMm <= 0 {
			if neededH > 0 {
				paperHeightMm = neededH
			}
		} else if neededH > paperHeightMm {
			paperHeightMm = neededH
		}
	}

	gdi, err := winapi.NewGDIPrinterWithPaperSize(targetPrinter, tpl.WidthMm, paperHeightMm)
	if err != nil {
		return err
	}
	defer gdi.Close()

	if err := gdi.StartDoc(tpl.Name); err != nil {
		return err
	}
	defer gdi.EndDoc()

	for _, doc := range documents {
		gdi.StartPage()
		RenderTemplate(gdi, tpl, doc)
		gdi.EndPage()
	}

	// Eksekusi potong kertas (Auto-Cut) setelah dokumen SELESAI seluruhnya:
	// 1. Template eksplisit menyetel auto_cut: true, ATAU
	// 2. Dokumen memiliki Area Data Berulang (band / table), ATAU
	// 3. Printer adalah printer struk roll (TM-U220, TM-T82, dll)
	if tpl.AutoCut || hasLoopBand || isReceiptOrDotMatrix || gdi.DpiY <= 180 {
		time.Sleep(150 * time.Millisecond)
		_ = winapi.CutPaper(targetPrinter)
	}

	return nil
}

// PrintLabels provides backward compatibility
func PrintLabels(targetPrinter string, templateName string, labels []DocumentData) error {
	return PrintDocuments(targetPrinter, templateName, labels)
}
