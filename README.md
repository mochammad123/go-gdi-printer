# Print Service (Windows Native GDI)

Aplikasi Backend Print Service berbasis **Golang murni (Standard Library)** yang menembak langsung ke **Windows GDI (`gdi32.dll`, `user32.dll`, & `winspool.drv`)** untuk menghasilkan cetakan dokumen, tiket, struk kasir, nota, barcode, label, dan gambar dengan resolusi tajam di berbagai tipe printer (Zebra / Thermal / Dot Matrix / Laser) tanpa perlu coding ZPL / ESC-POS.

- **Ukuran File**: Hanya ~9.3 MB (Single standalone `.exe`)
- **Dependencies**: 0 external library (Pure Go Standard Library)
- **Kompatibilitas**: Windows 10 / 11 / Server (x64)
- **Fitur Utama**:
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
# 1. Jalankan REST API Server (Port default: 8080)
go run .

# 2. Jalankan dengan port kustom (misal: port 9000)
go run . -port 9000

# 3. Cek daftar printer Windows yang terpasang
go run . -printers

# 4. Cek daftar template label yang tersedia
go run . -templates

# 5. Langsung tes cetak 1 dokumen dari template
go run . -test -template label_template.json
```

### Opsi B: Mode Build (Compile ke File Executable `.exe`)
Jika ingin mendistribusikan atau menjalankan sebagai file standalone executable (karena file `.exe` diabaikan oleh `.gitignore`):

#### 1. Build Standar (Dengan Jendela Console / Log Debugging)
Log aktivitas cetak dan error akan langsung terlihat di jendela terminal:
```powershell
# Compile aplikasi
go build -o print-service.exe .

# Jalankan file binary
.\print-service.exe
```

#### 2. Build Silent / Background Service (Tanpa Jendela Terminal Hitam)
Gunakan flag `-ldflags "-H=windowsgui -s -w"`. Saat file `.exe` dijalankan atau diklik dua kali, aplikasi akan langsung berjalan di background secara senyap:
```powershell
# Compile mode GUI / silent
go build -ldflags "-H=windowsgui -s -w" -o print-service.exe .

# Jalankan file binary
.\print-service.exe
```

---

## 3. Dokumentasi REST API

Format respon API standar:
```json
{
  "message": "Pesan informasi",
  "result": null // atau data objek/array
}
```

---

### A. Cek Status Server (Healthcheck)
* **URL**: `GET http://localhost:8080/`
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
