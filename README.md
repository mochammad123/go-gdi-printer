# Print Service (Windows Native GDI)

Aplikasi Backend Print Service berbasis **Golang murni (Standard Library)** yang menembak langsung ke **Windows GDI (`gdi32.dll`, `user32.dll`, & `winspool.drv`)** untuk menghasilkan cetakan dokumen, tiket, struk kasir, nota, barcode, label, dan gambar dengan resolusi tajam di berbagai tipe printer (Zebra / Thermal / Dot Matrix / Laser) tanpa perlu coding ZPL / ESC-POS.

- **Ukuran File**: Hanya ~6.87 MB (Single standalone `.exe` hasil optimasi `-ldflags="-s -w"`)
- **Penggunaan RAM**: Sangat ringan (~3 MB Go Heap Standby)
- **Dependencies**: 0 external library (Pure Go Standard Library & Windows Win32 API)
- **Kompatibilitas**: Windows 10 / 11 / Server (x64)
- **Fitur Utama**:
  - **Embedded Template & Asset System (`//go:embed`)**: Seluruh 16 file template JSON dan logo asset sudah di-embed langsung ke dalam binary `.exe`. Anda bisa membawa satu file `.exe` ini ke mana saja tanpa folder `template/`, dan service akan tetap berjalan normal! Jika folder `template/` belum ada, service otomatis mengekstraknya di samping file `.exe` agar mudah dilihat dan diedit.
  - **Native Windows Control Panel (Delphi-Style Form)**: Form desktop native Windows (`user32.dll` + `gdi32.dll`) dengan font `Segoe UI`, combobox printer, input port, tes cetak, dan status koneksi tanpa perlu membuka browser.
  - **Custom Windows System Tray**: Menggunakan icon logo resmi Knitto, menetap di taskbar Windows (background service).
  - **Single-Instance Protection**: Menjalankan `.exe` berkali-kali tidak akan bentrok / tabrakan port; otomatis mendeteksi bahwa service sudah aktif dan langsung membuka Control Panel.
  - **Embedded Web Dashboard** (`/dashboard`): Antarmuka web modern langsung dari Go tanpa dependency luar.
  - **Konfigurasi Permanen** (`config.json`): Menyimpan pilihan default printer & port agar tetap tersimpan.
  - **Base64 PNG Preview API** (`POST /api/preview`) untuk preview langsung di web / React modal sebelum mencetak.
  - **Image & Logo Printing** dengan transparent alpha-blending ke kertas putih.
  - **Barcode Generator** (Code 128 dengan kompensasi pemuaian panas printhead thermal).
  - **Multi-page & Multi-document Batch Printing**.

---

## 1. Prasyarat & Persiapan Awal (Setelah Clone / Pull)

### A. Prasyarat Sistem
* **Sistem Operasi**: Windows 10 / 11 / Server (x64) *(Aplikasi menggunakan Windows Native GDI & Winspool Driver)*.
* **Golang**: Pastikan Go sudah terpasang di sistem (Go 1.20+ disarankan).  
  Cek instalasi Go di terminal:
  ```powershell
  go version
  ```

### B. Langkah Pertama Setelah `git pull` / Clone
1. Buka PowerShell / Terminal di folder proyek.
2. Sinkronkan modul Go:
   ```powershell
   go mod tidy
   ```
   *(Catatan: Proyek ini menggunakan 100% **Go Standard Library**, tanpa dependency pihak ketiga sehingga tidak perlu download package luar)*.

---

## 2. Cara Menjalankan Aplikasi

Terdapat dua cara untuk menjalankan aplikasi ini:

### Opsi A: Mode Development (Langsung Run dengan `go run`)
Gunakan opsi ini saat mengembangkan aplikasi atau debugging tanpa perlu meng-compile file binary terlebih dahulu:
```powershell
# 1. Jalankan REST API Server + System Tray (Default: port 8080, tray aktif)
go run .

# 2. Jalankan tanpa System Tray (Headless / Terminal Standalone saja)
go run . -tray=false

# 3. Jalankan dengan port kustom (misal: port 9000)
go run . -port 9000

# 4. Cek daftar printer Windows yang terpasang
go run . -printers

# 5. Cek daftar template label yang tersedia
go run . -templates

# 6. Langsung tes cetak 1 dokumen dari template
go run . -test -template label_template.json
```

### Opsi B: Mode Build (Compile ke File Executable `.exe`)

#### 1. Rekomendasi Build: Standalone Tray App (Ukuran Ramping ~6.7 MB)
Compile dengan `-ldflags="-s -w"` untuk membuang simbol debug dan mengecilkan ukuran:
```powershell
# Compile aplikasi (Ramping ~6.7 MB)
go build -ldflags="-s -w" -o print-service.exe .

# Jalankan:
.\print-service.exe
```
> **Fitur System Tray pada mode ini:**
> - Icon printer otomatis muncul di pojok kanan bawah taskbar (System Tray).
> - **Klik 2x / Klik Kiri**: Membuka **Native Windows Control Panel** (Form Delphi-style).
> - **Klik Kanan**: Menampilkan menu:
>   - 🖥️ **Buka Control Panel (Windows Native)**
>   - 🌐 **Buka Web Dashboard (Browser)**
>   - 🖨️ Status Printer Aktif
>   - 💻 Sembunyikan / Tampilkan Terminal
>   - ❌ Keluar / Stop Service
> - Jika ingin mematikan tray dan murni jalan di terminal saja, jalankan dengan: `.\print-service.exe -tray=false`.

