# Graph Report - idlixdownloader  (2026-09-25)

## Corpus Check
- 24 files · ~18,655 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 1 file(s) not represented in the graph (top: (none) 1)

## Summary
- 268 nodes · 670 edges · 18 communities (17 shown, 1 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS · INFERRED: 1 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `716a4370`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Server
- Bot
- Manager
- binmanager.go
- DB
- LoadConfig
- server.go
- go_pkg_net_http
- github.com/bwmarrin/discordgo.Session
- github.com/bwmarrin/discordgo.MessageComponent
- IDLIX Downloader (v0.2.0-alpha)
- IDLIX Downloader (Go Refactor)
- github.com/iamnotpirates/idlixdownloader
- downloader.go
- go_pkg_time
- models.go
- go_pkg_fmt
- db.go

## God Nodes (most connected - your core abstractions)
1. `Manager` - 31 edges
2. `Server` - 26 edges
3. `DB` - 22 edges
4. `DownloadTask` - 20 edges
5. `LoadConfig()` - 18 edges
6. `respondJSON()` - 18 edges
7. `Bot` - 16 edges
8. `Bot` - 14 edges
9. `respondError()` - 14 edges
10. `main()` - 12 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [EXTRACTED]
  cmd/server/main.go → internal/binmanager/binmanager.go
- `main()` --calls--> `GetAppDataDir()`  [EXTRACTED]
  cmd/server/main.go → internal/config/config.go
- `main()` --calls--> `LoadConfig()`  [EXTRACTED]
  cmd/server/main.go → internal/config/config.go
- `main()` --calls--> `Open()`  [EXTRACTED]
  cmd/server/main.go → internal/db/db.go
- `main()` --calls--> `New()`  [EXTRACTED]
  cmd/server/main.go → internal/bot/bot.go

## Import Cycles
- None detected.

## Communities (18 total, 1 thin omitted)

### Community 0 - "Server"
Cohesion: 0.25
Nodes (9): chi.Mux, net/http.Handler, net/http.Request, net/http.ResponseWriter, corsMiddleware(), respondError(), respondJSON(), sanitizeFilename() (+1 more)

### Community 1 - "Bot"
Cohesion: 0.08
Nodes (27): DownloadThreadState, ExploreSession, main(), go_pkg_testing, github.com/bwmarrin/discordgo.Channel, github.com/bwmarrin/discordgo.ChannelType, github.com/bwmarrin/discordgo.Ready, net/http.HandlerFunc (+19 more)

### Community 2 - "Manager"
Cohesion: 0.16
Nodes (7): TaskListener, io.Reader, os/exec.Cmd, Bot, Manager, DownloadStatus, DownloadTask

### Community 3 - "binmanager.go"
Cohesion: 0.29
Nodes (8): BinManager, go_pkg_archive_zip, go_pkg_io, copyBinary(), downloadFile(), extractZipBinary(), fileExists(), New()

### Community 4 - "DB"
Cohesion: 0.16
Nodes (5): database/sql.DB, sync.Mutex, DB, Open(), TaskLog

### Community 5 - "LoadConfig"
Cohesion: 0.38
Nodes (9): go_pkg_bufio, DefaultConfig(), GetAppDataDir(), GetBinDir(), GetConfigPath(), LoadConfig(), LoadEnv(), SaveConfig() (+1 more)

### Community 6 - "server.go"
Cohesion: 0.17
Nodes (14): go_pkg_context, go_pkg_github_com_go_chi_chi_v5, go_pkg_github_com_go_chi_chi_v5_middleware, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_binmanager, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_bot, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_db, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_downloader, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_extractor (+6 more)

### Community 7 - "go_pkg_net_http"
Cohesion: 0.25
Nodes (6): go_pkg_bytes, go_pkg_embed, go_pkg_net_http, net/http.Client, Notifier, New()

### Community 8 - "github.com/bwmarrin/discordgo.Session"
Cohesion: 0.33
Nodes (6): github.com/bwmarrin/discordgo.ButtonStyle, github.com/bwmarrin/discordgo.InteractionCreate, github.com/bwmarrin/discordgo.Session, getBtnStyle(), Bot, sanitizeTitle()

### Community 9 - "github.com/bwmarrin/discordgo.MessageComponent"
Cohesion: 0.28
Nodes (6): github.com/bwmarrin/discordgo.MessageComponent, github.com/bwmarrin/discordgo.MessageEmbed, Bot, Bot, StorageInfo, GetDiskSpace()

### Community 10 - "IDLIX Downloader (v0.2.0-alpha)"
Cohesion: 0.25
Nodes (7): Build dari Source, 🚀 Fitur Utama, IDLIX Downloader (v0.2.0-alpha), ⚙️ Konfigurasi, 📄 Lisensi, 📦 Menjalankan Aplikasi, 🛠️ Tech Stack

### Community 11 - "IDLIX Downloader (Go Refactor)"
Cohesion: 0.33
Nodes (5): Akses, IDLIX Downloader (Go Refactor), Output Library Format, Ringkasan Project, Tech Stack

### Community 13 - "downloader.go"
Cohesion: 0.17
Nodes (12): go_pkg_github_com_iamnotpirates_idlixdownloader_internal_notifier, go_pkg_math, go_pkg_os, go_pkg_os_exec, go_pkg_regexp, go_pkg_runtime, go_pkg_syscall, go_pkg_unsafe (+4 more)

### Community 14 - "go_pkg_time"
Cohesion: 0.23
Nodes (10): go_pkg_github_com_bwmarrin_discordgo, go_pkg_github_com_dustin_go_humanize, go_pkg_github_com_google_uuid, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_models, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_system, go_pkg_log, go_pkg_path_filepath, go_pkg_strconv (+2 more)

### Community 15 - "models.go"
Cohesion: 0.24
Nodes (7): EpisodeInfo, MovieDetails, SeriesDetails, StreamSources, SeasonInfo, SubtitleTrack, SeasonDownloadRequest

### Community 16 - "go_pkg_fmt"
Cohesion: 0.29
Nodes (7): go_pkg_encoding_json, go_pkg_fmt, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_httpclient, go_pkg_github_com_puerkitobio_goquery, go_pkg_net_url, go_pkg_strings, Event

### Community 17 - "db.go"
Cohesion: 0.40
Nodes (4): go_pkg_database_sql, go_pkg_github_com_iamnotpirates_idlixdownloader_internal_config, go_pkg_modernc_org_sqlite, go_pkg_sync

## Knowledge Gaps
- **12 isolated node(s):** `github.com/iamnotpirates/idlixdownloader`, `DownloadRequest`, `Event`, `Ringkasan Project`, `Tech Stack` (+7 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 59 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Manager` connect `Manager` to `Server`, `Bot`, `DB`, `go_pkg_net_http`, `downloader.go`?**
  _High betweenness centrality (0.199) - this node is a cross-community bridge._
- **Why does `Server` connect `Server` to `Bot`, `Manager`, `DB`, `LoadConfig`, `server.go`?**
  _High betweenness centrality (0.129) - this node is a cross-community bridge._
- **Why does `Bot` connect `Bot` to `Manager`, `DB`, `LoadConfig`, `server.go`, `github.com/bwmarrin/discordgo.Session`?**
  _High betweenness centrality (0.128) - this node is a cross-community bridge._
- **What connects `github.com/iamnotpirates/idlixdownloader`, `DownloadRequest`, `Event` to the rest of the system?**
  _12 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Bot` be split into smaller, more focused modules?**
  _Cohesion score 0.07804878048780488 - nodes in this community are weakly interconnected._