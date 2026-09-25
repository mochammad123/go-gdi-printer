package engine

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"print-service/tray"
)

var dashboardHTML string

func init() {
	iconBase64 := base64.StdEncoding.EncodeToString(tray.DefaultIconBytes)
	dashboardHTML = fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Knitto Print Service - Dashboard</title>
  <link rel="icon" type="image/png" href="data:image/png;base64,%s">
  <style>
    :root {
      --primary: #2563eb;
      --primary-hover: #1d4ed8;
      --success: #10b981;
      --success-bg: #ecfdf5;
      --success-border: #a7f3d0;
      --warning: #f59e0b;
      --danger: #ef4444;
      --bg: #f8fafc;
      --card-bg: #ffffff;
      --text: #0f172a;
      --text-muted: #64748b;
      --border: #e2e8f0;
      --radius: 12px;
    }
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    }
    body {
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding-bottom: 40px;
    }
    .navbar {
      background: #ffffff;
      border-bottom: 1px solid var(--border);
      padding: 14px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      position: sticky;
      top: 0;
      z-index: 100;
      box-shadow: 0 1px 3px rgba(0,0,0,0.04);
    }
    .navbar-brand {
      display: flex;
      align-items: center;
      gap: 12px;
      text-decoration: none;
      color: inherit;
    }
    .navbar-brand img {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    .brand-title {
      font-size: 18px;
      font-weight: 700;
      color: #0f172a;
    }
    .brand-subtitle {
      font-size: 12px;
      color: var(--text-muted);
    }
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 14px;
      background: var(--success-bg);
      border: 1px solid var(--success-border);
      color: #065f46;
      border-radius: 9999px;
      font-size: 13px;
      font-weight: 600;
    }
    .pulse-dot {
      width: 8px;
      height: 8px;
      background-color: var(--success);
      border-radius: 50%%;
      box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
      animation: pulse 1.8s infinite;
    }
    @keyframes pulse {
      0%% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
      }
      70%% {
        transform: scale(1);
        box-shadow: 0 0 0 8px rgba(16, 185, 129, 0);
      }
      100%% {
        transform: scale(0.95);
        box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
      }
    }
    .container {
      max-width: 960px;
      margin: 28px auto 0;
      padding: 0 20px;
    }
    .alert-banner {
      background: var(--success-bg);
      border: 1px solid var(--success-border);
      border-radius: var(--radius);
      padding: 16px 20px;
      margin-bottom: 24px;
      display: flex;
      align-items: center;
      gap: 16px;
    }
    .alert-icon {
      font-size: 28px;
      flex-shrink: 0;
    }
    .alert-content h4 {
      font-size: 15px;
      color: #065f46;
      font-weight: 700;
      margin-bottom: 2px;
    }
    .alert-content p {
      font-size: 13px;
      color: #047857;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(440px, 1fr));
      gap: 20px;
      margin-bottom: 24px;
    }
    @media (max-width: 640px) {
      .grid {
        grid-template-columns: 1fr;
      }
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 24px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.02);
    }
    .card-title {
      font-size: 16px;
      font-weight: 700;
      margin-bottom: 6px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .card-desc {
      font-size: 13px;
      color: var(--text-muted);
      margin-bottom: 18px;
    }
    .form-group {
      margin-bottom: 16px;
    }
    .form-label {
      display: block;
      font-size: 13px;
      font-weight: 600;
      margin-bottom: 6px;
      color: #334155;
    }
    .form-select, .form-input {
      width: 100%%;
      padding: 10px 14px;
      border: 1px solid var(--border);
      border-radius: 8px;
      font-size: 14px;
      background: #fdfdfd;
      color: #0f172a;
      outline: none;
      transition: border-color 0.2s;
    }
    .form-select:focus, .form-input:focus {
      border-color: var(--primary);
      background: #ffffff;
    }
    .btn-group {
      display: flex;
      gap: 10px;
      margin-top: 14px;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      padding: 10px 18px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 600;
      border: none;
      cursor: pointer;
      transition: background-color 0.15s, transform 0.05s;
    }
    .btn:active {
      transform: scale(0.98);
    }
    .btn-primary {
      background: var(--primary);
      color: #ffffff;
    }
    .btn-primary:hover {
      background: var(--primary-hover);
    }
    .btn-secondary {
      background: #e2e8f0;
      color: #334155;
    }
    .btn-secondary:hover {
      background: #cbd5e1;
    }
    .info-list {
      list-style: none;
    }
    .info-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 0;
      border-bottom: 1px solid #f1f5f9;
      font-size: 13px;
    }
    .info-item:last-child {
      border-bottom: none;
    }
    .info-label {
      color: var(--text-muted);
    }
    .info-value {
      font-weight: 600;
      color: #0f172a;
    }
    .badge {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 6px;
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
    }
    .badge-primary {
      background: #eff6ff;
      color: #1d4ed8;
      border: 1px solid #bfdbfe;
    }
    .badge-success {
      background: #ecfdf5;
      color: #047857;
      border: 1px solid #a7f3d0;
    }
    .toast {
      position: fixed;
      bottom: 24px;
      right: 24px;
      background: #0f172a;
      color: #ffffff;
      padding: 12px 20px;
      border-radius: 10px;
      font-size: 13px;
      font-weight: 500;
      box-shadow: 0 10px 25px rgba(0,0,0,0.2);
      display: flex;
      align-items: center;
      gap: 10px;
      transform: translateY(100px);
      opacity: 0;
      transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
      z-index: 999;
    }
    .toast.show {
      transform: translateY(0);
      opacity: 1;
    }
    .template-card {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      background: #f8fafc;
      border: 1px solid var(--border);
      border-radius: 8px;
      margin-bottom: 10px;
    }
    .template-info strong {
      font-size: 14px;
      display: block;
    }
    .template-info span {
      font-size: 12px;
      color: var(--text-muted);
    }
  </style>