#### 2. Build Silent Background Tanpa Terminal Console (`-H=windowsgui`)
Jika ingin binary yang saat di-double click tidak pernah memunculkan jendela console hitam sama sekali:
```powershell
go build -ldflags="-H=windowsgui -s -w" -o print-service.exe .
```

---

## 3. Antarmuka Pengguna (UI) & REST API

Aplikasi menyediakan **Dua Pilihan Antarmuka**:

### A. Native Windows Control Panel (Delphi-Style Form)
* **Cara Membuka**: Klik 2x / klik kiri pada icon Tray, atau klik kanan icon tray -> **🖥️ Buka Control Panel (Windows)**.
* **Karakteristik**:
  - Tampilan form desktop native Windows murni (`user32.dll` + `gdi32.dll`) dengan font `Segoe UI`.
  - Sangat cepat, 0-latency, tanpa browser, tanpa dependency runtime tambahan.
  - Dropdown ComboBox berisi seluruh printer Windows yang terpasang.
  - Tombol **"💾 Jadikan Default Printer"**: Mengubah default printer Windows secara instan.
  - Tombol **"📄 Tes Cetak Label"**: Mengirim perintah cetak tes ke printer yang dipilih.
  - Input & Tombol **"💾 Simpan Port"**: Mengubah port REST API dan disimpan permanen.
  - Tombol **"🌐 Buka Web Dashboard"**: Membuka dashboard versi web browser.
  - Tombol **"⬇️ Sembunyikan ke Tray"** / tombol 'X': Menyembunyikan form kembali ke System Tray tanpa mematikan service.

---

### B. Web Dashboard UI (Tetap Tersedia)
* **Cara Membuka**: Buka di browser: `http://localhost:8080/dashboard` atau klik kanan icon tray -> **🌐 Buka Web Dashboard (Browser)**.
* **Fitur Dashboard**:
  - Tampilan web modern Knitto (100% offline).
  - Status koneksi real-time ("ONLINE & TERKONEKSI").
  - Ubah default printer, ganti port HTTP, cek template, dan tes cetak langsung dari browser.

---

### C. Cek Status Server (Healthcheck API)
* **URL**: `GET http://localhost:8080/api/status` (atau `GET /` dengan Header `Accept: application/json`)
* **Response**:
```json
{
  "message": "Print Service online",
  "result": {
    "status": "online",
    "service": "Print Service (Windows Native GDI)",
    "default_printer": "ZDesigner ZD220-203dpi",
    "templates": [
      "label_kain_80x30.json",
      "label_pecahan.json"
    ],
    "endpoints": [
      "GET  /",
      "GET  /api/printers",
      "GET  /api/templates",
      "POST /api/preview",
      "POST /api/print"
    ]
  }
}
```

---

### B. Dapatkan Preview Gambar Base64 PNG (Untuk Tampilan Web / React Modal)
* **URL**: `POST http://localhost:8080/api/preview`
* **Header**: `Content-Type: application/json`
* **Body**:
```json
{
  "PrinterName": "",
  "TemplateName": "label_kain_80x30",
  "Data": [
    {
      "Penerima": "ANDY LUCKITO",
      "Pemesan": "MAMANG RACING"
    }
  ]
}
```
* **Response**:
```json
{
  "message": "Berhasil membuat preview 1 dokumen",
  "result": [
    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
  ]
}
```

---

### C. Cetak Dokumen / Print (Silent Print)
* **URL**: `POST http://localhost:8080/api/print`
* **Header**: `Content-Type: application/json`
* **Body**:
```json
{
  "PrinterName": "",
  "TemplateName": "label_kain_80x30",
  "Data": [
    {
      "Penerima": "ANDY LUCKITO",
      "Pemesan": "MAMANG RACING"
    }
  ]
}
```
*(Catatan: Jika `PrinterName` dikosongkan `""`, aplikasi otomatis mencetak ke printer default).*

---

### D. Cek Daftar Printer yang Terpasang
* **URL**: `GET http://localhost:8080/api/printers`

---

### E. Cek Daftar File Template
* **URL**: `GET http://localhost:8080/api/templates`

---

## 4. Struktur Folder & Arsitektur Proyek

Proyek ini menggunakan **Standard Go Modular Architecture**:

```
golang/
│
├── main.go                 # Entry point: CLI flags, routing HTTP REST API
│
├── tray/                   # Native Windows System Tray (Shell_NotifyIconW & Popup Menu)
│   └── tray.go             # Tray icon, context menu, hide/show console, balloon notification
│
├── winapi/                 # Low-level Windows Win32 Driver & GDI
│   ├── win_gdi.go          # Winspool (printer), GDI32 (font, rect, pen, DIB), User32
│   └── barcode.go          # Native Code128 bar drawing ke HDC
│
├── engine/                 # Core Template, Rendering & Preview Engine
│   ├── template.go         # Load & parsing JSON template, variable interpolation
│   ├── printer.go          # Orchestration cetak multi-dokumen ke printer fisik
│   ├── image_loader.go     # Loader & decoder gambar (file lokal, URL, Base64)
│   └── preview.go          # Generator Base64 PNG untuk web/React modal
│
├── template/               # Koleksi file template JSON label
├── assets/                 # Koleksi file gambar / logo
├── go.mod                  # Go module definition (print-service)
├── .gitignore              # Konfigurasi file yang diabaikan Git
└── print-service.exe       # Standalone executable (dihasilkan setelah go build)
```
