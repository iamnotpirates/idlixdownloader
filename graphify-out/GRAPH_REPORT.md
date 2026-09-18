# Graph Report - idlixdownloader  (2026-09-11)

## Corpus Check
- 42 files · ~77,021 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 355 nodes · 588 edges · 28 communities (23 shown, 4 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 51 edges (avg confidence: 0.96)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Svelte Frontend Application State
- Axum REST API Routes
- Async Download Engine & Pipeline
- Domain Models & Configuration
- NPM Package Scripts & Config
- HTTP Scraper & Curl Client
- TypeScript Node Compiler Config
- TypeScript App Compiler Config
- External Binary Lifecycle Manager
- Frontend Dependencies & Tooling
- IDLIX Stream & Metadata Extractor
- Settings Interface & UI Preferences
- Download Location Confirmation Modal
- System Architecture & Documentation
- Social & Documentation Icons Sprite
- Windows System Tray Integration
- Catalog Browser & Theme Design
- Downloads Manager UI Dashboard
- Media Search & Results View
- Series Media Modal & Episodes
- Brand Vector Logo & Favicon
- Hero Isometric Graphic Artwork
- Backend Main Entrypoint Lifecycle
- TypeScript Workspace Project Config
- Application Desktop Icon Asset
- Vite Framework Brand Logo
- Rust Cargo Package Manifest

## God Nodes (most connected - your core abstractions)
1. `DownloadManager` - 19 edges
2. `compilerOptions` - 15 edges
3. `AppState` - 15 edges
4. `IdlixClient` - 14 edges
5. `AppError` - 14 edges
6. `CurlClient` - 12 edges
7. `process_stream_output()` - 11 edges
8. `compilerOptions` - 10 edges
9. `search_media()` - 10 edges
10. `BinManager` - 9 edges

## Surprising Connections (you probably didn't know these)
- `Embedded Single Binary Architecture` --references--> `Frontend Entrypoint HTML`  [INFERRED]
  README.md → frontend/index.html
- `AppState` --references--> `DownloadManager`  [EXTRACTED]
  src/routes.rs → src/downloader.rs
- `IdlixClient` --references--> `CurlClient`  [EXTRACTED]
  src/extractor.rs → src/http_client.rs
- `start_download()` --references--> `CreateDownloadRequest`  [EXTRACTED]
  src/routes.rs → src/models.rs
- `Frontend Entrypoint HTML` --conceptually_related_to--> `Frontend Svelte + TS + Vite Architecture`  [INFERRED]
  frontend/index.html → frontend/README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Embedded Web Application Packaging** — readme_md_idlix_downloader, readme_md_embedded_single_binary, frontend_index_html_root [INFERRED 0.85]

## Communities (28 total, 4 thin omitted)

### Community 0 - "Svelte Frontend Application State"
Cohesion: 0.07
Nodes (21): handleDownloadStarted(), handleKeyDown(), handleSearch(), loadCatalog(), syncRoute(), createDownloadSocket(), fetchCatalog(), searchMedia() (+13 more)

### Community 1 - "Axum REST API Routes"
Cohesion: 0.14
Nodes (39): Json, Query, Response, AppError, AppState, extract_stream_sources(), ExtractRequest, get_catalog() (+31 more)

### Community 2 - "Async Download Engine & Pipeline"
Cohesion: 0.14
Nodes (28): Mutex, Path, R, Receiver, RwLock, Sender, BinPaths, download_and_convert_vtt_to_srt() (+20 more)

### Community 3 - "Domain Models & Configuration"
Cohesion: 0.13
Nodes (21): EpisodeInfo, SeasonInfo, AppConfig, CreateDownloadRequest, DownloadStatus, DownloadTask, EpisodeInfo, MediaItem (+13 more)

### Community 4 - "NPM Package Scripts & Config"
Cohesion: 0.09
Nodes (22): name, private, scripts, build, check, dev, preview, type (+14 more)

### Community 5 - "HTTP Scraper & Curl Client"
Cohesion: 0.16
Nodes (15): CurlClient, Default, Option, PathBuf, Result, Self, String, Value (+7 more)

### Community 6 - "TypeScript Node Compiler Config"
Cohesion: 0.12
Nodes (16): compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, noEmit, noFallthroughCasesInSwitch (+8 more)

### Community 7 - "TypeScript App Compiler Config"
Cohesion: 0.14
Nodes (13): compilerOptions, allowArbitraryExtensions, allowJs, checkJs, module, moduleDetection, noEmit, target (+5 more)

### Community 8 - "External Binary Lifecycle Manager"
Cohesion: 0.36
Nodes (6): BinManager, Option, PathBuf, Result, Self, String

### Community 9 - "Frontend Dependencies & Tooling"
Cohesion: 0.15
Nodes (13): devDependencies, clsx, lucide-svelte, svelte, svelte-check, @sveltejs/vite-plugin-svelte, tailwind-merge, tailwindcss (+5 more)

### Community 10 - "IDLIX Stream & Metadata Extractor"
Cohesion: 0.23
Nodes (8): MovieDetails, Option, Result, Self, SeriesDetails, StreamSources, String, test_extract_moana()

### Community 11 - "Settings Interface & UI Preferences"
Cohesion: 0.27
Nodes (10): Download Location Confirmation Option, Settings Dark Theme Interface, Application Settings Header, Settings Tab UI Screenshot, IDLIX Base Mirror URL Setting, Movies Directory Setting, Application Navigation Bar, Save Changes Button (+2 more)

### Community 12 - "Download Location Confirmation Modal"
Cohesion: 0.25
Nodes (9): Cancel Action Button (Batal), Manual Folder Selector with Browse Button, Default Folder Option (Z:\Film\Movies), Jellyfin/Plex Automatic Folder Structure Preview, Modal Header (Konfirmasi Download - Film: Moana 2), Confirm Start Download Button (Mulai Download), Storage Location Selection (Pilih Lokasi Penyimpanan), Subtitle Language Dropdown (Subtitle Bahasa: Indonesian) (+1 more)

### Community 13 - "System Architecture & Documentation"
Cohesion: 0.25
Nodes (8): Frontend Entrypoint HTML, Frontend Svelte + TS + Vite Architecture, Anti-Blocking DNS-over-HTTPS (DoH), Embedded Single Binary Architecture, IDLIX Downloader, Lossless Remuxing to MP4, Smart Folder Organization, Zero-Config Dependency Binaries

### Community 14 - "Social & Documentation Icons Sprite"
Cohesion: 0.29
Nodes (7): Bluesky Icon Symbol, Discord Icon Symbol, Documentation Icon Symbol, GitHub Icon Symbol, Social Icon Symbol, SVG Icon Sprite Sheet, X (Twitter) Icon Symbol

### Community 15 - "Windows System Tray Integration"
Cohesion: 0.33
Nodes (6): open_folder_path(), Box, Error, Result, String, run_tray_loop()

### Community 16 - "Catalog Browser & Theme Design"
Cohesion: 0.40
Nodes (6): Dark Mode Design System, Browse Catalog UI Screenshot, Catalog Media Card, Popular and Featured Media Grid, Application Navigation Bar, Search Hero Section

### Community 17 - "Downloads Manager UI Dashboard"
Cohesion: 0.40
Nodes (6): Download Queue Container, Empty Downloads Queue State, Download Manager Header, Downloads Tab UI Screenshot, Application Navigation Bar, Total Tasks Counter Badge

### Community 18 - "Media Search & Results View"
Cohesion: 0.50
Nodes (5): Search Results Media Grid, Search Results Header (Hasil Pencarian 'Dune'), Reset Search Action Link (Reset Pencarian), Search Bar with Query Input, Search Results View

### Community 19 - "Series Media Modal & Episodes"
Cohesion: 0.50
Nodes (5): Episode Download List, Media Modal Header (Dune: Prophecy), Season Navigation Tabs (Season 1), Subtitle Selector Dropdown (Indonesian), Media Detail Modal (TV Series)

### Community 20 - "Brand Vector Logo & Favicon"
Cohesion: 0.50
Nodes (5): Brand Geometric Symbol, Brand Color Palette, SVG Filtered Lighting and Glow Effects, Frontend Favicon SVG, Vector Application Favicon

### Community 21 - "Hero Isometric Graphic Artwork"
Cohesion: 1.00
Nodes (4): Base Layer (Purple Neon Illuminated Slab), Hero Banner 3D Isometric Layered Graphic, Dashed Projection and Elevation Guide Lines, Top Floating Layer (Isometric Rounded Plate)

### Community 22 - "Backend Main Entrypoint Lifecycle"
Cohesion: 0.50
Nodes (4): main(), Box, Error, Result

## Knowledge Gaps
- **94 isolated node(s):** `idlixdownloader`, `name`, `private`, `version`, `type` (+89 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 145 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `IdlixClient` connect `Async Download Engine & Pipeline` to `Axum REST API Routes`, `IDLIX Stream & Metadata Extractor`, `HTTP Scraper & Curl Client`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **Why does `DownloadManager` connect `Async Download Engine & Pipeline` to `Axum REST API Routes`?**
  _High betweenness centrality (0.045) - this node is a cross-community bridge._
- **Why does `CurlClient` connect `HTTP Scraper & Curl Client` to `IDLIX Stream & Metadata Extractor`, `Async Download Engine & Pipeline`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **What connects `idlixdownloader`, `name`, `private` to the rest of the system?**
  _94 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Svelte Frontend Application State` be split into smaller, more focused modules?**
  _Cohesion score 0.0707070707070707 - nodes in this community are weakly interconnected._
- **Should `Axum REST API Routes` be split into smaller, more focused modules?**
  _Cohesion score 0.1353658536585366 - nodes in this community are weakly interconnected._
- **Should `Async Download Engine & Pipeline` be split into smaller, more focused modules?**
  _Cohesion score 0.13813813813813813 - nodes in this community are weakly interconnected._