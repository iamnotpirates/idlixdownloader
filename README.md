# 🏴‍☠️ IDLIX Downloader (Rust + Svelte)

High-performance, lightweight, and modern web-based scraper and multi-threaded stream downloader for IDLIX.

- **Backend**: Rust (Axum, Tokio, Reqwest, Rust-Embed)
- **Frontend**: Svelte 5 + Tailwind CSS + Lucide Icons (Bun / Vite)
- **Single Executable**: Total size **~8 MB** with embedded web UI.
- **Auto-healing Dependencies**: Automatically checks and downloads `ffmpeg` and `N_m3u8DL-RE` binaries if missing.

---

## 🚀 Quick Start

### 1. Run Binary Directly
```bash
cargo run --release
```
Buka browser di **http://localhost:8989**.

### 2. Development Mode (Hot Reload)
Terminal 1 (Backend Rust API):
```bash
cargo run
```

Terminal 2 (Frontend Svelte Vite):
```bash
cd frontend
bun dev
```
Buka **http://localhost:5173** (otomatis proxy ke backend Rust di port 8989).

---

## 🛠️ Build Single Executable
```bash
# 1. Build frontend
cd frontend
bun run build
cd ..

# 2. Compile release binary
cargo build --release
```
Hasil executable mandiri ada di `target/release/idlixdownloader.exe`.
