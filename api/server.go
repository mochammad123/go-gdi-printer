package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"print-service/engine"
	"print-service/previewgui"
	"print-service/tray"
	"print-service/winapi"
)

// SetupServer menjalankan REST API server (+ optional system tray) sampai dihentikan.
func SetupServer(port int, enableTray bool, buildMode string) {
	mux := http.NewServeMux()

	var (
		serverMu   sync.RWMutex
		activePort = port
		currentLn  net.Listener
		httpServer *http.Server
		exitChan   = make(chan struct{})
		switchMu   sync.Mutex

		trayApp *tray.TrayApp
		trayMu  sync.Mutex
	)

	getActivePort := func() int {
		serverMu.RLock()
		defer serverMu.RUnlock()
		return activePort
	}

	updateTrayPrinter := func(printerName string) {
		trayMu.Lock()
		defer trayMu.Unlock()
		if trayApp != nil {
			trayApp.UpdatePrinter(printerName)
		}
	}

	updateTrayPort := func(p int) {
		trayMu.Lock()
		defer trayMu.Unlock()
		if trayApp != nil {
			trayApp.UpdatePort(p)
		}
	}

	switchPort := func(newPort int) error {
		switchMu.Lock()
		defer switchMu.Unlock()

		curPort := getActivePort()
		if newPort == curPort {
			return nil
		}
		if newPort < 1024 || newPort > 65535 {
			return fmt.Errorf("port harus antara 1024 dan 65535")
		}

		// 1. Coba bind port baru terlebih dahulu
		newLn, err := net.Listen("tcp", fmt.Sprintf(":%d", newPort))
		if err != nil {
			return fmt.Errorf("port %d gagal dibuka atau sedang digunakan oleh aplikasi lain: %w", newPort, err)
		}

		// 2. Simpan ke config.json
		curCfg := engine.GetConfig()
		curCfg.Port = newPort
		if err := engine.SaveConfig(curCfg); err != nil {
			_ = newLn.Close()
			return fmt.Errorf("gagal menyimpan konfigurasi: %w", err)
		}

		// 3. Update listener dan activePort
		serverMu.Lock()
		oldLn := currentLn
		currentLn = newLn
		activePort = newPort
		serverMu.Unlock()

		// 4. Update Tray Icon & Native Windows UI
		updateTrayPort(newPort)

		// 5. Mulai melayani request di port baru
		go func(ln net.Listener, p int) {
			log.Printf("📡 HTTP Server sekarang aktif di port baru :%d\n", p)
			if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				// listener ditutup saat port berganti lagi atau server shutdown
			}
		}(newLn, newPort)

		// 6. Graceful close listener lama agar response ke client saat ini sempat terkirim
		go func(old net.Listener) {
			if old != nil {
				time.Sleep(500 * time.Millisecond)
				_ = old.Close()
			}
		}(oldLn)

		log.Printf("Port berhasil dialihkan dari :%d ke :%d\n", curPort, newPort)
		return nil
	}

	// Terapkan default printer dari config jika telah disimpan sebelumnya
	cfg := engine.GetConfig()
	if cfg.DefaultPrinter != "" {
		_ = winapi.SetDefaultPrinter(cfg.DefaultPrinter)
	}
	defPrinter, _ := winapi.GetDefaultPrinter()

	// Endpoint: Dashboard Web UI & Root Health Check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
			sendError(w, http.StatusNotFound, "Endpoint tidak ditemukan")
			return
		}

		// Jika request dari browser, tampilkan Dashboard UI
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") || r.URL.Path == "/dashboard" {
			engine.ServeDashboard(w, r)
			return
		}

		// Fallback JSON status untuk API client
		templates, _ := engine.ListTemplates()
		curDef, _ := winapi.GetDefaultPrinter()
		sendSuccess(w, "Print Service online", map[string]interface{}{
			"status":          "online",
			"service":         "Print Service (Windows Native GDI)",
			"build_mode":      buildMode,
			"default_printer": curDef,
			"port":            getActivePort(),
			"template_dir":    engine.GetTemplateDir(),
			"templates":       templates,
			"autostart":       winapi.IsAutoStartEnabled(),
			"endpoints": []string{
				"GET  /",
				"GET  /dashboard",
				"GET  /api/status",
				"GET  /api/printers",
				"GET  /api/templates",
				"POST /api/templates/open-dir",
				"POST /api/templates/upload",
				"POST /api/config/printer",
				"POST /api/config/port",
				"POST /api/config/autostart",
				"POST /api/config/template-dir",
				"POST /api/preview",
				"POST /api/preview-window",
				"POST /api/print",
			},
		})
	})

	// Endpoint: Status Lengkap untuk Web Dashboard
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		curDef, _ := winapi.GetDefaultPrinter()
		templates, _ := engine.ListTemplates()
		curCfg := engine.GetConfig()

		sendSuccess(w, "Print Service status", map[string]interface{}{
			"status":          "online",
			"service":         "Print Service (Windows Native GDI)",
			"build_mode":      buildMode,
			"default_printer": curDef,
			"port":            getActivePort(),
			"configured_port": curCfg.Port,
			"template_dir":    engine.GetTemplateDir(),
			"templates":       templates,
			"autostart":       winapi.IsAutoStartEnabled(),
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

	// Endpoint: Ubah Default Printer (Disimpan ke Windows & config.json)
	mux.HandleFunc("/api/config/printer", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req struct {
			DefaultPrinter string `json:"default_printer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		req.DefaultPrinter = strings.TrimSpace(req.DefaultPrinter)
		if req.DefaultPrinter == "" {
			sendError(w, http.StatusBadRequest, "Nama printer tidak boleh kosong")
			return
		}

		// Atur di Windows GDI
		err := winapi.SetDefaultPrinter(req.DefaultPrinter)
		if err != nil {
			log.Printf("Gagal mengatur Windows default printer: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Simpan di config.json
		curCfg := engine.GetConfig()
		curCfg.DefaultPrinter = req.DefaultPrinter
		_ = engine.SaveConfig(curCfg)

		updateTrayPrinter(req.DefaultPrinter)

		log.Printf("Default printer berhasil diubah ke: %s\n", req.DefaultPrinter)
		sendSuccess(w, fmt.Sprintf("Default printer berhasil diubah ke: %s", req.DefaultPrinter), req.DefaultPrinter)
	})

	// Endpoint: Ubah Port (Disimpan ke config.json)
	mux.HandleFunc("/api/config/port", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req struct {
			Port int `json:"port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		if req.Port < 1024 || req.Port > 65535 {
			sendError(w, http.StatusBadRequest, "Port harus antara 1024 dan 65535")
			return
		}

		if err := switchPort(req.Port); err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Printf("Port baru (%d) berhasil aktif dan disimpan ke config.json\n", req.Port)
		sendSuccess(w, fmt.Sprintf("Port %d berhasil diterapkan dan disimpan ke config.json", req.Port), req.Port)
	})

	// Endpoint: Ubah Auto-Start Windows Booting
	mux.HandleFunc("/api/config/autostart", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		if err := winapi.SetAutoStart(req.Enabled); err != nil {
			log.Printf("Gagal mengatur auto-start Windows: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		msg := "Auto-start saat booting dinonaktifkan"
		if req.Enabled {
			msg = "Auto-start saat booting berhasil diaktifkan"
		}
		log.Println(msg)
		sendSuccess(w, msg, map[string]interface{}{
			"autostart": req.Enabled,
		})
	})

	// Endpoint: Ubah Lokasi Template Directory (Disimpan ke config.json)
	mux.HandleFunc("/api/config/template-dir", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		var req struct {
			TemplateDir string `json:"template_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		cleanedPath := engine.NormalizePath(strings.TrimSpace(req.TemplateDir))
		curCfg := engine.GetConfig()
		curCfg.TemplateDir = cleanedPath
		if err := engine.SaveConfig(curCfg); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal menyimpan konfigurasi: %v", err))
			return
		}

		log.Printf("Template directory berhasil diubah ke: %s\n", cleanedPath)
		sendSuccess(w, "Lokasi folder template berhasil disimpan ke config.json", map[string]string{
			"template_dir": cleanedPath,
			"active_dir":   engine.GetTemplateDir(),
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

	// Endpoint: Buka Folder Template di Windows Explorer
	mux.HandleFunc("/api/templates/open-dir", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		err := engine.OpenTemplateFolder()
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal membuka folder template: %v", err))
			return
		}

		sendSuccess(w, "Berhasil membuka folder template di Explorer", engine.GetTemplateDir())
	})

	// Endpoint: Upload Template File (.json)
	mux.HandleFunc("/api/templates/upload", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method harus POST")
			return
		}

		// Batasi ukuran file max 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			sendError(w, http.StatusBadRequest, "Ukuran file terlalu besar atau form-data salah")
			return
		}

		file, header, err := r.FormFile("template_file")
		if err != nil {
			sendError(w, http.StatusBadRequest, "Field form-data 'template_file' tidak ditemukan")
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".json") {
			sendError(w, http.StatusBadRequest, "Format file harus berupa .json")
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Gagal membaca konten file")
			return
		}
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

		var testTpl engine.DocumentTemplate
		if err := json.Unmarshal(data, &testTpl); err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("Konten bukan format JSON template yang valid: %v", err))
			return
		}

		destDir := engine.EnsureTemplateFolder()
		destPath := filepath.Join(destDir, header.Filename)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal menyimpan file template: %v", err))
			return
		}

		log.Printf("Template baru berhasil diunggah: %s\n", destPath)
		sendSuccess(w, fmt.Sprintf("Template '%s' berhasil diunggah ke folder template", header.Filename), map[string]string{
			"filename": header.Filename,
			"path":     destPath,
		})
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

		docs := req.GetData()
		if len(docs) == 0 {
			docs = []engine.DocumentData{{}}
		}

		targetPrinter := req.GetPrinterName()
		if targetPrinter == "" {
			targetPrinter, _ = winapi.GetDefaultPrinter()
		}

		tplName := req.GetTemplateName()
		if tplName == "" {
			sendError(w, http.StatusBadRequest, "template_name wajib diisi! Harap cantumkan nama template di payload JSON (contoh: {\"template_name\": \"cetak_struk_kasir_8\"})")
			return
		}

		tmplDir := req.GetTemplateDir()
		log.Printf("Menerima request cetak %d dokumen (template: '%s', dir: '%s') ke: %s\n", len(docs), tplName, tmplDir, targetPrinter)

		err = engine.PrintDocumentsWithDir(targetPrinter, tplName, tmplDir, docs)
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

		tplName := req.GetTemplateName()
		if tplName == "" {
			sendError(w, http.StatusBadRequest, "template_name wajib diisi! Harap cantumkan nama template di payload JSON (contoh: {\"template_name\": \"cetak_struk_kasir_8\"})")
			return
		}

		docs := req.GetData()
		targetPrinter := req.GetPrinterName()
		tmplDir := req.GetTemplateDir()
		log.Printf("Menerima request preview %d dokumen (template: '%s', dir: '%s')\n", len(docs), tplName, tmplDir)

		previews, err := engine.GeneratePreviewsWithDir(tplName, tmplDir, docs, targetPrinter)
		if err != nil {
			log.Printf("Gagal membuat preview: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Otomatis buka jendela Native Windows FastReport Preview di layar desktop (kecuali query ?window=false)
		if r.URL.Query().Get("window") != "false" {
			go func() {
				_, _ = previewgui.ShowFastReportPreview(targetPrinter, tplName, tmplDir, docs)
			}()
		}

		sendSuccess(w, fmt.Sprintf("Berhasil membuat preview %d dokumen", len(previews)), previews)
	}

	// Endpoint: FastReport Native Windows Desktop Preview Window
	previewWindowHandler := func(w http.ResponseWriter, r *http.Request) {
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

		tplName := req.GetTemplateName()
		if tplName == "" {
			sendError(w, http.StatusBadRequest, "template_name wajib diisi! Harap cantumkan nama template di payload JSON (contoh: {\"template_name\": \"cetak_struk_kasir_8\"})")
			return
		}

		docs := req.GetData()
		targetPrinter := req.GetPrinterName()
		tmplDir := req.GetTemplateDir()
		log.Printf("Membuka Native Windows FastReport Preview Window untuk template: '%s' (dir: '%s')\n", tplName, tmplDir)

		_, err = previewgui.ShowFastReportPreview(targetPrinter, tplName, tmplDir, docs)
		if err != nil {
			log.Printf("Gagal membuka window preview: %v\n", err)
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendSuccess(w, fmt.Sprintf("Jendela Native Windows FastReport Print Preview untuk template '%s' berhasil dibuka di desktop", tplName), map[string]interface{}{
			"template_name":  tplName,
			"template_dir":   tmplDir,
			"target_printer": targetPrinter,
			"document_count": len(docs),
		})
	}

	mux.HandleFunc("/api/print", printHandler)
	mux.HandleFunc("/api/preview", previewHandler)
	mux.HandleFunc("/api/preview-window", previewWindowHandler)

	httpServer = &http.Server{
		Handler: mux,
	}

	initialLn, err := net.Listen("tcp", fmt.Sprintf(":%d", activePort))
	if err != nil {
		log.Fatalf("Gagal mendengarkan di port %d: %v", activePort, err)
	}
	currentLn = initialLn

	go func(ln net.Listener) {
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			// listener closed
		}
	}(initialLn)

	// Buka juga port alternatif (8080 / 9000) agar Postman selalu terhubung di kedua port
	secondaryPort := 8080
	if activePort == 8080 {
		secondaryPort = 9000
	}
	if secLn, errSec := net.Listen("tcp", fmt.Sprintf(":%d", secondaryPort)); errSec == nil {
		log.Printf("📡 Port sekunder :%d aktif melayani request (kompatibilitas ganda port 8080 & 9000)\n", secondaryPort)
		go func(ln net.Listener) {
			if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				// listener closed
			}
		}(secLn)
	}

	if enableTray {
		cb := tray.GuiCallbacks{
			GetInstalledPrinters: func() ([]string, error) {
				return winapi.ListInstalledPrinters()
			},
			OnSetDefaultPrinter: func(printerName string) error {
				err := winapi.SetDefaultPrinter(printerName)
				if err != nil {
					return err
				}
				curCfg := engine.GetConfig()
				curCfg.DefaultPrinter = printerName
				_ = engine.SaveConfig(curCfg)
				updateTrayPrinter(printerName)
				return nil
			},
			OnTestPrint: func(printerName string) error {
				return engine.PrintDocuments(printerName, "label_kain_80x30", []engine.DocumentData{{}})
			},
			OnSavePort: func(p int) error {
				return switchPort(p)
			},
			OnOpenWebDashboard: func() {
				p := getActivePort()
				_ = exec.Command("cmd", "/c", "start", fmt.Sprintf("http://localhost:%d/dashboard", p)).Start()
			},
			OnOpenTemplateFolder: func() {
				_ = engine.OpenTemplateFolder()
			},
			OnAddTemplate: func(filePath string) (string, error) {
				return engine.AddTemplateFile(filePath)
			},
			OnOpenPreview: func() {
				curDef, _ := winapi.GetDefaultPrinter()
				_, _ = previewgui.ShowFastReportPreview(curDef, "cetak_struk_kasir_8", "", []engine.DocumentData{{}})
			},
		}

		var errTray error
		trayApp, errTray = tray.StartTray(activePort, defPrinter, cb, func() {
			log.Println("Menghentikan server dari Tray...")
			select {
			case <-exitChan:
			default:
				close(exitChan)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(ctx)
			os.Exit(0)
		})
		if errTray != nil {
			log.Printf("⚠️ Gagal mengaktifkan tray: %v\n", errTray)
		} else {
			defer trayApp.Stop()
			fmt.Println("📌 System Tray aktif di taskbar (klik kanan icon untuk menu)")
		}

		// Jika aplikasi dibuka dengan klik ganda di File Explorer, otomatis sembunyikan terminal ke background
		if winapi.IsLaunchedFromExplorer() {
			trayApp.HideConsole()
		}
	}

	// Jalankan memory trimmer background ala Delphi: menjaga RAM di Task Manager tetap ~3 - 5 MB
	winapi.StartMemoryTrimmer(30 * time.Second)

	fmt.Println("==================================================")
	fmt.Println("🚀 Knitto Print Service (Windows Native GDI)")
	fmt.Printf("📦 Mode Build         : %s\n", buildMode)
	fmt.Printf("📡 API Server berjalan di http://localhost:%d\n", activePort)
	fmt.Printf("🌐 Web Dashboard      : http://localhost:%d/dashboard\n", activePort)
	fmt.Printf("📂 Folder Template    : %s\n", engine.GetTemplateDir())
	fmt.Printf("🖨️  Default Printer    : %s\n", defPrinter)
	if enableTray {
		fmt.Println("💻 Mode               : System Tray Aktif (Background)")
	} else {
		fmt.Println("💻 Mode               : Silent Background Daemon (Tanpa Tray)")
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("Endpoints:")
	fmt.Printf("  GET  http://localhost:%d/dashboard (Web Dashboard)\n", activePort)
	fmt.Printf("  GET  http://localhost:%d/api/status\n", activePort)
	fmt.Printf("  GET  http://localhost:%d/api/printers\n", activePort)
	fmt.Printf("  GET  http://localhost:%d/api/templates\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/templates/open-dir\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/templates/upload\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/config/printer\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/config/port\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/preview\n", activePort)
	fmt.Printf("  POST http://localhost:%d/api/print\n", activePort)
	fmt.Println("==================================================")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-exitChan:
	case <-sigChan:
		log.Println("Menerima sinyal stop, menghentikan server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

