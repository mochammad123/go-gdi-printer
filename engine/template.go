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

// NormalizePath normalizes Windows file paths and UNC network share paths (e.g. Synology NAS).
// It trims surrounding quotes/whitespace, fixes forward slashes, and corrects typos like
// "\\192.168.20.2:\share" -> "\\192.168.20.2\share".
func NormalizePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), "\"'`")
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "/", "\\")
	if strings.HasPrefix(p, `\\`) {
		rest := p[2:]
		parts := strings.SplitN(rest, `\`, 2)
		host := strings.TrimSuffix(parts[0], ":")
		if len(parts) == 2 {
			p = `\\` + host + `\` + parts[1]
		} else {
			p = `\\` + host
		}
	}
	return filepath.Clean(p)
}

// GetTemplateDir returns the absolute path to the template folder.
// Priority:
// 1. KNITTO_TEMPLATE_DIR environment variable
// 2. template_dir in config.json (e.g. Synology NAS: \\192.168.20.2\faisal\template or Z:\template)
// 3. template/ folder adjacent to the executable
func GetTemplateDir() string {
	if custom := os.Getenv("KNITTO_TEMPLATE_DIR"); custom != "" {
		return NormalizePath(custom)
	}
	cfg := GetConfig()
	if cfg.TemplateDir != "" {
		return NormalizePath(cfg.TemplateDir)
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

type TableColumn struct {
	Key   string  `json:"key"`             // Kunci field di array JSON (misal: "kain", "qty", "harga")
	Title string  `json:"title"`           // Judul kolom di header tabel (misal: "Nama Kain", "Qty")
	Width float64 `json:"width"`           // Lebar kolom dalam mm
	Align string  `json:"align,omitempty"` // "left", "center", "right"
}

type TemplateElement struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"` // "text", "barcode", "line", "image", "table"
	X          float64       `json:"x"`
	Y          float64       `json:"y"`
	Text       string        `json:"text"`
	FontSize   float64       `json:"fontSize"`
	FontFamily string        `json:"fontFamily,omitempty"` // "Arial", "Tahoma", "Verdana", dll
	Bold       bool          `json:"bold"`
	Align      string        `json:"align,omitempty"` // "left", "center", "right"
	Width      float64       `json:"width"`
	Height     float64       `json:"height"`
	Dashed     bool          `json:"dashed"`
	DotWidth   int32         `json:"dot_width,omitempty"` // Opsional: paksa ketebalan garis/modul (misal: 2, 3 dot)
	Src        string        `json:"src,omitempty"`       // Path file, base64 data, atau tag {{variable}}
	BandID     string        `json:"bandId,omitempty"`    // ID band jika elemen ini berada di dalam Area Data Berulang
	DataKey    string        `json:"dataKey,omitempty"`   // Kunci array JSON untuk table/band loop (default: "data")
	Columns    []TableColumn `json:"columns,omitempty"`   // Konfigurasi kolom tabel
	RowHeight  float64       `json:"rowHeight,omitempty"` // Tinggi tiap baris tabel dalam mm (default: 5.5)
	ShowHeader bool          `json:"showHeader,omitempty"`// Tampilkan judul kolom di header
}

type DocumentTemplate struct {
	Name       string            `json:"name"`
	WidthMm    float64           `json:"width_mm"`
	HeightMm   float64           `json:"height_mm"`
	AutoHeight bool              `json:"auto_height,omitempty"` // true = panjang kertas dinamis mengikuti isi data
	AutoCut    bool              `json:"auto_cut,omitempty"`    // true = otomatis potong kertas di akhir dokumen
	Elements   []TemplateElement `json:"elements"`
}

// CalculateDocumentHeightMm menghitung panjang fisik kertas (dalam mm) yang dibutuhkan untuk menampung seluruh isi data
func CalculateDocumentHeightMm(tpl *DocumentTemplate, doc DocumentData) float64 {
	// 1. Cek apakah ada elemen band (Area Data Berulang)
	var loopBand *TemplateElement
	isChildOfBand := make(map[string]bool)
	for i := range tpl.Elements {
		if tpl.Elements[i].Type == "band" {
			loopBand = &tpl.Elements[i]
			break
		}
	}

	bandShiftY := 0.0
	bandEndY := 0.0
	if loopBand != nil {
		bandHeight := loopBand.Height
		if bandHeight <= 0 {
			bandHeight = 8.0
		}
		dataKey := loopBand.DataKey
		if dataKey == "" {
			dataKey = "data"
		}
		rowCount := 0
		if rawArr, ok := doc[dataKey]; ok {
			if slice, ok := rawArr.([]interface{}); ok {
				rowCount = len(slice)
			}
		}
		actualHeight := float64(rowCount) * bandHeight
		if rowCount > 1 {
			bandShiftY = actualHeight - bandHeight
		} else if rowCount == 0 {
			bandShiftY = -bandHeight
		}
		bandEndY = loopBand.Y + actualHeight

		for _, child := range tpl.Elements {
			if child.ID != loopBand.ID && child.Type != "band" {
				if child.BandID == loopBand.ID || (child.Y >= loopBand.Y && child.Y < loopBand.Y+bandHeight) {
					isChildOfBand[child.ID] = true
				}
			}
		}
	}

	// 2. Cek apakah ada elemen table legacy
	tableShiftY := 0.0
	tableEndY := 0.0
	for _, el := range tpl.Elements {
		if el.Type == "table" {
			dataKey := el.DataKey
			if dataKey == "" {
				dataKey = "data"
			}
			rowCount := 0
			if rawArr, ok := doc[dataKey]; ok {
				if slice, ok := rawArr.([]interface{}); ok {
					rowCount = len(slice)
				}
			}
			rowHeight := el.RowHeight
			if rowHeight <= 0 {
				rowHeight = 5.5
			}
			headerHeight := 0.0
			if el.ShowHeader {
				headerHeight = rowHeight + 1.0 // + spasi garis header
			}
			actualTableHeight := headerHeight + float64(rowCount)*rowHeight
			designHeight := el.Height
			if designHeight <= 0 {
				designHeight = 15.0
			}
			if actualTableHeight > designHeight {
				tableShiftY = actualTableHeight - designHeight
			}
			tableEndY = el.Y + actualTableHeight
			break
		}
	}

	maxBottomY := 0.0
	if loopBand != nil && bandEndY > maxBottomY {
		maxBottomY = bandEndY
	}

	for _, el := range tpl.Elements {
		if isChildOfBand[el.ID] || el.Type == "band" {
			continue
		}

		bottomY := el.Y
		// Jika elemen berada di bawah band atau tabel, dorong ke bawah sesuai ekspansi
		if loopBand != nil && el.Y >= loopBand.Y+loopBand.Height {
			bottomY += bandShiftY
		} else if tableShiftY > 0 && el.Type != "table" && el.Y >= tableEndY-tableShiftY {
			bottomY += tableShiftY
		}

		switch el.Type {
		case "table":
			bottomY = tableEndY
		case "text":
			fontSizePt := el.FontSize * 0.75
			if fontSizePt <= 0 {
				fontSizePt = 7.5
			}
			fontHeightMm := fontSizePt * 0.352778
			if el.Height > 0 {
				bottomY += el.Height
			} else {
				bottomY += fontHeightMm * 1.3
			}
		case "barcode":
			h := el.Height
			if h <= 0 {
				h = 7.5
			}
			bottomY += h
		case "image":
			h := el.Height
			if h <= 0 {
				h = 15.0
			}
			bottomY += h
		case "line":
			bottomY += 1.0
		}
		if bottomY > maxBottomY {
			maxBottomY = bottomY
		}
	}
	// Berikan safety margin bawah (8 mm) agar isi terakhir melewati pisau pemotong
	safetyMarginMm := 8.0
	return maxBottomY + safetyMarginMm
}

// LabelTemplate is an alias for backward compatibility
type LabelTemplate = DocumentTemplate

// ResolveTemplatePath finds the template file in template/ or root directories
func ResolveTemplatePath(templateName string) (string, error) {
	return ResolveTemplatePathWithDir(templateName, "")
}

// ResolveTemplatePathWithDir resolves a template path with an optional custom directory override (e.g. Synology NAS UNC)
func ResolveTemplatePathWithDir(templateName string, customDir string) (string, error) {
	raw := NormalizePath(templateName)
	if raw == "" {
		return "", fmt.Errorf("nama template wajib diisi (tidak boleh kosong)! Harap tentukan nama file template")
	}

	name := raw
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}

	// 1. Cek langsung jika input sudah berupa full path atau UNC path jaringan (misal: \\192.168.20.2\faisal\template\xxx.json atau Z:\xxx.json)
	if filepath.IsAbs(name) || strings.HasPrefix(name, `\\`) {
		if fi, err := os.Stat(name); err == nil && !fi.IsDir() {
			return name, nil
		}
	}

	baseName := filepath.Base(name)

	// 2. Jika customDir diberikan (dari payload request API)
	if customDir != "" {
		cDir := NormalizePath(customDir)
		candidate := filepath.Join(cDir, baseName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, nil
		}
	}

	tmplDir := GetTemplateDir()
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// 3. Urutan pencarian:
	// - <tmplDir>/<name> (Bisa berupa Synology NAS UNC / config.json / KNITTO_TEMPLATE_DIR)
	// - template/<name>  (CWD)
	// - <exeDir>/template/<name>
	// - <exeDir>/<name>
	// - <name>
	// - ../template/<name>
	// - ../<name>
	candidates := []string{
		filepath.Join(tmplDir, baseName),
		filepath.Join("template", baseName),
		filepath.Join(exeDir, "template", baseName),
		filepath.Join(exeDir, baseName),
		name,
		filepath.Join("..", "template", baseName),
		filepath.Join("..", baseName),
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("file template '%s' tidak ditemukan di disk maupun jaringan", name)
}

// LoadTemplate reads the requested template JSON from disk or from embedded templates in the binary
func LoadTemplate(templateName string) (*DocumentTemplate, error) {
	return LoadTemplateWithDir(templateName, "")
}

// LoadTemplateWithDir reads template JSON with an optional custom directory override (e.g. Synology NAS)
func LoadTemplateWithDir(templateName string, customDir string) (*DocumentTemplate, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return nil, fmt.Errorf("nama template wajib diisi")
	}
	baseName := filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(baseName), ".json") {
		baseName += ".json"
	}

	// 1. Prioritas 1: Baca dari disk/jaringan jika ada (Synology / folder lokal)
	targetPath, err := ResolveTemplatePathWithDir(name, customDir)
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

	// 2. Prioritas 2 (Fallback): Baca dari embedded FS di dalam binary jika jaringan/file disk tidak ada
	if embeddedTemplatesFS != nil {
		embeddedPath := baseName
		if embeddedSubdir != "" {
			embeddedPath = embeddedSubdir + "/" + baseName
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

	return nil, fmt.Errorf("file template '%s' tidak ditemukan di folder template (Synology/lokal) maupun di dalam binary: %v", name, err)
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

// renderSingleElement draws an individual element (text, barcode, line, image) to GDI
func renderSingleElement(gdi *winapi.GDIPrinter, el TemplateElement, elY float64, doc DocumentData, tplWidthMm float64) {
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
		fontSizePt := fontSizePx * 0.75

		fontName := strings.TrimSpace(el.FontFamily)
		if fontName == "" {
			if gdi.DpiY <= 180 {
				fontName = "Tahoma"
			} else {
				fontName = "Arial"
			}
		}

		minPt := 3.5
		if gdi.DpiY <= 180 {
			minPt = 6.0
		}

		if el.Width > 0 && el.Height > 0 {
			xPx := gdi.MmToPxX(el.X)
			yPx := gdi.MmToPxY(elY)
			wPx := gdi.MmToPxX(el.Width)
			hPx := gdi.MmToPxY(el.Height)

			currentPt := fontSizePt
			hFont := gdi.CreateFont(fontName, currentPt, el.Bold)
			neededHPx := gdi.MeasureTextBoxPx(text, hFont, wPx)

			for neededHPx > hPx && currentPt > minPt {
				gdi.DeleteFont(hFont)
				currentPt -= 0.5
				if currentPt < minPt {
					currentPt = minPt
				}
				hFont = gdi.CreateFont(fontName, currentPt, el.Bold)
				neededHPx = gdi.MeasureTextBoxPx(text, hFont, wPx)
			}

			rc := winapi.RECT{
				Left:   xPx,
				Top:    yPx,
				Right:  xPx + wPx,
				Bottom: yPx + hPx,
			}
			gdi.DrawTextBox(text, hFont, rc, el.Align)
			gdi.DeleteFont(hFont)

		} else if el.Width > 0 {
			xPx := gdi.MmToPxX(el.X)
			yPx := gdi.MmToPxY(elY)
			wPx := gdi.MmToPxX(el.Width)

			hFont := gdi.CreateFont(fontName, fontSizePt, el.Bold)
			neededHPx := gdi.MeasureTextBoxPx(text, hFont, wPx)
			rc := winapi.RECT{
				Left:   xPx,
				Top:    yPx,
				Right:  xPx + wPx,
				Bottom: yPx + neededHPx,
			}
			gdi.DrawTextBox(text, hFont, rc, el.Align)
			gdi.DeleteFont(hFont)

		} else {
			hFont := gdi.CreateFont(fontName, fontSizePt, el.Bold)
			xPrint := el.X
			gdi.DrawText(xPrint, elY, text, hFont, el.Align)
			gdi.DeleteFont(hFont)
		}

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
		gdi.DrawBarcode128Ex(el.X, elY, w, h, barcodeValue, el.DotWidth)

	case "line":
		w := el.Width
		if w <= 0 {
			w = tplWidthMm - 4.0
		}
		gdi.DrawLine(el.X, elY, el.X+w, elY, el.Dashed)

	case "image":
		src := strings.TrimSpace(el.Src)
		if src == "" {
			src = strings.TrimSpace(el.Text)
		}
		for k, v := range doc {
			src = strings.ReplaceAll(src, "{{"+k+"}}", fmt.Sprintf("%v", v))
		}
		if src == "" {
			return
		}

		img, err := LoadImage(src)
		if err != nil {
			fmt.Printf("[GDI Image] Gagal memuat gambar (%s): %v\n", src, err)
			return
		}

		w := el.Width
		h := el.Height
		if w <= 0 {
			w = 15.0
		}
		if h <= 0 {
			h = 15.0
		}

		if err := gdi.DrawImage(img, el.X, elY, w, h); err != nil {
			fmt.Printf("[GDI Image] Gagal mencetak gambar: %v\n", err)
		}
	}
}

// RenderTemplate draws the template elements to GDI dynamically
func RenderTemplate(gdi *winapi.GDIPrinter, tpl *DocumentTemplate, doc DocumentData) {
	// 1. Cek apakah ada elemen band (Area Data Berulang)
	var loopBand *TemplateElement
	var bandChildren []TemplateElement
	isChildOfBand := make(map[string]bool)
	var bandItems []map[string]interface{}
	bandShiftY := 0.0
	bandHeight := 8.0

	for i := range tpl.Elements {
		if tpl.Elements[i].Type == "band" {
			loopBand = &tpl.Elements[i]
			break
		}
	}

	if loopBand != nil {
		bandHeight = loopBand.Height
		if bandHeight <= 0 {
			bandHeight = 8.0
		}
		dataKey := loopBand.DataKey
		if dataKey == "" {
			dataKey = "data"
		}
		if rawArr, ok := doc[dataKey]; ok {
			if slice, ok := rawArr.([]interface{}); ok {
				for _, it := range slice {
					if m, ok := it.(map[string]interface{}); ok {
						bandItems = append(bandItems, m)
					}
				}
			}
		}

		for _, el := range tpl.Elements {
			if el.ID != loopBand.ID && el.Type != "band" {
				if el.BandID == loopBand.ID || (el.Y >= loopBand.Y && el.Y < loopBand.Y+bandHeight) {
					bandChildren = append(bandChildren, el)
					isChildOfBand[el.ID] = true
				}
			}
		}

		rowCount := len(bandItems)
		actualBandHeight := float64(rowCount) * bandHeight
		if rowCount > 1 {
			bandShiftY = actualBandHeight - bandHeight
		} else if rowCount == 0 {
			bandShiftY = -bandHeight
		}
	}

	// 2. Cek apakah ada elemen table legacy dan hitung pergeseran Y (shiftY) akibat ekspansi baris dinamis
	tableShiftY := 0.0
	tableEndY := 0.0
	for _, el := range tpl.Elements {
		if el.Type == "table" {
			dataKey := el.DataKey
			if dataKey == "" {
				dataKey = "data"
			}
			rowCount := 0
			if rawArr, ok := doc[dataKey]; ok {
				if slice, ok := rawArr.([]interface{}); ok {
					rowCount = len(slice)
				}
			}
			rowHeight := el.RowHeight
			if rowHeight <= 0 {
				rowHeight = 5.5
			}
			headerHeight := 0.0
			if el.ShowHeader {
				headerHeight = rowHeight + 1.0 // + spasi garis header
			}
			actualTableHeight := headerHeight + float64(rowCount)*rowHeight
			designHeight := el.Height
			if designHeight <= 0 {
				designHeight = 15.0
			}
			if actualTableHeight > designHeight {
				tableShiftY = actualTableHeight - designHeight
			}
			tableEndY = el.Y + actualTableHeight
			break
		}
	}

	for _, el := range tpl.Elements {
		// Elemen anak dari band akan di-render berulang di dalam band itu sendiri
		if isChildOfBand[el.ID] {
			continue
		}

		if el.Type == "band" {
			// Render pengulangan elemen untuk tiap item data di band
			for i, rowData := range bandItems {
				rowOffsetY := float64(i) * bandHeight
				mergedDoc := make(DocumentData)
				for k, v := range doc {
					mergedDoc[k] = v
				}
				for k, v := range rowData {
					mergedDoc[k] = v
				}

				for _, child := range bandChildren {
					childRelY := child.Y - loopBand.Y
					actualChildY := loopBand.Y + rowOffsetY + childRelY
					renderSingleElement(gdi, child, actualChildY, mergedDoc, tpl.WidthMm)
				}
			}
			continue
		}

		// Hitung posisi Y efektif: elemen di bawah band/tabel otomatis terdorong ke bawah
		elY := el.Y
		if loopBand != nil && el.Y >= loopBand.Y+loopBand.Height {
			elY += bandShiftY
		} else if tableShiftY > 0 && el.Type != "table" && el.Y >= tableEndY-tableShiftY {
			elY += tableShiftY
		}

		switch el.Type {
		case "table":
			dataKey := el.DataKey
			if dataKey == "" {
				dataKey = "data"
			}
			fontName := strings.TrimSpace(el.FontFamily)
			if fontName == "" {
				if gdi.DpiY <= 180 {
					fontName = "Tahoma"
				} else {
					fontName = "Arial"
				}
			}
			fontSizePx := el.FontSize
			if fontSizePx <= 0 {
				fontSizePx = 9.0
			}
			fontSizePt := fontSizePx * 0.75
			rowHeightMm := el.RowHeight
			if rowHeightMm <= 0 {
				rowHeightMm = 5.5
			}

			// Font untuk Header (Bold) dan Data (Normal)
			hFontNormal := gdi.CreateFont(fontName, fontSizePt, false)
			hFontBold := gdi.CreateFont(fontName, fontSizePt, true)

			curY := el.Y

			// A. Header Kolom Tabel (jika diaktifkan)
			if el.ShowHeader && len(el.Columns) > 0 {
				curX := el.X
				for _, col := range el.Columns {
					wCol := col.Width
					if wCol <= 0 {
						wCol = 20.0
					}
					xPx := gdi.MmToPxX(curX)
					yPx := gdi.MmToPxY(curY)
					wPx := gdi.MmToPxX(wCol)
					hPx := gdi.MmToPxY(rowHeightMm)

					rc := winapi.RECT{
						Left:   xPx,
						Top:    yPx,
						Right:  xPx + wPx,
						Bottom: yPx + hPx,
					}
					gdi.DrawTextBox(col.Title, hFontBold, rc, col.Align)
					curX += wCol
				}
				curY += rowHeightMm
				// Garis pemisah bawah header
				totalW := curX - el.X
				gdi.DrawLine(el.X, curY, el.X+totalW, curY, false)
				curY += 1.0 // jeda spasi kecil
			}

			// B. Looping Baris Data Dinamis
			if rawArr, ok := doc[dataKey]; ok {
				if slice, ok := rawArr.([]interface{}); ok {
					for _, item := range slice {
						itemMap, ok := item.(map[string]interface{})
						if !ok {
							continue
						}
						curX := el.X
						for _, col := range el.Columns {
							wCol := col.Width
							if wCol <= 0 {
								wCol = 20.0
							}
							val := ""
							if v, exists := itemMap[col.Key]; exists {
								val = fmt.Sprintf("%v", v)
							}

							xPx := gdi.MmToPxX(curX)
							yPx := gdi.MmToPxY(curY)
							wPx := gdi.MmToPxX(wCol)
							hPx := gdi.MmToPxY(rowHeightMm)

							rc := winapi.RECT{
								Left:   xPx,
								Top:    yPx,
								Right:  xPx + wPx,
								Bottom: yPx + hPx,
							}
							gdi.DrawTextBox(val, hFontNormal, rc, col.Align)
							curX += wCol
						}
						curY += rowHeightMm
					}
				}
			}

			gdi.DeleteFont(hFontNormal)
			gdi.DeleteFont(hFontBold)

		default:
			renderSingleElement(gdi, el, elY, doc, tpl.WidthMm)
		}
	}
}
