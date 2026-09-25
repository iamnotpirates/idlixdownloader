package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/db"
	"github.com/iamnotpirates/idlixdownloader/internal/downloader"
	"github.com/iamnotpirates/idlixdownloader/internal/extractor"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
	"github.com/iamnotpirates/idlixdownloader/internal/scraper"
)

type Bot struct {
	session    *discordgo.Session
	db         *db.DB
	scraper    *scraper.Scraper
	extractor  *extractor.IdlixClient
	downloader *downloader.Manager
	cfg        models.AppConfig

	guildID            string
	categoryID         string
	searchChannelID    string
	downloadsChannelID string
	historyChannelID   string

	dashboardMsgID string
	mu             sync.RWMutex

	// Live download tracking
	activeThreads map[string]*DownloadThreadState // taskID -> state
	threadsMu     sync.RWMutex

	// Search/Explore sessions
	exploreSessions map[string]*ExploreSession // sessionID -> state
	exploreMu       sync.RWMutex
}

type DownloadThreadState struct {
	ThreadID    string
	MessageID   string
	LastUpdated time.Time
	Task        *models.DownloadTask
}

type ExploreSession struct {
	Items        []models.MediaItem
	CurrentIndex int
	Page         int
	Query        string
	Category     string
	ViewMode     string // "list" or "detail"
	CreatedAt    time.Time
}

func New(
	database *db.DB,
	sc *scraper.Scraper,
	ext *extractor.IdlixClient,
	dl *downloader.Manager,
) *Bot {
	cfg := config.LoadConfig()
	token := strings.TrimSpace(cfg.DiscordBotToken)
	if token == "" {
		log.Println("ℹ️ Discord bot token is empty. Discord Bot UI will not start.")
		return nil
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Printf("⚠️ Failed to initialize Discord session: %v\n", err)
		return nil
	}

	b := &Bot{
		session:            dg,
		db:                 database,
		scraper:            sc,
		extractor:          ext,
		downloader:         dl,
		cfg:                cfg,
		guildID:            cfg.DiscordGuildID,
		searchChannelID:    cfg.DiscordSearchChannelID,
		downloadsChannelID: cfg.DiscordDownloadsChannelID,
		historyChannelID:   cfg.DiscordHistoryChannelID,
		activeThreads:      make(map[string]*DownloadThreadState),
		exploreSessions:    make(map[string]*ExploreSession),
	}

	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onInteractionCreate)

	// Hook downloader events for live discord updates
	dl.AddListener(b.handleDownloaderEvent)

	return b
}

func (b *Bot) Start() error {
	if b == nil || b.session == nil {
		return nil
	}

	b.session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("error opening discord connection: %w", err)
	}

	log.Println("🤖 IDLIX Discord Bot connected successfully!")
	return nil
}

func (b *Bot) Stop() {
	if b == nil || b.session == nil {
		return
	}
	_ = b.session.Close()
	log.Println("🛑 IDLIX Discord Bot disconnected.")
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("🤖 Logged in as: %s#%s (ID: %s)\n", s.State.User.Username, s.State.User.Discriminator, s.State.User.ID)

	// Resolve Guild and Channels under Category
	go func() {
		b.ensureChannels()
		b.registerSlashCommands()
		b.refreshDashboard()
	}()
}

