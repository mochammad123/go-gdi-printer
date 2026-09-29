package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"print-service/api"
	"print-service/engine"
	"print-service/previewgui"
	"print-service/tray"
	"print-service/winapi"
)

var (
	// DefaultTray menentukan apakah icon tray aktif secara default.
	// Bisa di-override saat build via -ldflags "-X main.DefaultTray=false"
	DefaultTray = "true"
)

func main() {
	// Jika dipanggil dari terminal / CMD, sambungkan ke console parent
	winapi.AttachParentConsole()

	// Daftarkan template dan aset sesuai mode build (Full Bundle vs Pisah / External)
	initEmbedded()
	engine.EnsureTemplateFolder()
	engine.EnsureAssetsFolder()

	defaultTrayBool := (strings.ToLower(DefaultTray) != "false" && DefaultTray != "0")

	port := flag.Int("port", 0, "Port untuk REST API server (default: 0 = membaca dari config.json atau default 8080)")
	templateFlag := flag.String("template", "", "Nama file template di folder template/ (wajib diisi jika tes, contoh: label_kain_80x30.json)")
	trayFlag := flag.Bool("tray", defaultTrayBool, "Aktifkan icon System Tray di taskbar Windows (default: sesuai mode build)")
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
	previewFlag := flag.Bool("preview", false, "Buka jendela FastReport Print Preview native Windows untuk melihat tampilan cetak")
	templateDirFlagCustom := flag.String("template-dir-path", "", "Folder custom untuk template (opsional, contoh: \\\\192.168.20.2\\Program_Holis\\faisal\\template)")
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

	// 3b. Opsi CLI: Buka jendela FastReport Print Preview Native Windows
	if *previewFlag {
		targetTpl := *templateFlag
		if strings.TrimSpace(targetTpl) == "" {
			targetTpl = "cetak_struk_kasir_8"
		}
		def, _ := winapi.GetDefaultPrinter()
		fmt.Printf("==================================================\n")
		fmt.Printf("🖥️  Membuka FastReport Print Preview (Native Windows GUI)\n")
		fmt.Printf("📄 Template: %s\n", targetTpl)
		if *templateDirFlagCustom != "" {
			fmt.Printf("📁 Folder  : %s\n", *templateDirFlagCustom)
		}
		fmt.Printf("🖨️  Printer : %s\n", def)
		fmt.Printf("==================================================\n")
		previewWin, err := previewgui.ShowFastReportPreview(def, targetTpl, *templateDirFlagCustom, []engine.DocumentData{{}})
		if err != nil {
			log.Fatalf("❌ Gagal membuka preview: %v", err)
		}
		<-previewWin.DoneChan
		fmt.Println("Jendela FastReport Preview ditutup.")
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
	api.SetupServer(activePort, *trayFlag, BuildMode)
}
