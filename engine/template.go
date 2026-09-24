package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"print-service/winapi"
)

type TemplateElement struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"` // "text", "barcode", "line", "image"
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Text     string  `json:"text"`
	FontSize float64 `json:"fontSize"`
	Bold     bool    `json:"bold"`
	Align    string  `json:"align,omitempty"` // "left", "center", "right"
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Dashed   bool    `json:"dashed"`
	DotWidth int32   `json:"dot_width,omitempty"` // Opsional: paksa ketebalan garis/modul (misal: 2, 3 dot)
	Src      string  `json:"src,omitempty"`       // Path file, base64 data, atau tag {{variable}}
}

type DocumentTemplate struct {
	Name     string            `json:"name"`
	WidthMm  float64           `json:"width_mm"`
	HeightMm float64           `json:"height_mm"`
	Elements []TemplateElement `json:"elements"`
}

// LabelTemplate is an alias for backward compatibility
type LabelTemplate = DocumentTemplate

// ResolveTemplatePath finds the template file in template/ or root directories
func ResolveTemplatePath(templateName string) (string, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return "", fmt.Errorf("nama template wajib diisi (tidak boleh kosong)! Harap tentukan nama file template")
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// Search order:
	// 1. template/<name>
	// 2. <name>
	// 3. <exeDir>/template/<name>
	// 4. <exeDir>/<name>
	candidates := []string{
		filepath.Join("template", name),
		name,
		filepath.Join("..", "template", name),
		filepath.Join("..", name),
		filepath.Join(exeDir, "template", name),
		filepath.Join(exeDir, name),
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("file template '%s' tidak ditemukan! Pastikan file berada di folder 'template/' (contoh: template/%s)", name, name)
}

// LoadTemplate reads the requested template JSON from template/ or executable directory
func LoadTemplate(templateName string) (*DocumentTemplate, error) {
	targetPath, err := ResolveTemplatePath(templateName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file template %s: %w", targetPath, err)
	}

	var tpl DocumentTemplate
	if err := json.Unmarshal(data, &tpl); err != nil {
		return nil, fmt.Errorf("gagal membaca format JSON file %s: %w", targetPath, err)
	}

	if tpl.WidthMm <= 0 || tpl.HeightMm <= 0 {
		return nil, fmt.Errorf("ukuran template tidak valid pada %s: lebar=%.1f mm, tinggi=%.1f mm", targetPath, tpl.WidthMm, tpl.HeightMm)
	}

	return &tpl, nil
}

// ListTemplates scans the template/ directory and returns all available .json templates
func ListTemplates() ([]string, error) {
	dirs := []string{"template", filepath.Join("..", "template")}
	if exePath, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exePath), "template"))
	}

	seen := make(map[string]bool)
	var list []string

	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				if !seen[e.Name()] {
					seen[e.Name()] = true
					list = append(list, e.Name())
				}
			}
		}
	}

	return list, nil
}

// RenderTemplate draws the template elements to GDI dynamically
func RenderTemplate(gdi *winapi.GDIPrinter, tpl *DocumentTemplate, doc DocumentData) {
	for _, el := range tpl.Elements {
		switch el.Type {
		case "text":
			text := el.Text
			for k, v := range doc {
				text = strings.ReplaceAll(text, "{{"+k+"}}", fmt.Sprintf("%v", v))
			}
			fontSizePx := el.FontSize
			if fontSizePx <= 0 {
				fontSizePx = 10.0
			}
			// Konversi px (layar web 96 DPI) ke pt (GDI printer): 1 px = 0.75 pt (72/96)
			fontSizePt := fontSizePx * 0.75
			hFont := gdi.CreateFont("Arial", fontSizePt, el.Bold)

			xPrint := el.X
			align := strings.ToLower(strings.TrimSpace(el.Align))
			if el.Width > 0 {
				if align == "right" {
					xPrint = el.X + el.Width
				} else if align == "center" {
					xPrint = el.X + (el.Width / 2.0)
				}
			}

			gdi.DrawText(xPrint, el.Y, text, hFont, el.Align)
			gdi.DeleteFont(hFont)

		case "barcode":
			barcodeValue := el.Text
			for k, v := range doc {
				barcodeValue = strings.ReplaceAll(barcodeValue, "{{"+k+"}}", fmt.Sprintf("%v", v))
			}
			w := el.Width
			if w <= 0 {
				w = 34.0
			}
			h := el.Height
			if h <= 0 {
				h = 7.5
			}
			gdi.DrawBarcode128Ex(el.X, el.Y, w, h, barcodeValue, el.DotWidth)

		case "line":
			w := el.Width
			if w <= 0 {
				w = tpl.WidthMm - 4.0
			}
			gdi.DrawLine(el.X, el.Y, el.X+w, el.Y, el.Dashed)

		case "image":
			src := strings.TrimSpace(el.Src)
			if src == "" {
				src = strings.TrimSpace(el.Text)
			}
			for k, v := range doc {
				src = strings.ReplaceAll(src, "{{"+k+"}}", fmt.Sprintf("%v", v))
			}
			if src == "" {
				continue
			}

			img, err := LoadImage(src)
			if err != nil {
				fmt.Printf("[GDI Image] Gagal memuat gambar (%s): %v\n", src, err)
				continue
			}

			w := el.Width
			h := el.Height
			if w <= 0 {
				w = 15.0
			}
			if h <= 0 {
				h = 15.0
			}

			if err := gdi.DrawImage(img, el.X, el.Y, w, h); err != nil {
				fmt.Printf("[GDI Image] Gagal mencetak gambar: %v\n", err)
			}
		}
	}
}
