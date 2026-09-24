package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"print-service/engine"
	"print-service/winapi"
)

func main() {
	port := flag.Int("port", 8080, "Port untuk REST API server")
	templateFlag := flag.String("template", "", "Nama file template di folder template/ (wajib diisi, contoh: label_kain_80x30.json)")
	testPrint := flag.Bool("test", false, "Cetak 1 dokumen langsung dari template untuk tes")
	infoFlag := flag.Bool("info", false, "Tampilkan informasi ukuran kertas & resolusi printer default")
	listPrinters := flag.Bool("printers", false, "Tampilkan daftar printer yang terpasang")
	listTemplates := flag.Bool("templates", false, "Tampilkan daftar file template yang tersedia di folder template/")
	flag.Parse()

	// 0. Opsi CLI: Info printer
	if *infoFlag {
		if strings.TrimSpace(*templateFlag) == "" {
			log.Fatalf("Error: flag -template wajib diisi! Contoh: .\\print-service.exe -info -template label_kain_80x30")
		}
		def, err := winapi.GetDefaultPrinter()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		tpl, err := engine.LoadTemplate(*templateFlag)
		if err != nil {
			log.Fatalf("Error template: %v", err)
		}
		gdi, err := winapi.NewGDIPrinterWithPaperSize(def, tpl.WidthMm, tpl.HeightMm)
		if err != nil {
			log.Fatalf("Error NewGDIPrinter: %v", err)
		}
		defer gdi.Close()

		wMm, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 4)     // HORZSIZE
		hMm, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 6)     // VERTSIZE
		wPx, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 8)     // HORZRES
		hPx, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 10)    // VERTRES
		offX, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 112)  // PHYSICALOFFSETX
		offY, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 113)  // PHYSICALOFFSETY
		physW, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 110) // PHYSICALWIDTH
		physH, _, _ := winapi.ProcGetDeviceCaps.Call(gdi.HDC, 111) // PHYSICALHEIGHT

		fmt.Printf("Printer Name : %s\n", def)
		fmt.Printf("Template Name: %s (%s)\n", tpl.Name, *templateFlag)
		fmt.Printf("DPI (X x Y)  : %d x %d DPI\n", gdi.DpiX, gdi.DpiY)
		fmt.Printf("Ukuran Driver: %d mm x %d mm\n", wMm, hMm)
		fmt.Printf("Resolusi Area: %d px x %d px\n", wPx, hPx)
		fmt.Printf("Physical Area: %d px x %d px\n", physW, physH)
		fmt.Printf("Offset Margin: X=%d px, Y=%d px\n", offX, offY)
		return
	}

	// 1. Opsi CLI: List printers
	if *listPrinters {
		def, _ := winapi.GetDefaultPrinter()
		fmt.Printf("Default Printer: %s\n\n", def)
		printers, err := winapi.ListInstalledPrinters()
		if err != nil {
			log.Fatalf("Error list printer: %v", err)
		}
		fmt.Println("Daftar Printer Terpasang:")
		for i, p := range printers {
			marker := "  "
			if strings.EqualFold(p, def) {
				marker = "-> "
			}
			fmt.Printf("%s[%d] %s\n", marker, i+1, p)
		}
		return
	}

	// 2. Opsi CLI: List templates
	if *listTemplates {
		templates, err := engine.ListTemplates()
		if err != nil {
			log.Fatalf("Error list templates: %v", err)
		}
		fmt.Printf("Daftar Template di folder 'template/': (%d template ditemukan)\n", len(templates))
		for i, t := range templates {
			fmt.Printf("  [%d] %s\n", i+1, t)
		}
		return
	}

	// 3. Opsi CLI: Test print langsung dari template
	if *testPrint {
		if strings.TrimSpace(*templateFlag) == "" {
			log.Fatalf("Error: flag -template wajib diisi! Contoh: .\\print-service.exe -test -template label_kain_80x30")
		}
		def, err := winapi.GetDefaultPrinter()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Printf("Mencetak dokumen menggunakan template '%s' ke: %s\n", *templateFlag, def)
		err = engine.PrintDocuments(def, *templateFlag, []engine.DocumentData{{}})
		if err != nil {
			log.Fatalf("Gagal mencetak: %v", err)
		}
		fmt.Printf("Sukses mencetak dokumen dari template '%s'!\n", *templateFlag)
		return
	}

	// 4. Mode Server REST API
	setupAPIServer(*port)
}

