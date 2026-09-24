# IDLIX Downloader (v0.2.0-alpha)

> High-performance IDLIX media scraper & automated Jellyfin/Plex downloader with mobile-first WebApp, built with Go, modern SQLite, Alpine.js, and Tailwind CSS.

---

## 🚀 Fitur Utama

- **⚡ Go Single Binary**: Backend ringan dan cepat berbasis Chi router + embedded web UI tanpa dependensi eksternal.
- **📱 Mobile-First WebApp**: Antarmuka responsif yang dioptimalkan untuk akses HP via Tailscale HTTPS (`https://bismillah.sokoke-everest.ts.net:8989`), lengkap dengan Bottom Navigation Bar dan Bottom Sheet Drawer (94vh).
- **📦 Batch Season Downloader**: Unduh seluruh episode dalam 1 season hanya dengan 1-tap.
- **📊 Real-time Progress Parsing**: Live speed (MB/s), persentase, dan estimasi waktu selesai (ETA) via Server-Sent Events (SSE).
- **🛡️ FFmpeg Integrity Probe**: Validasi file lokal yang sudah ada sebelum memulai download untuk mencegah duplikasi atau corrupt files.
- **💾 Disk Storage Monitor**: Pantau kapasitas sisa harddisk penyimpanan media (`Z:\Film`) langsung dari dashboard.
- **🔔 Notifikasi Telegram**: Pengiriman notifikasi otomatis ke bot/channel/topic Telegram saat proses unduhan selesai.
- **🧹 Queue Management**: Fitur Retry untuk tugas yang gagal dan tombol Bersihkan Riwayat yang aman.
- **🌐 Cloudflare DoH Bypass**: Integrasi DNS over HTTPS untuk resolusi mirror tanpa hambatan ISP.

---

## 🛠️ Tech Stack

- **Backend**: Go 1.22+ (Chi, SQLite WAL mode via `modernc.org/sqlite`)
- **Frontend**: HTML5, Tailwind CSS, Alpine.js, Lucide Icons (`//go:embed`)
- **Download Engine**: `N_m3u8DL-RE` + `FFmpeg`
- **Output Structure**: Standar Jellyfin / Plex (`Movies/{Title} ({Year})/` dan `Series/{Title} ({Year})/Season {N}/`)

---

## 📦 Menjalankan Aplikasi

### Build dari Source
```bash
go build -o idlixdownloader.exe ./cmd/server
./idlixdownloader.exe
```

Server default berjalan di port `:8989` dan otomatis terhubung dengan **Gateway Dev Hub**.

---

## ⚙️ Konfigurasi

File konfigurasi otomatis disimpan di `%LOCALAPPDATA%\iamnotpirates\idlixdownloader\config.json`:
```json
{
  "movies_dir": "Z:\\Film\\Movies",
  "series_dir": "Z:\\Film\\Series",
  "sub_lang": "Indonesian",
  "confirm_download": false,
  "base_url": "https://z2.idlixku.com",
  "max_concurrent_tasks": 1,
  "notify_telegram": false,
  "telegram_bot_token": "",
  "telegram_chat_id": ""
}
```

---

## 📄 Lisensi
MIT License.
