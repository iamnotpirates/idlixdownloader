# Graph Report - idlixdownloader  (2026-09-25)

## Corpus Check
- 23 files · ~17,083 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 2 file(s) not represented in the graph (top: (none) 1, .exe~ 1)

## Summary
- 254 nodes · 633 edges · 13 communities (12 shown, 1 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f1c09ba3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Server
- main
- Manager
- binmanager.go
- DB
- LoadConfig
- downloader.go
- go_pkg_net_http
- github.com/bwmarrin/discordgo.Session
- GetDiskSpace
- IDLIX Downloader (v0.2.0-alpha)
- IDLIX Downloader (Go Refactor)
- github.com/iamnotpirates/idlixdownloader

## God Nodes (most connected - your core abstractions)
1. `Manager` - 31 edges
2. `Server` - 26 edges
3. `DownloadTask` - 20 edges
4. `DB` - 19 edges
5. `LoadConfig()` - 18 edges
6. `respondJSON()` - 18 edges
7. `Bot` - 16 edges
8. `respondError()` - 14 edges
9. `main()` - 12 edges
10. `Bot` - 12 edges

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

## Communities (13 total, 1 thin omitted)

### Community 0 - "Server"
Cohesion: 0.25
Nodes (9): chi.Mux, net/http.Handler, net/http.Request, net/http.ResponseWriter, corsMiddleware(), respondError(), respondJSON(), sanitizeFilename() (+1 more)

### Community 1 - "main"
Cohesion: 0.11
Nodes (22): main(), sync.RWMutex, New(), New(), IdlixClient, New(), CurlClient, New() (+14 more)

### Community 2 - "Manager"
Cohesion: 0.14
Nodes (8): TaskListener, io.Reader, os/exec.Cmd, generateProgressBar(), Bot, Manager, DownloadStatus, DownloadTask

### Community 3 - "binmanager.go"
Cohesion: 0.27
Nodes (9): BinManager, go_pkg_archive_zip, go_pkg_io, copyBinary(), downloadFile(), extractZipBinary(), fileExists(), BinPaths (+1 more)

### Community 4 - "DB"
Cohesion: 0.20
Nodes (5): database/sql.DB, sync.Mutex, DB, Open(), TaskLog

### Community 5 - "LoadConfig"
Cohesion: 0.14
Nodes (16): DownloadThreadState, ExploreSession, go_pkg_bufio, github.com/bwmarrin/discordgo.Channel, github.com/bwmarrin/discordgo.ChannelType, github.com/bwmarrin/discordgo.Ready, time.Time, Bot (+8 more)

### Community 6 - "downloader.go"
Cohesion: 0.08
Nodes (45): go_pkg_context, go_pkg_database_sql, go_pkg_encoding_json, go_pkg_fmt, go_pkg_github_com_bwmarrin_discordgo, go_pkg_github_com_dustin_go_humanize, go_pkg_github_com_go_chi_chi_v5, go_pkg_github_com_go_chi_chi_v5_middleware (+37 more)

### Community 7 - "go_pkg_net_http"
Cohesion: 0.18
Nodes (8): go_pkg_bytes, go_pkg_embed, go_pkg_net_http, net/http.Client, net/http.HandlerFunc, Notifier, New(), Handler()

### Community 8 - "github.com/bwmarrin/discordgo.Session"
Cohesion: 0.42
Nodes (4): github.com/bwmarrin/discordgo.InteractionCreate, github.com/bwmarrin/discordgo.Session, Bot, sanitizeTitle()

### Community 9 - "GetDiskSpace"
Cohesion: 0.22
Nodes (6): github.com/bwmarrin/discordgo.MessageComponent, github.com/bwmarrin/discordgo.MessageEmbed, Bot, Bot, StorageInfo, GetDiskSpace()

### Community 10 - "IDLIX Downloader (v0.2.0-alpha)"
Cohesion: 0.25
Nodes (7): Build dari Source, 🚀 Fitur Utama, IDLIX Downloader (v0.2.0-alpha), ⚙️ Konfigurasi, 📄 Lisensi, 📦 Menjalankan Aplikasi, 🛠️ Tech Stack

### Community 11 - "IDLIX Downloader (Go Refactor)"
Cohesion: 0.33
Nodes (5): Akses, IDLIX Downloader (Go Refactor), Output Library Format, Ringkasan Project, Tech Stack

## Knowledge Gaps
- **12 isolated node(s):** `github.com/iamnotpirates/idlixdownloader`, `DownloadRequest`, `Event`, `Ringkasan Project`, `Tech Stack` (+7 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 54 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Manager` connect `Manager` to `Server`, `main`, `binmanager.go`, `DB`, `LoadConfig`, `downloader.go`, `go_pkg_net_http`?**
  _High betweenness centrality (0.213) - this node is a cross-community bridge._
- **Why does `Server` connect `Server` to `main`, `Manager`, `DB`, `LoadConfig`, `downloader.go`, `go_pkg_net_http`?**
  _High betweenness centrality (0.134) - this node is a cross-community bridge._
- **Why does `Bot` connect `LoadConfig` to `main`, `Manager`, `DB`, `downloader.go`, `github.com/bwmarrin/discordgo.Session`?**
  _High betweenness centrality (0.129) - this node is a cross-community bridge._
- **What connects `github.com/iamnotpirates/idlixdownloader`, `DownloadRequest`, `Event` to the rest of the system?**
  _12 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main` be split into smaller, more focused modules?**
  _Cohesion score 0.10685483870967742 - nodes in this community are weakly interconnected._
- **Should `Manager` be split into smaller, more focused modules?**
  _Cohesion score 0.1431451612903226 - nodes in this community are weakly interconnected._
- **Should `LoadConfig` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._