func (b *Bot) ensureChannels() {
	// If guild is not set, pick the first guild where bot is present
	guilds := b.session.State.Guilds
	if len(guilds) == 0 {
		return
	}

	var targetGuild *discordgo.Guild
	if b.guildID != "" {
		for _, g := range guilds {
			if g.ID == b.guildID {
				targetGuild = g
				break
			}
		}
	}
	if targetGuild == nil {
		targetGuild = guilds[0]
		b.guildID = targetGuild.ID
	}

	categoryName := b.cfg.DiscordCategoryName
	if categoryName == "" {
		categoryName = "iDLiX Downloader"
	}

	channels, err := b.session.GuildChannels(b.guildID)
	if err != nil {
		log.Printf("⚠️ Failed to get guild channels: %v\n", err)
		return
	}

	// 1. First priority: find parent category from existing idlix-downloader-forum channel
	for _, ch := range channels {
		if strings.Contains(strings.ToLower(ch.Name), "idlix-downloader-forum") || ch.ID == "1553011892342235161" {
			if ch.ParentID != "" {
				b.categoryID = ch.ParentID
				log.Printf("📁 Found existing parent category from forum channel: %s\n", b.categoryID)
				break
			}
		}
	}

	// 2. Fallback: Search category by name
	if b.categoryID == "" {
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildCategory {
				if strings.EqualFold(strings.TrimSpace(ch.Name), strings.TrimSpace(categoryName)) || strings.Contains(strings.ToLower(ch.Name), "idlix") {
					b.categoryID = ch.ID
					break
				}
			}
		}
	}

	// 3. Clean up any accidental duplicate categories
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory && ch.ID != b.categoryID && strings.EqualFold(strings.TrimSpace(ch.Name), strings.TrimSpace(categoryName)) {
			log.Printf("🧹 Cleaning up duplicate category: %s (%s)\n", ch.Name, ch.ID)
			// Delete child channels inside duplicate category first if empty
			for _, subCh := range channels {
				if subCh.ParentID == ch.ID {
					_, _ = b.session.ChannelDelete(subCh.ID)
				}
			}
			_, _ = b.session.ChannelDelete(ch.ID)
		}
	}

	// 4. Map existing child channels under category
	for _, ch := range channels {
		if ch.ParentID == b.categoryID {
			name := strings.ToLower(ch.Name)
			if strings.Contains(name, "search") || strings.Contains(name, "request") {
				b.searchChannelID = ch.ID
			} else if strings.Contains(name, "download") && (strings.Contains(name, "active") || strings.Contains(name, "live") || strings.Contains(name, "task")) {
				b.downloadsChannelID = ch.ID
			} else if strings.Contains(name, "history") || strings.Contains(name, "library") || strings.Contains(name, "selesai") {
				b.historyChannelID = ch.ID
			}
		}
	}

	// 3. Create missing channels under category
	if b.searchChannelID == "" {
		ch, err := b.createChannelUnderCategory("🔍・search-request", discordgo.ChannelTypeGuildText)
		if err == nil {
			b.searchChannelID = ch.ID
		}
	}
	if b.downloadsChannelID == "" {
		ch, err := b.createChannelUnderCategory("📥・active-downloads", discordgo.ChannelTypeGuildText)
		if err == nil {
			b.downloadsChannelID = ch.ID
		}
	}
	if b.historyChannelID == "" {
		ch, err := b.createChannelUnderCategory("📚・download-history", discordgo.ChannelTypeGuildText)
		if err == nil {
			b.historyChannelID = ch.ID
		}
	}

	// Update config in memory
	b.cfg.DiscordGuildID = b.guildID
	b.cfg.DiscordCategoryID = b.categoryID
	b.cfg.DiscordSearchChannelID = b.searchChannelID
	b.cfg.DiscordDownloadsChannelID = b.downloadsChannelID
	b.cfg.DiscordHistoryChannelID = b.historyChannelID
	_ = config.SaveConfig(b.cfg)

	log.Printf("✅ Discord Channels Ready: Search=%s, Downloads=%s, History=%s\n",
		b.searchChannelID, b.downloadsChannelID, b.historyChannelID)
}

func (b *Bot) createChannelUnderCategory(name string, cType discordgo.ChannelType) (*discordgo.Channel, error) {
	data := discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     cType,
		ParentID: b.categoryID,
	}
	ch, err := b.session.GuildChannelCreateComplex(b.guildID, data)
	if err != nil {
		log.Printf("⚠️ Failed to create channel %s: %v\n", name, err)
		return nil, err
	}
	log.Printf("✨ Created channel: %s (%s)\n", name, ch.ID)
	return ch, nil
}
