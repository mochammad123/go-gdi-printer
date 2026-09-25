package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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
	"print-service/tray"
	"print-service/winapi"
)

func main() {
	// Jika dipanggil dari terminal / CMD, sambungkan ke console parent
	winapi.AttachParentConsole()

	// Daftarkan template dan aset sesuai mode build (Full Bundle vs Pisah / External)
	initEmbedded()
	engine.EnsureTemplateFolder()
	engine.EnsureAssetsFolder()

	port := flag.Int("port", 0, "Port untuk REST API server (default: 0 = membaca dari config.json atau default 8080)")
	templateFlag := flag.String("template", "", "Nama file template di folder template/ (wajib diisi jika tes, contoh: label_kain_80x30.json)")
	trayFlag := flag.Bool("tray", true, "Aktifkan icon System Tray di taskbar Windows (default: true, gunakan -tray=false untuk mode non-tray)")
	testPrint := flag.Bool("test", false, "Cetak 1 dokumen langsung dari template untuk tes")
	infoFlag := flag.Bool("info", false, "Tampilkan informasi ukuran kertas & resolusi printer default")
	listPrinters := flag.Bool("printers", false, "Tampilkan daftar printer yang terpasang")
	listTemplates := flag.Bool("templates", false, "Tampilkan daftar file template yang tersedia di folder template/")
	addTemplateFlag := flag.String("add-template", "", "Salin file JSON template baru ke folder template service (contoh: -add-template \"C:\\path\\fileA.json\")")
	openTemplatesFlag := flag.Bool("open-templates", false, "Buka folder template service di Windows Explorer")
	templateDirFlag := flag.Bool("template-dir", false, "Tampilkan path lokasi folder template pada PC ini")
	autostartFlag := flag.String("autostart", "", "Atur auto-start saat Windows booting ('true' atau 'false')")
	portableFlag := flag.Bool("portable", false, "Jalankan di folder saat ini tanpa auto-install ke %LocalAppData%")
	installFlag := flag.Bool("install", false, "Paksa pasang/deploy ke %LocalAppData% dan buat shortcut")
	flag.Parse()

	// 0. Opsi CLI: Atur auto-start
	if strings.TrimSpace(*autostartFlag) != "" {
		enable := strings.EqualFold(strings.TrimSpace(*autostartFlag), "true") || strings.TrimSpace(*autostartFlag) == "1"
		err := winapi.SetAutoStart(enable)
		if err != nil {
			log.Fatalf("❌ Gagal mengatur auto-start: %v", err)
		}
		if enable {
			fmt.Println("✅ Auto-start saat Windows booting berhasil diaktifkan.")
		} else {
			fmt.Println("✅ Auto-start saat Windows booting berhasil dinonaktifkan.")
		}
		return
	}

	// 0. Opsi CLI: Tambah template baru
	if strings.TrimSpace(*addTemplateFlag) != "" {
		dest, err := engine.AddTemplateFile(*addTemplateFlag)
		if err != nil {
			log.Fatalf("❌ Gagal menyalin template: %v", err)
		}
		fmt.Println("==================================================")
		fmt.Println("✅ Berhasil menambahkan template baru!")
		fmt.Printf("📁 Lokasi File: %s\n", dest)
		fmt.Println("==================================================")
		return
	}

	// 0. Opsi CLI: Buka folder template di Windows Explorer
	if *openTemplatesFlag {
		dir := engine.GetTemplateDir()
		fmt.Printf("📂 Membuka folder template di Explorer: %s\n", dir)
		_ = engine.OpenTemplateFolder()
		return
	}

	// 0. Opsi CLI: Tampilkan path folder template
	if *templateDirFlag {
		fmt.Println(engine.GetTemplateDir())
		return
	}

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

	// 4. Single-Instance Check (Cegah multiple instance / port collision)
	hMutex, isAlreadyRunning, err := winapi.AcquireSingleInstance("Local\\KnittoPrintServiceMutex")
	if err == nil && isAlreadyRunning {
		fmt.Println("==================================================")
		fmt.Println("⚠️  Knitto Print Service sudah aktif di background!")
		if !tray.BringNativeGuiToFront() {
			cfg := engine.LoadConfig(0)
			targetPort := cfg.Port
			if targetPort == 0 {
				targetPort = 8080
			}
			fmt.Printf("🌐 Membuka Dashboard di browser: http://localhost:%d/dashboard\n", targetPort)
			_ = exec.Command("cmd", "/c", "start", fmt.Sprintf("http://localhost:%d/dashboard", targetPort)).Start()
		} else {
			fmt.Println("🖥️  Membuka Control Panel Windows...")
		}
		fmt.Println("==================================================")
		time.Sleep(500 * time.Millisecond)
		return
	}
	if hMutex != 0 {
		defer winapi.ReleaseMutexHandle(hMutex)
	}

	// 4b. Auto-Deploy ke %LOCALAPPDATA%\Knitto\PrintService jika dijalankan dari Downloads / Desktop / luar
	if (*installFlag || !winapi.IsRunningFromPermanentDir()) && !*portableFlag {
		targetExe, errDeploy := winapi.DeployToPermanentDirectory()
		if errDeploy == nil {
			// Lepas mutex saat ini agar executable di folder permanen bisa meng-acquire mutex
			if hMutex != 0 {
				winapi.ReleaseMutexHandle(hMutex)
				hMutex = 0
			}
			// Jalankan executable permanen di background
			_ = exec.Command(targetExe).Start()

			// Tampilkan popup informasi berhasil dipasang ke user
			winapi.ShowDeploySuccessDialog(winapi.GetPermanentAppDir())
			return
		} else {
			log.Printf("⚠️ Gagal auto-deploy: %v (melanjutkan di folder saat ini)\n", errDeploy)
		}
	}

	// Pastikan shortcut Desktop & Startup terdaftar
	winapi.EnsureShortcuts()

	// Daftarkan auto-start Windows secara otomatis agar selalu online seperti sistem Delphi
	if !winapi.IsAutoStartEnabled() {
		_ = winapi.SetAutoStart(true)
	}

	// 5. Muat Konfigurasi (config.json)
	cfg := engine.LoadConfig(*port)
	activePort := cfg.Port
	if *port > 0 {
		activePort = *port
	}

	// 6. Jalankan Server REST API & System Tray
	setupAPIServer(activePort, *trayFlag)
}

func setupAPIServer(port int, enableTray bool) {
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
			"build_mode":      BuildMode,
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
				"POST /api/preview",
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
			"build_mode":      BuildMode,
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
	fmt.Printf("📦 Mode Build         : %s\n", BuildMode)
	fmt.Printf("📡 API Server berjalan di http://localhost:%d\n", activePort)
	fmt.Printf("🌐 Web Dashboard      : http://localhost:%d/dashboard\n", activePort)
	fmt.Printf("📂 Folder Template    : %s\n", engine.GetTemplateDir())
	fmt.Printf("🖨️  Default Printer    : %s\n", defPrinter)
	if enableTray {
		fmt.Println("💻 Mode               : System Tray Aktif (Background)")
	} else {
		fmt.Println("💻 Mode               : Headless / Terminal Standalone (-tray=false)")
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
