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
	ModeSnake         string         `json:"mode,omitempty"` // "label" (fixed size) atau "receipt" (dynamic auto_height)
	AutoHeightSnake   *bool          `json:"auto_height,omitempty"`
	AutoCutSnake      *bool          `json:"auto_cut,omitempty"`
	DataSnake         []DocumentData `json:"data,omitempty"`

	// Backward compatibility: PascalCase
	PrinterName  string         `json:"PrinterName,omitempty"`
	TemplateName string         `json:"TemplateName,omitempty"`
	Template     string         `json:"Template,omitempty"`
	TemplateDir  string         `json:"TemplateDir,omitempty"`
	Mode         string         `json:"Mode,omitempty"`
	AutoHeight   *bool          `json:"AutoHeight,omitempty"`
	AutoCut      *bool          `json:"AutoCut,omitempty"`
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

// GetMode returns the print mode ("label" or "receipt")
func (r *PrintRequest) GetMode() string {
	if s := strings.TrimSpace(r.ModeSnake); s != "" {
		return strings.ToLower(s)
	}
	return strings.ToLower(strings.TrimSpace(r.Mode))
}

// GetAutoHeight returns whether auto_height is enabled, checking mode, payload, then fallback to template
func (r *PrintRequest) GetAutoHeight(defaultVal bool) bool {
	if r.GetMode() == "receipt" {
		return true
	}
	if r.GetMode() == "label" {
		return false
	}
	if r.AutoHeightSnake != nil {
		return *r.AutoHeightSnake
	}
	if r.AutoHeight != nil {
		return *r.AutoHeight
	}
	return defaultVal
}

// GetAutoCut returns whether auto_cut is enabled
func (r *PrintRequest) GetAutoCut(defaultVal bool) bool {
	if r.GetMode() == "label" {
		return false
	}
	if r.AutoCutSnake != nil {
		return *r.AutoCutSnake
	}
	if r.AutoCut != nil {
		return *r.AutoCut
	}
	return defaultVal
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
	return PrintDocumentsWithRequest(&PrintRequest{}, targetPrinter, templateName, templateDir, documents)
}

// PrintDocumentsWithRequest prints documents with request overrides (mode, auto_height, auto_cut)
func PrintDocumentsWithRequest(req *PrintRequest, targetPrinter string, templateName string, templateDir string, documents []DocumentData) error {
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

	// Tentukan apakah dokumen menggunakan mode AutoHeight (seperti EndlessHeight di FastReport)
	isAutoHeight := tpl.AutoHeight
	if req != nil {
		isAutoHeight = req.GetAutoHeight(tpl.AutoHeight)
	}

	upperPrinter := strings.ToUpper(targetPrinter)
	isReceiptOrDotMatrix := strings.Contains(upperPrinter, "TM-") ||
		strings.Contains(upperPrinter, "RECEIPT") ||
		strings.Contains(upperPrinter, "POS") ||
		strings.Contains(upperPrinter, "U220") ||
		strings.Contains(upperPrinter, "T82")

	// Fallback untuk struk kasir roll jika template belum memiliki tag auto_height eksplisit
	if !isAutoHeight && (req == nil || req.GetMode() != "label") {
		if tpl.HeightMm <= 0 || (isReceiptOrDotMatrix && tpl.HeightMm >= 200) {
			isAutoHeight = true
		}
	}

	// Hitung tinggi kertas yang dibutuhkan (paperHeightMm):
	// 1. Mode Label Stiker (isAutoHeight == false): DIKUNCI MATI pada tpl.HeightMm (misal 30 mm).
	//    Sama persis seperti FastReport EndlessHeight = False. Tidak ada auto-expand, tidak ada margin liar.
	// 2. Mode Struk Roll (isAutoHeight == true): Dihitung dinamis mengikuti isi baris data belanjaan.
	paperHeightMm := tpl.HeightMm
	if isAutoHeight {
		dynMax := 0.0
		for _, doc := range documents {
			neededH := CalculateDocumentHeightMm(tpl, doc)
			if neededH > dynMax {
				dynMax = neededH
			}
		}
		if dynMax > 0 {
			paperHeightMm = dynMax
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
	// Mengikuti FastReport: HANYA jika auto_cut aktif dan BUKAN label stiker
	isAutoCut := tpl.AutoCut
	if req != nil {
		isAutoCut = req.GetAutoCut(tpl.AutoCut)
	}
	if isAutoCut && (isAutoHeight || isReceiptOrDotMatrix) {
		time.Sleep(150 * time.Millisecond)
		_ = winapi.CutPaper(targetPrinter)
	}

	return nil
}

// PrintLabels provides backward compatibility
func PrintLabels(targetPrinter string, templateName string, labels []DocumentData) error {
	return PrintDocuments(targetPrinter, templateName, labels)
}
