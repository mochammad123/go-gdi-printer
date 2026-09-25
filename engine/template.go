package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"print-service/winapi"
)

var (
	embeddedTemplatesFS fs.FS
	embeddedSubdir      string
)

// RegisterEmbeddedTemplates registers the embedded filesystem containing default templates
func RegisterEmbeddedTemplates(efs fs.FS, subdir string) {
	embeddedTemplatesFS = efs
	embeddedSubdir = subdir
}

// GetTemplateDir returns the absolute path to the template/ folder adjacent to the executable.
// If the KNITTO_TEMPLATE_DIR environment variable is set, it takes precedence.
func GetTemplateDir() string {
	if custom := os.Getenv("KNITTO_TEMPLATE_DIR"); custom != "" {
		return custom
	}
	exePath, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exePath), "template")
	}
	return "template"
}

// OpenTemplateFolder opens the template directory in Windows File Explorer
func OpenTemplateFolder() error {
	dir := GetTemplateDir()
	_ = os.MkdirAll(dir, 0755)
	return exec.Command("cmd", "/c", "start", "", dir).Start()
}

// AddTemplateFile copies an external JSON template file into the service template folder
func AddTemplateFile(srcPath string) (string, error) {
	srcPath = strings.Trim(strings.TrimSpace(srcPath), "\"")
	if srcPath == "" {
		return "", fmt.Errorf("path file template asal tidak boleh kosong")
	}

	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file '%s': %w", srcPath, err)
	}
	srcData = bytes.TrimPrefix(srcData, []byte("\xef\xbb\xbf"))

	// Validasi bahwa file adalah JSON template yang valid
	var testTpl DocumentTemplate
	if err := json.Unmarshal(srcData, &testTpl); err != nil {
		return "", fmt.Errorf("file '%s' bukan format JSON template yang valid: %w", srcPath, err)
	}

	targetDir := EnsureTemplateFolder()
	fileName := filepath.Base(srcPath)
	if !strings.HasSuffix(strings.ToLower(fileName), ".json") {
		fileName += ".json"
	}

	destPath := filepath.Join(targetDir, fileName)
	if err := os.WriteFile(destPath, srcData, 0644); err != nil {
		return "", fmt.Errorf("gagal menyalin ke '%s': %w", destPath, err)
	}

	return destPath, nil
}

// EnsureTemplateFolder ensures a template/ folder exists next to the executable.
// If missing or if embedded templates are available, extracts any missing templates to disk.
func EnsureTemplateFolder() string {
	targetDir := GetTemplateDir()
	_ = os.MkdirAll(targetDir, 0755)

	if embeddedTemplatesFS != nil {
		entries, err := fs.ReadDir(embeddedTemplatesFS, embeddedSubdir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
					destPath := filepath.Join(targetDir, e.Name())
					// Hanya ekstrak jika file belum ada di disk (menghargai modifikasi pengguna)
					if _, err := os.Stat(destPath); os.IsNotExist(err) {
						filePath := e.Name()
						if embeddedSubdir != "" {
							filePath = embeddedSubdir + "/" + e.Name()
						}
						data, err := fs.ReadFile(embeddedTemplatesFS, filePath)
						if err == nil {
							_ = os.WriteFile(destPath, data, 0644)
						}
					}
				}
			}
		}
	}
	return targetDir
}

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

	tmplDir := GetTemplateDir()
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// Urutan pencarian:
	// 1. <tmplDir>/<name> (Folder template di sebelah exe / KNITTO_TEMPLATE_DIR)
	// 2. template/<name>  (CWD)
	// 3. <exeDir>/<name>
	// 4. <name>
	// 5. ../template/<name>
	candidates := []string{
		filepath.Join(tmplDir, name),
		filepath.Join("template", name),
		filepath.Join(exeDir, name),
		name,
		filepath.Join("..", "template", name),
		filepath.Join("..", name),
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("file template '%s' tidak ditemukan di disk", name)
}

// LoadTemplate reads the requested template JSON from disk or from embedded templates in the binary
func LoadTemplate(templateName string) (*DocumentTemplate, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return nil, fmt.Errorf("nama template wajib diisi")
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}

	// 1. Prioritas 1: Baca dari disk jika ada (memungkinkan kustomisasi file oleh pengguna)
	targetPath, err := ResolveTemplatePath(name)
	if err == nil {
		data, err := os.ReadFile(targetPath)
		if err == nil {
			data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
			var tpl DocumentTemplate
			if err := json.Unmarshal(data, &tpl); err == nil && tpl.WidthMm > 0 && tpl.HeightMm > 0 {
				return &tpl, nil
			}
		}
	}

	// 2. Prioritas 2 (Fallback): Baca dari embedded FS di dalam binary (selalu ada walau folder template tidak kebawa)
	if embeddedTemplatesFS != nil {
		embeddedPath := name
		if embeddedSubdir != "" {
			embeddedPath = embeddedSubdir + "/" + name
		}
		data, err := fs.ReadFile(embeddedTemplatesFS, embeddedPath)
		if err == nil {
			data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
			var tpl DocumentTemplate
			if err := json.Unmarshal(data, &tpl); err == nil && tpl.WidthMm > 0 && tpl.HeightMm > 0 {
				return &tpl, nil
			}
		}
	}

	return nil, fmt.Errorf("file template '%s' tidak ditemukan di folder template/ maupun di dalam binary", name)
}

// ListTemplates scans disk directories and embedded binary templates, returning deduplicated list
func ListTemplates() ([]string, error) {
	tmplDir := GetTemplateDir()
	dirs := []string{tmplDir, "template", filepath.Join("..", "template")}
	if exePath, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exePath), "template"))
	}

	seen := make(map[string]bool)
	var list []string

	// 1. Scan folder template di disk
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

	// 2. Scan template embedded di binary
	if embeddedTemplatesFS != nil {
		entries, err := fs.ReadDir(embeddedTemplatesFS, embeddedSubdir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
					if !seen[e.Name()] {
						seen[e.Name()] = true
						list = append(list, e.Name())
					}
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
