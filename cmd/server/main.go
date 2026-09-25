package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iamnotpirates/idlixdownloader/internal/binmanager"
	"github.com/iamnotpirates/idlixdownloader/internal/bot"
	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/db"
	"github.com/iamnotpirates/idlixdownloader/internal/downloader"
	"github.com/iamnotpirates/idlixdownloader/internal/extractor"
	"github.com/iamnotpirates/idlixdownloader/internal/httpclient"
	"github.com/iamnotpirates/idlixdownloader/internal/scraper"
	"github.com/iamnotpirates/idlixdownloader/internal/server"
	"github.com/iamnotpirates/idlixdownloader/internal/sse"
)

func main() {
	fmt.Println("🚀 Starting IDLIX Downloader (Go Engine)...")

	// 1. Config
	cfg := config.LoadConfig()
	fmt.Printf("📁 AppData Directory: %s\n", config.GetAppDataDir())
	fmt.Printf("🎬 Movies Directory: %s\n", cfg.MoviesDir)
	fmt.Printf("📺 Series Directory: %s\n", cfg.SeriesDir)

	// 2. Database
	database, err := db.Open()
	if err != nil {
		log.Fatalf("❌ Database init failed: %v", err)
	}
	defer database.Close()
	fmt.Println("✅ SQLite Database initialized")

	// 3. Binaries Manager
	binMgr := binmanager.New()
	binPaths, err := binMgr.EnsureBinaries()
	if err != nil {
		log.Printf("⚠️ Warning ensuring binaries: %v", err)
	} else {
		fmt.Printf("✅ N_m3u8DL-RE: %s\n", binPaths.NM3U8DLRE)
		fmt.Printf("✅ FFmpeg: %s\n", binPaths.FFmpeg)
	}

	// 4. Clients
	curlCli := httpclient.New()
	sc := scraper.New(cfg.BaseURL, curlCli)
	ext := extractor.New(cfg.BaseURL, curlCli)
	hub := sse.New()

	// 5. Downloader Manager
	dlMgr := downloader.New(database, ext, binPaths, hub)
	dlMgr.Start()
	fmt.Println("✅ Download Worker Pool started")

	// 6. Discord Bot Interface
	discordBot := bot.New(database, sc, ext, dlMgr)
	if discordBot != nil {
		if err := discordBot.Start(); err != nil {
			log.Printf("⚠️ Discord bot failed to start: %v\n", err)
		} else {
			defer discordBot.Stop()
			fmt.Println("🤖 Discord Bot UI initialized")
		}
	}

	// 7. HTTP Server (Optional / Fallback)
	srv := server.New(database, sc, ext, dlMgr, hub)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8989"
	}

	httpSrv := &http.Server{
		Addr:    ":" + port,
		Handler: srv.Router(),
	}

	go func() {
		fmt.Printf("🌐 Server running on http://localhost:%s\n", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️ HTTP Server notice: %v (Discord Bot UI remains fully active)\n", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	fmt.Println("👋 Server exited cleanly")
}