func setupAPIServer(port int) {
	mux := http.NewServeMux()

	// Endpoint: Health Check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.URL.Path != "/" {
			sendError(w, http.StatusNotFound, "Endpoint tidak ditemukan")
			return
		}
		def, _ := winapi.GetDefaultPrinter()
		templates, _ := engine.ListTemplates()
		sendSuccess(w, "Print Service online", map[string]interface{}{
			"status":          "online",
			"service":         "Print Service (Windows Native GDI)",
			"default_printer": def,
			"templates":       templates,
			"endpoints": []string{
				"GET  /",
				"GET  /api/printers",
				"GET  /api/templates",
				"POST /api/preview",
				"POST /api/print",
			},
		})
	})

	// Endpoint: List Printers
	mux.HandleFunc("/api/printers", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		def, _ := winapi.GetDefaultPrinter()
		list, err := winapi.ListInstalledPrinters()
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal mengambil daftar printer: %v", err))
			return
		}

		sendSuccess(w, "Berhasil mengambil daftar printer", map[string]interface{}{
			"default":  def,
			"printers": list,
		})
	})

	// Endpoint: List Templates
	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		templates, err := engine.ListTemplates()
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendSuccess(w, "Berhasil mengambil daftar template", templates)
	})

	// Endpoint: Print Document (Universal print endpoint)
	printHandler := func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req engine.PrintRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
			return
		}

		docs := req.Data
		if len(docs) == 0 {
			docs = []engine.DocumentData{{}}
		}

		targetPrinter := req.PrinterName
		if targetPrinter == "" {
			targetPrinter, _ = winapi.GetDefaultPrinter()
		}

		tplName := strings.TrimSpace(req.TemplateName)
		if tplName == "" {
			tplName = strings.TrimSpace(req.Template)
		}
		if tplName == "" {
			sendError(w, http.StatusBadRequest, "TemplateName wajib diisi! Harap cantumkan nama template di payload JSON (contoh: {\"TemplateName\": \"label_kain_80x30\"})")
			return
		}

		log.Printf("Menerima request cetak %d dokumen (template: '%s') ke: %s\n", len(docs), tplName, targetPrinter)

		err = engine.PrintDocuments(targetPrinter, tplName, docs)
		if err != nil {
			log.Printf("Gagal mencetak: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendSuccess(w, fmt.Sprintf("Berhasil mencetak %d dokumen menggunakan template '%s' ke %s", len(docs), tplName, targetPrinter), nil)
	}

	// Endpoint: Preview Document (Universal preview endpoint - Base64 PNG)
	previewHandler := func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req engine.PrintRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
			return
		}

		tplName := strings.TrimSpace(req.TemplateName)
		if tplName == "" {
			tplName = strings.TrimSpace(req.Template)
		}
		if tplName == "" {
			sendError(w, http.StatusBadRequest, "TemplateName wajib diisi! Harap cantumkan nama template di payload JSON (contoh: {\"TemplateName\": \"label_kain_80x30\"})")
			return
		}

		log.Printf("Menerima request preview %d dokumen (template: '%s')\n", len(req.Data), tplName)

		previews, err := engine.GeneratePreviews(tplName, req.Data, req.PrinterName)
		if err != nil {
			log.Printf("Gagal membuat preview: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendSuccess(w, fmt.Sprintf("Berhasil membuat preview %d dokumen", len(previews)), previews)
	}

	mux.HandleFunc("/api/print", printHandler)
	mux.HandleFunc("/api/preview", previewHandler)

	defPrinter, _ := winapi.GetDefaultPrinter()
	addr := fmt.Sprintf(":%d", port)

	fmt.Println("==================================================")
	fmt.Println("🚀 Print Service (Windows Native GDI)")
	fmt.Printf("📡 API Server berjalan di http://localhost:%d\n", port)
	fmt.Printf("🖨️  Default Printer : %s\n", defPrinter)
	fmt.Println("--------------------------------------------------")
	fmt.Println("Endpoints:")
	fmt.Printf("  GET  http://localhost:%d/api/printers\n", port)
	fmt.Printf("  GET  http://localhost:%d/api/templates\n", port)
	fmt.Printf("  POST http://localhost:%d/api/preview\n", port)
	fmt.Printf("  POST http://localhost:%d/api/print\n", port)
	fmt.Println("==================================================")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server berhenti: %v", err)
	}
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// APIResponse represents standard API response: { message, result }
type APIResponse struct {
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func sendSuccess(w http.ResponseWriter, message string, result interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{
		Message: message,
		Result:  result,
	})
}

func sendError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, APIResponse{
		Message: message,
		Result:  nil,
	})
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
