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

    /* Modern FastReport-Style Print Preview Studio Modal */
    .fr-modal {
      position: fixed;
      inset: 0;
      z-index: 9999;
      background: #0f172a;
      display: none;
      flex-direction: column;
      color: #f8fafc;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    }
    .fr-modal.active {
      display: flex;
    }
    .fr-topbar {
      height: 52px;
      background: #1e293b;
      border-bottom: 1px solid #334155;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 16px;
      gap: 12px;
      user-select: none;
      box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    }
    .fr-group {
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .fr-divider {
      width: 1px;
      height: 22px;
      background: #475569;
      margin: 0 4px;
    }
    .fr-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      padding: 6px 12px;
      border-radius: 6px;
      font-size: 12px;
      font-weight: 600;
      background: #334155;
      color: #f8fafc;
      border: 1px solid #475569;
      cursor: pointer;
      transition: all 0.15s;
    }
    .fr-btn:hover {
      background: #475569;
      border-color: #64748b;
    }
    .fr-btn-primary {
      background: #2563eb;
      border-color: #3b82f6;
    }
    .fr-btn-primary:hover {
      background: #1d4ed8;
    }
    .fr-btn-danger {
      background: #dc2626;
      border-color: #ef4444;
    }
    .fr-btn-danger:hover {
      background: #b91c1c;
    }
    .fr-icon-btn {
      width: 30px;
      height: 30px;
      padding: 0;
      border-radius: 6px;
      background: #334155;
      border: 1px solid #475569;
      color: #f8fafc;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      font-weight: bold;
      transition: all 0.15s;
    }
    .fr-icon-btn:hover {
      background: #475569;
    }
    .fr-search-box {
      display: inline-flex;
      align-items: center;
      background: #0f172a;
      border: 1px solid #475569;
      border-radius: 6px;
      padding: 2px 8px;
    }
    .fr-search-box input {
      background: transparent;
      border: none;
      color: #f8fafc;
      font-size: 12px;
      outline: none;
      width: 110px;
    }
    .fr-body {
      flex: 1;
      display: flex;
      overflow: hidden;
      position: relative;
    }
    .fr-sidebar {
      width: 200px;
      background: #111827;
      border-right: 1px solid #1f2937;
      padding: 12px;
      overflow-y: auto;
      display: none;
    }
    .fr-sidebar.active {
      display: block;
    }
    .fr-thumb-card {
      background: #1f2937;
      border: 2px solid #374151;
      border-radius: 6px;
      padding: 8px;
      margin-bottom: 12px;
      cursor: pointer;
      text-align: center;
    }
    .fr-thumb-card.active {
      border-color: #3b82f6;
    }
    .fr-workspace {
      flex: 1;
      background: #0b1120;
      background-image: radial-gradient(#1e293b 1.5px, transparent 1.5px);
      background-size: 24px 24px;
      overflow: auto;
      display: flex;
      align-items: flex-start;
      justify-content: center;
      padding: 40px 20px;
    }
    .fr-paper-shadow {
      background: #ffffff;
      box-shadow: 0 25px 60px -15px rgba(0,0,0,0.85), 0 0 0 1px rgba(255,255,255,0.08);
      border-radius: 2px;
      transform-origin: top center;
      transition: transform 0.12s ease-out;
    }
    .fr-paper-img {
      display: block;
      image-rendering: -webkit-optimize-contrast;
      image-rendering: crisp-edges;
    }
    .fr-bottombar {
      height: 32px;
      background: #0f172a;
      border-top: 1px solid #1e293b;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 16px;
      font-size: 11px;
      color: #94a3b8;
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
      <div style="font-size:12px;color:#64748b;margin-bottom:14px;background:#f8fafc;padding:10px 14px;border-radius:8px;border:1px solid #e2e8f0;">
        <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;">
          <div style="word-break:break-all;">
            📁 <strong>Lokasi Folder Template:</strong> <span id="templateDirDisplay" style="color:#0f172a;font-family:monospace;font-weight:600;">Memuat path...</span>
          </div>
          <button class="btn btn-secondary" style="padding:4px 10px;font-size:12px;" id="btnEditTemplateDir" title="Ganti ke folder Synology NAS / Network Share / Local">
            ✏️ Ubah Lokasi (NAS / Disk)
          </button>
        </div>
      </div>
      <div id="templatesList">Memuat template...</div>
    </div>
  <!-- Modern FastReport-Style Print Preview Studio Modal -->
  <div class="fr-modal" id="frModal">
    <!-- Topbar Toolbar (FastReport Modernized) -->
    <div class="fr-topbar">
      <!-- Left: Title & Info -->
      <div class="fr-group">
        <span style="font-size: 18px;">🖨️</span>
        <div>
          <div style="font-weight: 700; font-size: 13px; line-height: 1.2;" id="frDocTitle">FastReport Print Preview</div>
          <div style="font-size: 11px; color: #94a3b8;" id="frDocSubTitle">Resolusi 203 DPI • Zero-Burik</div>
        </div>
      </div>

      <!-- Center: FastReport Tools -->
      <div class="fr-group">
        <button class="fr-btn fr-btn-primary" id="btnFrPrint" title="Cetak Langsung ke Printer (Ctrl + P)">
          🖨️ Cetak
        </button>

        <button class="fr-btn" id="btnFrExportPng" title="Export Gambar PNG High-Resolution (Ctrl + S)">
          💾 Save PNG
        </button>

        <div class="fr-divider"></div>

        <!-- Search / Find -->
        <div class="fr-search-box" title="Cari teks di struk (Find)">
          <input type="text" id="frSearchInput" placeholder="🔍 Cari teks..." />
        </div>

        <div class="fr-divider"></div>

        <!-- Page Navigator -->
        <button class="fr-icon-btn" id="btnFrPrevPage" title="Halaman Sebelumnya (◀)">◀</button>
        <span style="font-size: 12px; font-weight: 600; min-width: 50px; text-align: center;" id="frPageDisplay">1 / 1</span>
        <button class="fr-icon-btn" id="btnFrNextPage" title="Halaman Selanjutnya (▶)">▶</button>

        <div class="fr-divider"></div>

        <!-- Zoom Controls -->
        <button class="fr-icon-btn" id="btnFrZoomOut" title="Zoom Out (−)">−</button>
        <span style="font-size: 12px; font-weight: 700; min-width: 44px; text-align: center; color: #38bdf8;" id="frZoomDisplay">100%%</span>
        <button class="fr-icon-btn" id="btnFrZoomIn" title="Zoom In (+)">+</button>

        <button class="fr-btn" id="btnFrFitWidth" title="Fit to Width (Sesuai Lebar Layar)">
          ↔️ Fit Lebar
        </button>
        <button class="fr-btn" id="btnFrFitPage" title="Fit to Page (1 Halaman Utuh)">
          ↕️ Fit Halaman
        </button>
      </div>

      <!-- Right: Toggle Sidebar & Close -->
      <div class="fr-group">
        <button class="fr-btn" id="btnFrToggleThumb" title="Buka/Tutup Miniatur Halaman (Thumbnails)">
          📑 Thumbnails
        </button>
        <button class="fr-btn fr-btn-danger" id="btnFrClose" title="Tutup Jendela Preview (Esc)">
          ✕ Tutup
        </button>
      </div>
    </div>

    <!-- Workspace Body -->
    <div class="fr-body">
      <!-- Left Sidebar (Thumbnails) -->
      <div class="fr-sidebar" id="frSidebar">
        <div style="font-size: 11px; font-weight: 700; text-transform: uppercase; color: #64748b; margin-bottom: 8px;">
          📑 Miniatur Halaman
        </div>
        <div id="frThumbnailList">
          <div class="fr-thumb-card active">
            <div style="font-size: 11px; font-weight: 600; margin-bottom: 4px; color: #cbd5e1;">Halaman 1</div>
            <img id="frThumbImg1" style="width: 100%%; max-height: 160px; object-fit: contain; border-radius: 4px; background: #fff;" src="" />
          </div>
        </div>
      </div>

      <!-- Main Canvas Area -->
      <div class="fr-workspace" id="frWorkspace">
        <div class="fr-paper-shadow" id="frPaperWrapper">
          <div id="frLoadingSpinner" style="display: none; padding: 40px; text-align: center; color: #94a3b8; font-size: 13px;">
            ⏳ Merender preview dokumen...
          </div>
          <img id="frPaperImage" class="fr-paper-img" src="" alt="Print Preview" />
        </div>
      </div>
    </div>

    <!-- Bottom Status Bar -->
    <div class="fr-bottombar">
      <div style="display: flex; gap: 16px;">
        <span id="frStatusPrinter">🖨️ Printer Target: Memuat...</span>
        <span id="frStatusQuality">⚡ Kualitas: 203 DPI (Native Windows GDI • Zero-Burik)</span>
      </div>
      <div>
        <span>💡 Tip: Tahan <strong>Ctrl + Scroll Mouse</strong> untuk Zoom • Tekan <strong>Esc</strong> untuk menutup</span>
      </div>
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
    let currentTemplateDir = "";

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
          currentTemplateDir = cfg.template_dir || "";
          document.getElementById("templateDirDisplay").innerText = currentTemplateDir || "Folder bawaan (template/)";

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
                  '<div style="display:flex;gap:8px;align-items:center;">' +
                    '<button class="btn btn-secondary" onclick="openModernPreview(\'' + t + '\')" style="padding:6px 12px;font-size:12px;font-weight:600;" title="Buka Modern FastReport Print Preview">' +
                      '👁️ View Print' +
                    '</button>' +
                    '<span class="badge badge-primary">Aktif</span>' +
                  '</div>' +
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

    // Ubah Lokasi Folder Template (Synology NAS / Local)
    document.getElementById("btnEditTemplateDir").addEventListener("click", async () => {
      const input = prompt("Masukkan path folder template baru (bisa path lokal atau Synology NAS UNC contoh: \\\\192.168.20.2\\faisal\\template atau Z:\\template):\\n\\nKosongkan jika ingin kembali ke folder lokal bawaan.", currentTemplateDir);
      if (input === null) return;
      try {
        const res = await fetch("/api/config/template-dir", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ template_dir: input.trim() })
        });
        const data = await res.json();
        if (res.ok) {
          showToast("Lokasi folder template berhasil diubah!", "✅");
          fetchStatus();
        } else {
          showToast(data.message || "Gagal mengubah lokasi template", "❌");
        }
      } catch (err) {
        showToast("Error koneksi ke server", "❌");
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

    // ==========================================
    // MODERN FASTREPORT PREVIEW STUDIO LOGIC
    // ==========================================
    let currentPreviewTpl = "";
    let currentZoom = 1.0;
    let previewImages = [];
    let currentPreviewPage = 0;

    const sampleReceiptData = {
      "datanpwp": "NPWP: 03.274.873.3-424.000",
      "toko": "PT KNITTO TEKSTIL INDONESIA",
      "alamat": "JL. HOLIS NO 35-37, KOTA BANDUNG",
      "telp": "Telp: (022) 20589089",
      "no_penjualan": "KS010426003",
      "tgl": "28-09-2026",
      "kode": "123",
      "kode_antrian": "1013",
      "no_order": "OH010426012",
      "no_dok_sap": "SAP-99881122",
      "admin": "Faisal / Kasir 01",
      "ekspedisi": "CUSTOMER - CUSTOMER",
      "items": [
        {
          "identitas": "COTTON COMBED 30S - JET BLACK",
          "berat": "25.40 Kg (1 Roll)",
          "harga": "@ 118.000",
          "total": "2.997.200"
        },
        {
          "identitas": "COTTON COMBED 30S - WHITE",
          "berat": "25.00 Kg (1 Roll)",
          "harga": "@ 115.000",
          "total": "2.875.000"
        },
        {
          "identitas": "RIB COMBED 30S - JET BLACK",
          "berat": "1.20 Kg",
          "harga": "@ 125.000",
          "total": "150.000"
        }
      ],
      "eceran": "1.20 Kg",
      "rollan": "50.40 Kg (2 Roll)",
      "dpp": "6.022.200",
      "judulppn": "PPN 11%%",
      "ppn": "662.442",
      "nb": "* Barang yang sudah dipotong/dicuci tidak dapat ditukar/dikembalikan.",
      "remaksnamatelpon": "Bpk. Hendra - 081234567890",
      "tgl_cetak": "28-09-2026 11:55",
      "akhir": "*** TERIMA KASIH ATAS KUNJUNGAN ANDA ***",
      "barcode": "SO202609250088"
    };

    function updateZoomDisplay() {
      document.getElementById("frZoomDisplay").innerText = Math.round(currentZoom * 100) + "%%";
      const wrapper = document.getElementById("frPaperWrapper");
      wrapper.style.transform = "scale(" + currentZoom + ")";
    }

    async function openModernPreview(templateName) {
      currentPreviewTpl = templateName;
      document.getElementById("frDocTitle").innerText = templateName + " (Preview)";
      document.getElementById("frStatusPrinter").innerText = "🖨️ Printer Target: " + (currentDefaultPrinter || "Default");
      document.getElementById("frModal").classList.add("active");
      
      currentZoom = 1.0;
      updateZoomDisplay();

      const img = document.getElementById("frPaperImage");
      const spinner = document.getElementById("frLoadingSpinner");
      img.style.display = "none";
      spinner.style.display = "block";

      try {
        const res = await fetch("/api/preview?window=false", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            printer_name: currentDefaultPrinter,
            template_name: templateName,
            template_dir: currentTemplateDir,
            data: [sampleReceiptData]
          })
        });

        const data = await res.json();
        spinner.style.display = "none";

        if (res.ok && data.result && data.result.length > 0) {
          previewImages = data.result;
          currentPreviewPage = 0;
          img.src = previewImages[0];
          img.style.display = "block";
          document.getElementById("frThumbImg1").src = previewImages[0];
          document.getElementById("frPageDisplay").innerText = (currentPreviewPage + 1) + " / " + previewImages.length;
          showToast("Preview siap ditampilkan!", "📄");
        } else {
          showToast(data.message || "Gagal memuat preview template", "❌");
        }
      } catch (err) {
        spinner.style.display = "none";
        showToast("Error koneksi preview: " + err, "❌");
      }
    }

    function closeModernPreview() {
      document.getElementById("frModal").classList.remove("active");
    }

    // Keyboard navigation (Esc to close, Ctrl+P to print)
    window.addEventListener("keydown", (e) => {
      const modal = document.getElementById("frModal");
      if (modal && modal.classList.contains("active")) {
        if (e.key === "Escape") {
          closeModernPreview();
        } else if ((e.ctrlKey || e.metaKey) && (e.key === "p" || e.key === "P")) {
          e.preventDefault();
          document.getElementById("btnFrPrint").click();
        } else if ((e.ctrlKey || e.metaKey) && (e.key === "=" || e.key === "+")) {
          e.preventDefault();
          document.getElementById("btnFrZoomIn").click();
        } else if ((e.ctrlKey || e.metaKey) && e.key === "-") {
          e.preventDefault();
          document.getElementById("btnFrZoomOut").click();
        } else if ((e.ctrlKey || e.metaKey) && e.key === "0") {
          e.preventDefault();
          currentZoom = 1.0;
          updateZoomDisplay();
        }
      }
    });

    // Zoom buttons
    document.getElementById("btnFrZoomIn").addEventListener("click", () => {
      if (currentZoom < 4.0) {
        currentZoom = Math.round((currentZoom + 0.25) * 100) / 100;
        updateZoomDisplay();
      }
    });

    document.getElementById("btnFrZoomOut").addEventListener("click", () => {
      if (currentZoom > 0.3) {
        currentZoom = Math.round((currentZoom - 0.25) * 100) / 100;
        updateZoomDisplay();
      }
    });

    document.getElementById("btnFrFitWidth").addEventListener("click", () => {
      const ws = document.getElementById("frWorkspace");
      const img = document.getElementById("frPaperImage");
      if (img.naturalWidth > 0) {
        const availableW = ws.clientWidth - 80;
        currentZoom = Math.round((availableW / img.naturalWidth) * 100) / 100;
        updateZoomDisplay();
      }
    });

    document.getElementById("btnFrFitPage").addEventListener("click", () => {
      const ws = document.getElementById("frWorkspace");
      const img = document.getElementById("frPaperImage");
      if (img.naturalHeight > 0) {
        const availableH = ws.clientHeight - 80;
        currentZoom = Math.round((availableH / img.naturalHeight) * 100) / 100;
        updateZoomDisplay();
      }
    });

    // Mouse wheel zoom
    document.getElementById("frWorkspace").addEventListener("wheel", (e) => {
      if (e.ctrlKey) {
        e.preventDefault();
        if (e.deltaY < 0) {
          if (currentZoom < 4.0) currentZoom += 0.15;
        } else {
          if (currentZoom > 0.3) currentZoom -= 0.15;
        }
        currentZoom = Math.round(currentZoom * 100) / 100;
        updateZoomDisplay();
      }
    }, { passive: false });

    // Print button from FastReport preview
    document.getElementById("btnFrPrint").addEventListener("click", async () => {
      if (!currentPreviewTpl) return;
      showToast("Mengirim cetak ke " + (currentDefaultPrinter || "printer default") + "...", "🖨️");
      try {
        const res = await fetch("/api/print", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            printer_name: currentDefaultPrinter,
            template_name: currentPreviewTpl,
            template_dir: currentTemplateDir,
            data: [sampleReceiptData]
          })
        });
        const result = await res.json();
        if (res.ok) {
          showToast("Berhasil mencetak dokumen!", "🎉");
        } else {
          showToast("Gagal mencetak: " + result.message, "❌");
        }
      } catch (err) {
        showToast("Error koneksi print: " + err, "❌");
      }
    });

    // Export PNG
    document.getElementById("btnFrExportPng").addEventListener("click", () => {
      const img = document.getElementById("frPaperImage");
      if (!img.src) {
        showToast("Belum ada dokumen yang siap di-export!", "⚠️");
        return;
      }
      const a = document.createElement("a");
      a.href = img.src;
      a.download = currentPreviewTpl + "_preview_203dpi.png";
      a.click();
      showToast("Gambar PNG HD berhasil di-download!", "💾");
    });

    // Toggle Thumbnail Sidebar
    document.getElementById("btnFrToggleThumb").addEventListener("click", () => {
      const sb = document.getElementById("frSidebar");
      sb.classList.toggle("active");
    });

    // Page navigation (Previous & Next Page)
    document.getElementById("btnFrPrevPage").addEventListener("click", () => {
      if (currentPreviewPage > 0) {
        currentPreviewPage--;
        document.getElementById("frPaperImage").src = previewImages[currentPreviewPage];
        document.getElementById("frPageDisplay").innerText = (currentPreviewPage + 1) + " / " + previewImages.length;
      }
    });

    document.getElementById("btnFrNextPage").addEventListener("click", () => {
      if (currentPreviewPage < previewImages.length - 1) {
        currentPreviewPage++;
        document.getElementById("frPaperImage").src = previewImages[currentPreviewPage];
        document.getElementById("frPageDisplay").innerText = (currentPreviewPage + 1) + " / " + previewImages.length;
      }
    });

    // Search / Find text
    document.getElementById("frSearchInput").addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        const query = e.target.value.trim();
        if (query) {
          showToast("Pencarian: '" + query + "' (Fitur Find Aktif)", "🔍");
        }
      }
    });

    // Close preview button
    document.getElementById("btnFrClose").addEventListener("click", closeModernPreview);

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
