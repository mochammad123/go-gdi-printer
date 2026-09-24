# Panduan Setup & Menjalankan Aplikasi (Setelah Git Pull / Clone)

Dokumen ini berisi panduan cepat bagi developer untuk menyiapkan dan menjalankan **Print Service (Windows Native GDI)** setelah melakukan `git pull` atau `git clone`.

---

## 1. Prasyarat Sistem (Prerequisites)

Sebelum menjalankan aplikasi, pastikan komputer memenuhi persyaratan berikut:

| Komponen | Persyaratan | Keterangan |
| :--- | :--- | :--- |
| **Sistem Operasi** | Windows 10 / 11 / Server (x64) | Wajib Windows karena aplikasi mengakses langsung Windows GDI (`gdi32.dll`, `user32.dll`, `winspool.drv`). |
| **Golang** | Go 1.20+ (Disarankan Go 1.23+) | Compiler untuk menjalankan atau build Go. |
| **Printer Driver** | Driver printer terpasang di Windows | Printer fisik / virtual (Zebra, Thermal POS, atau Microsoft Print to PDF). |

Cek instalasi Go di terminal PowerShell:
```powershell
go version
```

---

## 2. Langkah Awal Setelah Git Pull / Clone

### Langkah 1: Buka Terminal
Buka PowerShell atau Command Prompt, lalu arahkan ke direktori proyek ini:
```powershell
# Contoh navigasi:
cd path\ke\folder\golang
```

### Langkah 2: Sinkronisasi Go Modules
Jalankan perintah berikut:
```powershell
go mod tidy
```
> [!NOTE]
> Proyek ini menggunakan **100% Go Standard Library** tanpa external 3rd-party library, sehingga proses ini berlangsung instan.

---

## 3. Cara Menjalankan Aplikasi

Pilih salah satu cara di bawah sesuai kebutuhan:

### Opsi A: Mode Development (Langsung Run tanpa Compile)
Sangat disarankan saat masih tahap pengembangan fitur, testing kode, atau debugging:

```powershell
# 1. Jalankan REST API Server (Port default 8080)
go run .

# 2. Jalankan di port custom (contoh: port 9000)
go run . -port 9000

# 3. Cek daftar printer Windows yang terpasang
go run . -printers

# 4. Cek daftar file template JSON yang tersedia
go run . -templates

# 5. Langsung tes cetak 1 dokumen dari template
go run . -test -template label_template.json
```

---

### Opsi B: Mode Build (Kompilasi ke File `.exe`)
Gunakan opsi ini jika ingin menghasilkan file executable standalone `print-service.exe`:

#### 1. Build Standar (Console Mode)
Akan memunculkan jendela terminal hitam sehingga log aktivitas cetak dan error terlihat:
```powershell
go build -o print-service.exe .
.\print-service.exe
```

#### 2. Build Silent (Background Mode / Tanpa Jendela Terminal)
Aplikasi berjalan di background secara senyap tanpa memunculkan jendela hitam:
```powershell
go build -ldflags "-H=windowsgui -s -w" -o print-service.exe .
.\print-service.exe
```

---

## 4. Memverifikasi Aplikasi Berjalan

Setelah server dijalankan (pada port `8080`), buka browser atau jalankan perintah:

* **URL Healthcheck**: `http://localhost:8080/`

**Respon Sukses (JSON)**:
```json
{
  "message": "Print Service online",
  "result": {
    "status": "online",
    "service": "Print Service (Windows Native GDI)",
    "default_printer": "ZDesigner ZD220-203dpi",
    "templates": [
      "label_template.json"
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

## 5. Ringkasan Parameter / Flag CLI

| Flag | Nilai Bawaan | Deskripsi |
| :--- | :--- | :--- |
| `-port <nomor>` | `8080` | Menentukan port listening REST API server |
| `-printers` | `false` | Menampilkan seluruh nama printer yang terpasang di Windows |
| `-templates` | `false` | Menampilkan daftar template label JSON |
| `-template <nama>` | `""` | Menentukan template yang digunakan |
| `-test` | `false` | Melakukan test print 1 dokumen sampel ke printer |
| `-info` | `false` | Menampilkan informasi resolusi (DPI) & margin printer default |
