---
tags: [idlixdownloader, media-automation, go, mobile-ui, status-active]
created: 2026-09-23
---

# IDLIX Downloader (Go Refactor)

## Ringkasan Project
Aplikasi scraper & multi-threaded video stream downloader untuk IDLIX yang dioptimasi khusus untuk **Mobile Web / Telegram Mini App UI** dan kontrol remote home server.

## Tech Stack
- **Backend**: Go 1.27 (`net/http`, `chi`, `modernc.org/sqlite` pure Go)
- **Database**: SQLite WAL mode di `%LOCALAPPDATA%\iamnotpirates\idlixdownloader\idlix.db`
- **Network / Bypass**: DoH (Cloudflare 1.1.1.1 DNS-over-HTTPS) via `curl.exe` wrapper
- **Downloader Engine**: `N_m3u8DL-RE.exe` + `ffmpeg.exe` di `%LOCALAPPDATA%\iamnotpirates\idlixdownloader\bin\`
- **Frontend UI**:
  - **Discord Bot**: Native UI di kategori `iDLiX Downloader` (`🔍・search-request`, `📥・active-downloads` dgn Auto-Threads per task download, `📚・download-history`).
  - **Mobile Web**: Mobile-First UI (Tailwind CSS, Alpine.js) embedded via `go:embed` on port `8989`.
- **Real-Time Updates**: Discord Message Edit (2.5s throttled) & Server-Sent Events (SSE) `/api/events`

## Output Library Format
- **Movies**: `{MoviesDir}/{Title} ({Year})/{Title} ({Year}).mp4`
- **TV Series**: `{SeriesDir}/{Title} ({Year})/Season {N}/{Title} - S{N:02d}E{E:02d} - {EpisodeTitle}.mp4`
- **Subtitles**: Otomatis dikonversi dari WebVTT ke standard `.srt` berdampingan dengan file video.

## Akses
- **Port Web**: `8989`
- **Tailscale HTTPS**: `https://bismillah.sokoke-everest.ts.net:8989`