</head>
<body>

  <!-- Navbar -->
  <nav class="navbar">
    <a href="/" class="navbar-brand">
      <img src="data:image/png;base64,%s" alt="Knitto Logo" />
      <div>
        <div class="brand-title">Knitto Print Service</div>
        <div class="brand-subtitle">Windows Native GDI Tray Agent</div>
      </div>
    </a>
    <div class="status-badge" id="serviceBadge">
      <span class="pulse-dot"></span>
      <span id="serviceStatusText">ONLINE & TERKONEKSI</span>
    </div>
  </nav>

  <div class="container">
    <!-- Active Printer Notification Banner -->
    <div class="alert-banner" id="printerAlertBanner">
      <div class="alert-icon">🖨️</div>
      <div class="alert-content">
        <h4>Printer Terkoneksi Saat Ini</h4>
        <p id="currentPrinterDisplay">Memuat status printer...</p>
      </div>
    </div>

    <div class="grid">
      <!-- Card: Pilih / Ubah Default Printer -->
      <div class="card">
        <div class="card-title">⚙️ Konfigurasi Default Printer</div>
        <div class="card-desc">Pilih printer yang akan digunakan sebagai target cetak utama oleh print-service.</div>
        
        <div class="form-group">
          <label class="form-label" for="printerSelect">Daftar Printer Terpasang di Windows:</label>
          <select id="printerSelect" class="form-select">
            <option value="">-- Memuat printer... --</option>
          </select>
        </div>

        <div class="btn-group">
          <button class="btn btn-primary" id="btnSavePrinter">
            💾 Jadikan Default Printer
          </button>
          <button class="btn btn-secondary" id="btnTestPrint">
            📄 Tes Cetak Dokumen
          </button>
        </div>
      </div>

      <!-- Card: Konfigurasi Port Server & Startup -->
      <div class="card">
        <div class="card-title">🌐 Konfigurasi Jaringan & Boot</div>
        <div class="card-desc">Atur port REST API server lokal dan otomatisasi saat Windows booting.</div>

        <div class="form-group">
          <label class="form-label" for="portInput">Nomor Port HTTP:</label>
          <input type="number" id="portInput" class="form-input" min="1024" max="65535" value="8080" />
        </div>

        <div class="btn-group">
          <button class="btn btn-primary" id="btnSavePort">
            💾 Simpan Port
          </button>
        </div>

        <div style="margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--border); font-size: 13px; color: #047857; font-weight: 600; display: flex; align-items: center; gap: 8px;">
          <span>✓</span> Auto-Start Windows: Selalu Aktif di Background (Booting)
        </div>
      </div>
    </div>

    <!-- Status & Info Card -->
    <div class="card" style="margin-bottom: 24px;">
      <div class="card-title">ℹ️ Informasi Sistem Service</div>
      <div class="card-desc">Status operasional background print service saat ini.</div>

      <ul class="info-list">
        <li class="info-item">
          <span class="info-label">Mode Aplikasi</span>
          <span class="info-value"><span class="badge badge-success">Windows System Tray (Background)</span></span>
        </li>
        <li class="info-item">
          <span class="info-label">Windows Auto-Start</span>
          <span class="info-value" id="infoAutoStartDisplay"><span class="badge badge-success">Aktif (Auto-Boot)</span></span>
        </li>
        <li class="info-item">
          <span class="info-label">URL REST API Lokal</span>
          <span class="info-value" id="infoApiUrl">http://localhost:8080</span>
        </li>
        <li class="info-item">
          <span class="info-label">Konsumsi Memori RAM</span>
          <span class="info-value">~3 MB (Go Heap Runtime Standby)</span>
        </li>
        <li class="info-item">
          <span class="info-label">Status Background Service</span>
          <span class="info-value">Berjalan terus di background hingga proses di-kill / keluar dari tray.</span>
        </li>
      </ul>
    </div>

    <!-- Templates Card -->
    <div class="card">
      <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:12px;margin-bottom:12px;">
        <div>
          <div class="card-title" style="margin-bottom:4px;">📋 Template Dokumen Tersedia</div>
          <div class="card-desc" style="margin-bottom:0;">File template JSON yang siap digunakan untuk mencetak atau preview label.</div>
        </div>
        <div style="display:flex;gap:8px;flex-wrap:wrap;">
          <button class="btn btn-secondary" id="btnOpenTemplatesDir" title="Buka folder template di Windows Explorer">
            📂 Buka Folder Template
          </button>
          <label class="btn btn-primary" style="cursor:pointer;" title="Upload file template JSON baru">
            ➕ Upload Template (.json)
            <input type="file" id="inputUploadTemplate" accept=".json" style="display:none;" />
          </label>
        </div>
      </div>
      <div style="font-size:12px;color:#64748b;margin-bottom:14px;background:#f8fafc;padding:8px 12px;border-radius:6px;border:1px solid #e2e8f0;word-break:break-all;">
        📁 <strong>Lokasi Folder Template PC Ini:</strong> <span id="templateDirDisplay" style="color:#0f172a;font-family:monospace;font-weight:600;">Memuat path...</span>
      </div>
      <div id="templatesList">Memuat template...</div>
    </div>
  </div>

  <!-- Toast Notification -->
  <div class="toast" id="toast">
    <span id="toastIcon">🔔</span>
    <span id="toastMessage">Pesan</span>
  </div>

  <script>
    let currentDefaultPrinter = "";
    let currentPort = window.location.port || 8080;

    function showToast(message, icon = "✅") {
      const toast = document.getElementById("toast");
      const toastMsg = document.getElementById("toastMessage");
      const toastIcon = document.getElementById("toastIcon");
      toastMsg.innerText = message;
      toastIcon.innerText = icon;
      toast.classList.add("show");
      setTimeout(() => {
        toast.classList.remove("show");
      }, 3500);
    }

    async function fetchStatus() {
      try {
        const res = await fetch("/api/status");
        if (!res.ok) return;
        const data = await res.json();
        if (data.result) {
          const cfg = data.result;
          currentDefaultPrinter = cfg.default_printer || "";
          document.getElementById("currentPrinterDisplay").innerHTML = 
            'Terhubung ke: <strong>' + (currentDefaultPrinter || "Belum dipilih") + '</strong>';
          document.getElementById("infoApiUrl").innerText = window.location.origin;
          currentPort = cfg.port || currentPort;
          if (document.activeElement !== document.getElementById("portInput")) {
            document.getElementById("portInput").value = currentPort;
          }
          if (cfg.template_dir) {
            document.getElementById("templateDirDisplay").innerText = cfg.template_dir;
          }

          const badge = document.getElementById("infoAutoStartDisplay");
          if (badge) {
            badge.innerHTML = '<span class="badge badge-success">Selalu Aktif (Auto-Boot)</span>';
          }

          // Render templates
          if (Array.isArray(cfg.templates)) {
            const container = document.getElementById("templatesList");
            if (cfg.templates.length === 0) {
              container.innerHTML = '<div style="font-size:13px;color:#94a3b8;">Tidak ada file template ditemukan di folder template/</div>';
            } else {
              container.innerHTML = cfg.templates.map(t => 
                '<div class="template-card">' +
                  '<div class="template-info">' +
                    '<strong>📄 ' + t + '</strong>' +
                    '<span>Template Label Siap Cetak & Preview</span>' +
                  '</div>' +
                  '<span class="badge badge-primary">Aktif</span>' +
                '</div>'
              ).join('');
            }
          }
        }
      } catch (err) {
        console.error("Fetch status error:", err);
      }
    }

    async function fetchPrinters() {
      try {
        const res = await fetch("/api/printers");
        if (!res.ok) return;
        const data = await res.json();
        const select = document.getElementById("printerSelect");
        select.innerHTML = "";

        const printers = data.result.printers || [];
        const def = data.result.default || currentDefaultPrinter;
        currentDefaultPrinter = def;

        document.getElementById("currentPrinterDisplay").innerHTML = 
          'Terhubung ke: <strong>' + (def || "Tidak ada printer default") + '</strong>';

        if (printers.length === 0) {
          select.innerHTML = '<option value="">(Tidak ada printer terpasang)</option>';
          return;
        }

        printers.forEach(p => {
          const opt = document.createElement("option");
          opt.value = p;
          opt.innerText = p + (p === def ? " (Default Saat Ini)" : "");
          if (p === def) opt.selected = true;
          select.appendChild(opt);
        });
      } catch (err) {
        console.error("Fetch printers error:", err);
      }
    }

    // Set Default Printer
    document.getElementById("btnSavePrinter").addEventListener("click", async () => {
      const select = document.getElementById("printerSelect");
      const selectedPrinter = select.value;
      if (!selectedPrinter) {
        showToast("Pilih salah satu printer terlebih dahulu!", "⚠️");
        return;
      }

      try {
        const res = await fetch("/api/config/printer", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ default_printer: selectedPrinter })
        });
        const result = await res.json();
        if (res.ok) {
          showToast("Berhasil mengubah default printer ke: " + selectedPrinter, "✅");
          fetchPrinters();
        } else {
          showToast(result.message || "Gagal mengubah printer", "❌");
        }
      } catch (err) {
        showToast("Error koneksi ke server", "❌");
      }
    });

    // Test Print
    document.getElementById("btnTestPrint").addEventListener("click", async () => {
      const select = document.getElementById("printerSelect");
      const targetPrinter = select.value || currentDefaultPrinter;

      showToast("Mengirim perintah cetak tes ke " + targetPrinter + "...", "🖨️");

      try {
        const res = await fetch("/api/print", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            PrinterName: targetPrinter,
            TemplateName: "label_kain_80x30",
            Data: [{}]
          })
        });
        const result = await res.json();
        if (res.ok) {
          showToast("Cetak tes berhasil dikirim!", "🎉");
        } else {
          showToast("Gagal tes cetak: " + result.message, "⚠️");
        }
      } catch (err) {
        showToast("Gagal menghubungi server print", "❌");
      }
    });

    // Save Port
    document.getElementById("btnSavePort").addEventListener("click", async () => {
      const portVal = parseInt(document.getElementById("portInput").value, 10);
      if (!portVal || portVal < 1024 || portVal > 65535) {
        showToast("Nomor port harus antara 1024 dan 65535!", "⚠️");
        return;
      }

      if (portVal === currentPort) {
        showToast("Server sudah berjalan pada port " + portVal, "ℹ️");
        return;
      }

      try {
        const res = await fetch("/api/config/port", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ port: portVal })
        });
        const result = await res.json();
        if (res.ok) {
          showToast("Port berhasil diubah ke " + portVal + "! Mengalihkan ke port baru...", "🚀");
          currentPort = portVal;
          document.getElementById("infoApiUrl").innerText = window.location.protocol + "//" + window.location.hostname + ":" + portVal;
          setTimeout(() => {
            window.location.href = window.location.protocol + "//" + window.location.hostname + ":" + portVal + "/dashboard";
          }, 1500);
        } else {
          showToast(result.message || "Gagal menyimpan port", "❌");
        }
      } catch (err) {
        showToast("Error koneksi ke server", "❌");
      }
    });

    // Open Template Directory in Explorer
    document.getElementById("btnOpenTemplatesDir").addEventListener("click", async () => {
      try {
        const res = await fetch("/api/templates/open-dir", { method: "POST" });
        const data = await res.json();
        if (res.ok) {
          showToast("Membuka folder template di Windows Explorer...", "📂");
        } else {
          showToast(data.message || "Gagal membuka folder template", "❌");
        }
      } catch (err) {
        showToast("Gagal menghubungi server", "❌");
      }
    });

    // Upload Template JSON
    document.getElementById("inputUploadTemplate").addEventListener("change", async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      if (!file.name.toLowerCase().endsWith(".json")) {
        showToast("Hanya file format .json yang diperbolehkan!", "⚠️");
        e.target.value = "";
        return;
      }

      showToast("Mengunggah template: " + file.name + "...", "⏳");
      const formData = new FormData();
      formData.append("template_file", file);

      try {
        const res = await fetch("/api/templates/upload", {
          method: "POST",
          body: formData
        });
        const data = await res.json();
        if (res.ok) {
          showToast("Template " + file.name + " berhasil disimpan!", "🎉");
          fetchStatus();
        } else {
          showToast(data.message || "Gagal mengunggah template", "❌");
        }
      } catch (err) {
        showToast("Gagal mengunggah template ke server", "❌");
      } finally {
        e.target.value = "";
      }
    });

    // Initial load
    fetchStatus();
    fetchPrinters();

    // Periodic live healthcheck
    setInterval(fetchStatus, 4000);
  </script>
</body>
</html>`, iconBase64, iconBase64)
}

// ServeDashboard returns the HTML Dashboard UI
func ServeDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}
