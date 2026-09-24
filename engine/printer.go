package engine

import (
	"fmt"

	"print-service/winapi"
)

// DocumentData holds key-value pairs for dynamic variable interpolation
type DocumentData map[string]interface{}

// LabelData is an alias for backward compatibility
type LabelData = DocumentData

// PrintRequest defines the API payload
type PrintRequest struct {
	PrinterName  string         `json:"PrinterName"`
	TemplateName string         `json:"TemplateName"` // e.g. "label_kain_80x30", "receipt_80mm"
	Template     string         `json:"Template"`     // Alias for TemplateName
	Data         []DocumentData `json:"Data"`
}

// PrintDocuments prints one or more documents using the specified template to the printer
func PrintDocuments(targetPrinter string, templateName string, documents []DocumentData) error {
	if len(documents) == 0 {
		return fmt.Errorf("tidak ada data dokumen yang dicetak")
	}

	tpl, err := LoadTemplate(templateName)
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

	gdi, err := winapi.NewGDIPrinterWithPaperSize(targetPrinter, tpl.WidthMm, tpl.HeightMm)
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

	return nil
}

// PrintLabels provides backward compatibility
func PrintLabels(targetPrinter string, templateName string, labels []DocumentData) error {
	return PrintDocuments(targetPrinter, templateName, labels)
}
