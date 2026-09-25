package bot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

func (b *Bot) registerSlashCommands() {
	appID := b.session.State.User.ID

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "search",
			Description: "Cari film atau TV series di IDLIX",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "judul",
					Description: "Judul film atau series yang dicari",
					Required:    true,
				},
			},
		},
		{
			Name:        "explore",
			Description: "Jelajahi film & series trending / kategori populer",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "kategori",
					Description: "Pilih kategori explore",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "🔥 Featured / Trending", Value: "featured"},
						{Name: "🎬 Latest Movies", Value: "movies"},
						{Name: "📺 Latest Series", Value: "series"},
						{Name: "🌸 Anime", Value: "anime"},
						{Name: "🎭 Korean Drama", Value: "drama"},
					},
				},
			},
		},
		{
			Name:        "storage",
			Description: "Cek sisa kapasitas penyimpanan di home server",
		},
		{
			Name:        "downloads",
			Description: "Lihat ringkasan antrean unduhan saat ini",
		},
	}

	_, err := b.session.ApplicationCommandBulkOverwrite(appID, b.guildID, commands)
	if err != nil {
		log.Printf("⚠️ Failed to register slash commands: %v\n", err)
	} else {
		log.Println("✅ Slash commands registered successfully!")
	}
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleSlashCommand(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModalSubmit(s, i)
	}
}

func (b *Bot) handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name

	switch name {
	case "search":
		query := i.ApplicationCommandData().Options[0].StringValue()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeSearch(s, i, query, "Search Result")

	case "explore":
		category := "featured"
		if len(i.ApplicationCommandData().Options) > 0 {
			category = i.ApplicationCommandData().Options[0].StringValue()
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeExplore(s, i, category)

	case "storage":
		embed := b.buildStorageEmbed()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})

	case "downloads":
		tasks, _ := b.db.GetTasks()
		desc := "Daftar tugas unduhan:"
		if len(tasks) == 0 {
			desc = "Belum ada unduhan yang berjalan."
		}
		embed := &discordgo.MessageEmbed{
			Title:       "📥 Antrean Unduhan IDLIX",
			Description: desc,
			Color:       0x5865F2,
		}
		for idx, t := range tasks {
			if idx >= 10 {
				break
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("[%s] %s", t.Status, t.Title),
				Value:  fmt.Sprintf("Progress: %.1f%% | File: `%s`", t.Progress, t.FileName),
				Inline: false,
			})
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	}
}

func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// 1. Dashboard Modal trigger
	if customID == "btn_search_modal" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_search_input",
				Title:    "🔍 Cari Film / Series di IDLIX",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "input_query",
								Label:       "Judul Film atau Series",
								Style:       discordgo.TextInputShort,
								Placeholder: "Contoh: Inception, Breaking Bad, Solo Leveling",
								Required:    true,
								MinLength:   2,
								MaxLength:   100,
							},
						},
					},
				},
			},
		})
		return
	}

	// 2. Dashboard Explore Buttons
	if customID == "btn_explore_featured" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeExplore(s, i, "featured")
		return
	}
	if customID == "btn_explore_movies" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeExplore(s, i, "movies")
		return
	}
	if customID == "btn_explore_series" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeExplore(s, i, "series")
		return
	}
	if customID == "btn_explore_genres" {
		comps := b.buildGenreSelectComponent("genre_nav")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "🎭 Silakan pilih genre film/series:",
				Components: comps,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if customID == "btn_refresh_dash" {
		b.refreshDashboard()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ Dashboard berhasil diperbarui!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 3. Genre Select Menu
	if customID == "select_genre_explore" {
		selected := i.MessageComponentData().Values[0]
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeSearch(s, i, selected, fmt.Sprintf("Genre: %s", strings.ToUpper(selected)))
		return
	}

	// 4. Cancel Task Button
	if strings.HasPrefix(customID, "btn_cancel_task_") {
		taskID := strings.TrimPrefix(customID, "btn_cancel_task_")
		_ = b.downloader.Cancel(taskID)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		return
	}

	// Settings button
	if customID == "btn_open_settings" {
		b.showSettingsCard(s, i, true)
		return
	}

	// Change Concurrency Setting
	if strings.HasPrefix(customID, "set_concurrency_") {
		valStr := strings.TrimPrefix(customID, "set_concurrency_")
		val, _ := strconv.Atoi(valStr)
		if val > 0 {
			_ = b.db.SetSetting("max_concurrent_tasks", valStr)
			b.cfg.MaxConcurrentTasks = val
			_ = config.SaveConfig(b.cfg)
			b.showSettingsCard(s, i, false)
		}
		return
	}

	// Change Subtitle Language Setting
	if strings.HasPrefix(customID, "set_sublang_") {
		lang := strings.TrimPrefix(customID, "set_sublang_")
		if lang != "" {
			_ = b.db.SetSetting("sub_lang", lang)
			b.cfg.SubLang = lang
			_ = config.SaveConfig(b.cfg)
			b.showSettingsCard(s, i, false)
		}
		return
	}

	// Change Mirror BaseURL Setting
	if strings.HasPrefix(customID, "set_mirror_") {
		mirrorCode := strings.TrimPrefix(customID, "set_mirror_")
		newBaseURL := "https://z2.idlixku.com"
		if mirrorCode == "asia" {
			newBaseURL = "https://idlix.asia"
		}
		_ = b.db.SetSetting("base_url", newBaseURL)
		b.cfg.BaseURL = newBaseURL
		if b.scraper != nil {
			b.scraper.BaseURL = newBaseURL
		}
		if b.extractor != nil {
			b.extractor.BaseURL = newBaseURL
		}
		_ = config.SaveConfig(b.cfg)
		b.showSettingsCard(s, i, false)
		return
	}

	// Open Change Directories Modal
	if customID == "btn_open_change_dirs_modal" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_change_directories",
				Title:    "📁 Ubah Folder Penyimpanan",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "input_movies_dir",
								Label:       "Folder Film (Movies)",
								Style:       discordgo.TextInputShort,
								Placeholder: "Contoh: Z:\\Film\\Movies atau D:\\Movies",
								Value:       b.cfg.MoviesDir,
								Required:    true,
								MaxLength:   255,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "input_series_dir",
								Label:       "Folder Series (TV Shows)",
								Style:       discordgo.TextInputShort,
								Placeholder: "Contoh: Z:\\Film\\Series atau D:\\Series",
								Value:       b.cfg.SeriesDir,
								Required:    true,
								MaxLength:   255,
							},
						},
					},
				},
			},
		})
		return
	}

	// Close Settings popup
	if customID == "btn_close_session_settings" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "✅ Pengaturan ditutup. Pesan dashboard tetap aktif di channel ini.",
				Embeds:     []*discordgo.MessageEmbed{},
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// 5. Close Session Button (Direct Delete - Zero Leftover Text)
	if strings.HasPrefix(customID, "btn_close_session_") {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		if i.Message != nil {
			_ = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		}
		b.mu.Lock()
		if i.Message != nil && i.Message.ID == b.lastSearchMsgID {
			b.lastSearchMsgID = ""
		}
		b.mu.Unlock()
		return
	}

	// Back to Dashboard from Settings
	if customID == "btn_back_to_dashboard" {
		embed := b.buildDashboardEmbed()
		comps := b.buildDashboardComponents()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: comps,
			},
		})
		return
	}

	// 6. Select Item from List Dropdown
	if strings.HasPrefix(customID, "select_item_from_list_") {
		sessionID := strings.TrimPrefix(customID, "select_item_from_list_")
		selectedIdxStr := i.MessageComponentData().Values[0]
		selectedIdx, _ := strconv.Atoi(selectedIdxStr)

		sess := b.getExploreSession(sessionID)
		if sess != nil {
			sess.CurrentIndex = selectedIdx
			sess.ViewMode = "detail"
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}

	// 7. Back to List Button
	if strings.HasPrefix(customID, "btn_back_to_list_") {
		sessionID := strings.TrimPrefix(customID, "btn_back_to_list_")
		sess := b.getExploreSession(sessionID)
		if sess != nil {
			sess.ViewMode = "list"
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}

	// 8. List Page Navigation Buttons (Prev / Next)
	if strings.HasPrefix(customID, "btn_page_prev_") {
		sessionID := strings.TrimPrefix(customID, "btn_page_prev_")
		sess := b.getExploreSession(sessionID)
		if sess != nil && sess.Page > 0 {
			sess.Page--
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}
	if strings.HasPrefix(customID, "btn_page_next_") {
		sessionID := strings.TrimPrefix(customID, "btn_page_next_")
		sess := b.getExploreSession(sessionID)
		if sess != nil {
			sess.Page++
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}

	// 9. Detail Carousel Navigation Buttons (Prev / Next)
	if strings.HasPrefix(customID, "btn_detail_prev_") {
		sessionID := strings.TrimPrefix(customID, "btn_detail_prev_")
		sess := b.getExploreSession(sessionID)
		if sess != nil && sess.CurrentIndex > 0 {
			sess.CurrentIndex--
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}
	if strings.HasPrefix(customID, "btn_detail_next_") {
		sessionID := strings.TrimPrefix(customID, "btn_detail_next_")
		sess := b.getExploreSession(sessionID)
		if sess != nil && sess.CurrentIndex < len(sess.Items)-1 {
			sess.CurrentIndex++
			embed, comps := b.buildExploreView(sessionID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: comps,
				},
			})
		}
		return
	}

	// 10. Movie Download Button
	if strings.HasPrefix(customID, "btn_movie_dl_") {
		parts := strings.Split(strings.TrimPrefix(customID, "btn_movie_dl_"), "_")
		sessionID := parts[0]
		slug := strings.Join(parts[1:], "_")

		b.cleanSearchMessage(s, i)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		go b.startMovieDownload(s, i, sessionID, slug)
		return
	}

	// 11. Series View / Episode Selector Button
	if strings.HasPrefix(customID, "btn_series_view_") {
		parts := strings.Split(strings.TrimPrefix(customID, "btn_series_view_"), "_")
		sessionID := parts[0]
		slug := strings.Join(parts[1:], "_")

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.showSeriesSelector(s, i, sessionID, slug)
		return
	}

	// 12. Series Season Selected Dropdown
	if strings.HasPrefix(customID, "select_season_") {
		parts := strings.Split(strings.TrimPrefix(customID, "select_season_"), "_")
		sessionID := parts[0]
		slug := strings.Join(parts[1:], "_")
		seasonStr := i.MessageComponentData().Values[0]
		seasonNum, _ := strconv.Atoi(seasonStr)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.showEpisodeSelector(s, i, sessionID, slug, seasonNum)
		return
	}

	// 13. Single Episode Download Selected
	if strings.HasPrefix(customID, "select_ep_dl_") {
		parts := strings.Split(strings.TrimPrefix(customID, "select_ep_dl_"), "_")
		sessionID := parts[0]
		seasonNum, _ := strconv.Atoi(parts[1])
		slug := strings.Join(parts[2:], "_")
		epNumStr := i.MessageComponentData().Values[0]
		epNum, _ := strconv.Atoi(epNumStr)

		b.cleanSearchMessage(s, i)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		go b.startEpisodeDownload(s, i, sessionID, slug, seasonNum, epNum)
		return
	}

	// 14. Download All Season Episodes Button
	if strings.HasPrefix(customID, "btn_dl_all_season_") {
		parts := strings.Split(strings.TrimPrefix(customID, "btn_dl_all_season_"), "_")
		sessionID := parts[0]
		seasonNum, _ := strconv.Atoi(parts[1])
		slug := strings.Join(parts[2:], "_")

		b.cleanSearchMessage(s, i)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		go b.startSeasonDownload(s, i, sessionID, slug, seasonNum)
		return
	}
}

func (b *Bot) cleanSearchMessage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Message != nil {
		_ = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
	}
	b.mu.Lock()
	if i.Message != nil && i.Message.ID == b.lastSearchMsgID {
		b.lastSearchMsgID = ""
	}
	b.mu.Unlock()
}

func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	if customID == "modal_search_input" {
		query := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		go b.executeSearch(s, i, query, "Search Result")
		return
	}

	if customID == "modal_change_directories" {
		moviesDir := strings.TrimSpace(i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
		seriesDir := strings.TrimSpace(i.ModalSubmitData().Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

		if moviesDir != "" {
			_ = os.MkdirAll(moviesDir, 0755)
			b.cfg.MoviesDir = moviesDir
			_ = b.db.SetSetting("movies_dir", moviesDir)
		}
		if seriesDir != "" {
			_ = os.MkdirAll(seriesDir, 0755)
			b.cfg.SeriesDir = seriesDir
			_ = b.db.SetSetting("series_dir", seriesDir)
		}
		_ = config.SaveConfig(b.cfg)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Folder Penyimpanan Berhasil Diperbarui!**\n\n📁 **Movies**: `%s`\n📺 **Series**: `%s`\n\nTersimpan persisten di SQLite & config.json.", b.cfg.MoviesDir, b.cfg.SeriesDir),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
}

func (b *Bot) postSearchResponse(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	// Auto clean-up previous search message to keep channel pristine
	b.mu.Lock()
	if b.lastSearchMsgID != "" {
		_ = s.ChannelMessageDelete(i.ChannelID, b.lastSearchMsgID)
		b.lastSearchMsgID = ""
	}
	b.mu.Unlock()

	msg, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
	if err == nil && msg != nil {
		b.mu.Lock()
		b.lastSearchMsgID = msg.ID
		b.mu.Unlock()
	}
}

func (b *Bot) executeSearch(s *discordgo.Session, i *discordgo.InteractionCreate, query, category string) {
	items, err := b.scraper.SearchContent(query)
	if err != nil || len(items) == 0 {
		msg := fmt.Sprintf("❌ Tidak ditemukan hasil untuk pencarian: **%s**", query)
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: msg,
		})
		return
	}

	sessionID := b.createExploreSession(items, query, category)
	embed, comps := b.buildExploreView(sessionID)
	b.postSearchResponse(s, i, embed, comps)
}

func (b *Bot) executeExplore(s *discordgo.Session, i *discordgo.InteractionCreate, category string) {
	var items []models.MediaItem
	var err error

	switch category {
	case "movies":
		items, err = b.scraper.SearchContentWithType("2026", "movie")
		if err != nil || len(items) == 0 {
			items, err = b.scraper.SearchContentWithType("2025", "movie")
		}
	case "series":
		items, err = b.scraper.SearchContentWithType("2026", "tv_series")
		if err != nil || len(items) == 0 {
			items, err = b.scraper.SearchContentWithType("2025", "tv_series")
		}
	case "anime":
		items, err = b.scraper.SearchContent("anime")
	case "drama":
		items, err = b.scraper.SearchContent("drama")
	default:
		items, err = b.scraper.FetchFeatured()
	}

	if err != nil || len(items) == 0 {
		if category == "movies" {
			items, _ = b.scraper.SearchContentWithType("2025", "movie")
		} else if category == "series" {
			items, _ = b.scraper.SearchContentWithType("2025", "tv_series")
		} else {
			items, _ = b.scraper.SearchContent("2025")
		}
	}

	catLabel := "🔥 Featured & Trending"
	if category == "movies" {
		catLabel = "🎬 Latest Movies"
	} else if category == "series" {
		catLabel = "📺 Latest TV Series"
	} else if category == "anime" {
		catLabel = "🌸 Anime Collection"
	} else if category == "drama" {
		catLabel = "🎭 Korean Drama"
	}

	sessionID := b.createExploreSession(items, category, catLabel)
	embed, comps := b.buildExploreView(sessionID)
	b.postSearchResponse(s, i, embed, comps)
}

func (b *Bot) startMovieDownload(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID, slug string) {
	details, err := b.extractor.FetchMovieDetails(slug)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal mengambil detail film: %v", err),
		})
		return
	}

	cfg := config.LoadConfig()
	cleanTitle := sanitizeTitle(details.Title)
	yearStr := details.Year
	if yearStr == "" {
		yearStr = "N/A"
	}

	outputDir := filepath.Join(cfg.MoviesDir, fmt.Sprintf("%s (%s)", cleanTitle, yearStr))
	fileName := fmt.Sprintf("%s (%s)", cleanTitle, yearStr)
	pageURL := fmt.Sprintf("%s/movie/%s", b.extractor.BaseURL, slug)
	mediaID := details.ID

	taskID := uuid.New().String()[:8]
	task := &models.DownloadTask{
		ID:        taskID,
		Title:     details.Title,
		MediaType: "movie",
		Year:      &yearStr,
		PageURL:   &pageURL,
		MediaID:   &mediaID,
		OutputDir: outputDir,
		FileName:  fileName,
		Status:    models.StatusQueued,
		Progress:  0.0,
		Speed:     "",
		ETA:       "Queued",
		CreatedAt: time.Now().Unix(),
	}

	if err := b.downloader.Enqueue(task); err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memasukkan ke antrean download: %v", err),
		})
		return
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("🚀 **Unduhan Dimulai!**\nFilm: **%s (%s)**\nThread live progress otomatis dibuat di <#%s>.", details.Title, yearStr, b.downloadsChannelID),
	})
}

func (b *Bot) showSeriesSelector(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID, slug string) {
	details, err := b.extractor.FetchSeriesDetails(slug)
	if err != nil || len(details.Seasons) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memuat data season series: %v", err),
		})
		return
	}

	var options []discordgo.SelectMenuOption
	for _, season := range details.Seasons {
		options = append(options, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("Season %d", season.SeasonNum),
			Value:       strconv.Itoa(season.SeasonNum),
			Description: fmt.Sprintf("Total %d episode", len(season.Episodes)),
			Emoji:       &discordgo.ComponentEmoji{Name: "📺"},
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📺 %s - Pilih Season", details.Title),
		Description: "Silakan pilih nomor Season dari dropdown di bawah:",
		Color:       0x5865F2,
	}

	comps := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("select_season_%s_%s", sessionID, slug),
					Placeholder: "📺 Pilih Season...",
					Options:     options,
				},
			},
		},
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (b *Bot) showEpisodeSelector(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID, slug string, seasonNum int) {
	details, err := b.extractor.FetchSeriesDetails(slug)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memuat season: %v", err),
		})
		return
	}

	var targetSeason *models.SeasonInfo
	for _, season := range details.Seasons {
		if season.SeasonNum == seasonNum {
			targetSeason = &season
			break
		}
	}

	if targetSeason == nil || len(targetSeason.Episodes) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Tidak ada episode ditemukan untuk Season %d.", seasonNum),
		})
		return
	}

	var options []discordgo.SelectMenuOption
	for _, ep := range targetSeason.Episodes {
		if len(options) >= 25 { // Discord max 25 options per select
			break
		}
		label := fmt.Sprintf("Episode %d", ep.EpisodeNum)
		if ep.Title != "" {
			label = fmt.Sprintf("E%02d: %s", ep.EpisodeNum, ep.Title)
			if len(label) > 100 {
				label = label[:97] + "..."
			}
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       strconv.Itoa(ep.EpisodeNum),
			Description: fmt.Sprintf("Download S%02dE%02d", seasonNum, ep.EpisodeNum),
			Emoji:       &discordgo.ComponentEmoji{Name: "▶️"},
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📺 %s - Season %d", details.Title, seasonNum),
		Description: fmt.Sprintf("Pilih episode spesifik atau klik tombol **Unduh Semua Episode** (Total %d eps).", len(targetSeason.Episodes)),
		Color:       0x5865F2,
	}

	comps := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("select_ep_dl_%s_%d_%s", sessionID, seasonNum, slug),
					Placeholder: "▶️ Pilih salah satu episode untuk diunduh...",
					Options:     options,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    fmt.Sprintf("⬇️ Unduh Semua Episode (Season %d)", seasonNum),
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("btn_dl_all_season_%s_%d_%s", sessionID, seasonNum, slug),
					Emoji:    &discordgo.ComponentEmoji{Name: "⬇️"},
				},
			},
		},
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (b *Bot) startEpisodeDownload(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID, slug string, seasonNum, epNum int) {
	details, err := b.extractor.FetchSeriesDetails(slug)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memulai download episode: %v", err),
		})
		return
	}

	var targetEp *models.EpisodeInfo
	for _, season := range details.Seasons {
		if season.SeasonNum == seasonNum {
			for _, ep := range season.Episodes {
				if ep.EpisodeNum == epNum {
					targetEp = &ep
					break
				}
			}
		}
	}

	if targetEp == nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Episode tidak ditemukan.",
		})
		return
	}

	cfg := config.LoadConfig()
	cleanTitle := sanitizeTitle(details.Title)
	yearStr := details.Year
	if yearStr == "" {
		yearStr = "N/A"
	}

	epTitle := sanitizeTitle(targetEp.Title)
	if epTitle == "" {
		epTitle = fmt.Sprintf("Episode %d", epNum)
	}

	outputDir := filepath.Join(cfg.SeriesDir, fmt.Sprintf("%s (%s)", cleanTitle, yearStr), fmt.Sprintf("Season %d", seasonNum))
	fileName := fmt.Sprintf("%s - S%02dE%02d - %s", cleanTitle, seasonNum, epNum, epTitle)
	mediaID := targetEp.MediaID
	if mediaID == "" {
		mediaID = targetEp.Slug
	}
	pageURL := fmt.Sprintf("%s/series/%s", b.extractor.BaseURL, slug)

	taskID := uuid.New().String()[:8]
	task := &models.DownloadTask{
		ID:         taskID,
		Title:      fmt.Sprintf("%s S%02dE%02d", details.Title, seasonNum, epNum),
		MediaType:  "series",
		Year:       &yearStr,
		SeasonNum:  &seasonNum,
		EpisodeNum: &epNum,
		PageURL:    &pageURL,
		MediaID:    &mediaID,
		OutputDir:  outputDir,
		FileName:   fileName,
		Status:     models.StatusQueued,
		Progress:   0.0,
		Speed:      "",
		ETA:        "Queued",
		CreatedAt:  time.Now().Unix(),
	}

	if err := b.downloader.Enqueue(task); err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memasukkan episode ke antrean: %v", err),
		})
		return
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("🚀 **Unduhan Dimulai!**\nSeries: **%s - S%02dE%02d**\nThread live progress otomatis dibuat di <#%s>.", details.Title, seasonNum, epNum, b.downloadsChannelID),
	})
}

func (b *Bot) startSeasonDownload(s *discordgo.Session, i *discordgo.InteractionCreate, sessionID, slug string, seasonNum int) {
	details, err := b.extractor.FetchSeriesDetails(slug)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("❌ Gagal memulai download season: %v", err),
		})
		return
	}

	var targetSeason *models.SeasonInfo
	for _, season := range details.Seasons {
		if season.SeasonNum == seasonNum {
			targetSeason = &season
			break
		}
	}

	if targetSeason == nil || len(targetSeason.Episodes) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Tidak ada episode ditemukan dalam season ini.",
		})
		return
	}

	cfg := config.LoadConfig()
	cleanTitle := sanitizeTitle(details.Title)
	yearStr := details.Year
	if yearStr == "" {
		yearStr = "N/A"
	}
	outputDir := filepath.Join(cfg.SeriesDir, fmt.Sprintf("%s (%s)", cleanTitle, yearStr), fmt.Sprintf("Season %d", seasonNum))
	pageURL := fmt.Sprintf("%s/series/%s", b.extractor.BaseURL, slug)

	count := 0
	for _, ep := range targetSeason.Episodes {
		eNum := ep.EpisodeNum
		epTitle := sanitizeTitle(ep.Title)
		if epTitle == "" {
			epTitle = fmt.Sprintf("Episode %d", eNum)
		}
		fileName := fmt.Sprintf("%s - S%02dE%02d - %s", cleanTitle, seasonNum, eNum, epTitle)
		mediaID := ep.MediaID
		if mediaID == "" {
			mediaID = ep.Slug
		}

		sNum := seasonNum
		taskID := uuid.New().String()[:8]
		task := &models.DownloadTask{
			ID:         taskID,
			Title:      fmt.Sprintf("%s S%02dE%02d", details.Title, seasonNum, eNum),
			MediaType:  "series",
			Year:       &yearStr,
			SeasonNum:  &sNum,
			EpisodeNum: &eNum,
			PageURL:    &pageURL,
			MediaID:    &mediaID,
			OutputDir:  outputDir,
			FileName:   fileName,
			Status:     models.StatusQueued,
			Progress:   0.0,
			Speed:      "",
			ETA:        "Queued",
			CreatedAt:  time.Now().Unix(),
		}

		_ = b.downloader.Enqueue(task)
		count++
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("🚀 **%d Episode Berhasil Masuk Antrean!**\nSeries: **%s (Season %d)**\nSemua task akan otomatis membuat live thread di <#%s>.", count, details.Title, seasonNum, b.downloadsChannelID),
	})
}

func sanitizeTitle(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, ch := range invalid {
		name = strings.ReplaceAll(name, ch, "")
	}
	return strings.TrimSpace(name)
}

func (b *Bot) showSettingsCard(s *discordgo.Session, i *discordgo.InteractionCreate, initialOpen bool) {
	currentMax := b.cfg.MaxConcurrentTasks
	if currentMax <= 0 {
		currentMax = 1
	}

	subLang := b.cfg.SubLang
	if subLang == "" {
		subLang = "Indonesian"
	}

	baseURL := b.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://z2.idlixku.com"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ Pengaturan Downloader & Library",
		Description: "Pengaturan engine, preferensi bahasa subtitle, mirror IDLIX, dan jumlah antrean unduhan.\n*(Panel ini bersifat private/ephemeral dan tidak akan mengubah tampilan dashboard yang di-pin)*",
		Color:       0xFEE75C,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⚡ Max Concurrent Tasks",
				Value:  fmt.Sprintf("Saat ini: **%d task berjalan bersamaan**", currentMax),
				Inline: false,
			},
			{
				Name:   "🇮🇩 Subtitle Language",
				Value:  fmt.Sprintf("Saat ini: **`%s`**", subLang),
				Inline: true,
			},
			{
				Name:   "🌐 IDLIX Active Mirror",
				Value:  fmt.Sprintf("Saat ini: **`%s`**", baseURL),
				Inline: true,
			},
			{
				Name:   "📁 Movies Directory",
				Value:  fmt.Sprintf("`%s`", b.cfg.MoviesDir),
				Inline: false,
			},
			{
				Name:   "📺 Series Directory",
				Value:  fmt.Sprintf("`%s`", b.cfg.SeriesDir),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Tersimpan persisten di SQLite & config.json",
		},
	}

	comps := []discordgo.MessageComponent{
		// Row 1: Concurrency Limits
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "1 Task (Aman)",
					Style:    getBtnStyle(currentMax == 1),
					CustomID: "set_concurrency_1",
					Emoji:    &discordgo.ComponentEmoji{Name: "1️⃣"},
				},
				discordgo.Button{
					Label:    "2 Tasks",
					Style:    getBtnStyle(currentMax == 2),
					CustomID: "set_concurrency_2",
					Emoji:    &discordgo.ComponentEmoji{Name: "2️⃣"},
				},
				discordgo.Button{
					Label:    "3 Tasks",
					Style:    getBtnStyle(currentMax == 3),
					CustomID: "set_concurrency_3",
					Emoji:    &discordgo.ComponentEmoji{Name: "3️⃣"},
				},
				discordgo.Button{
					Label:    "5 Tasks (Max)",
					Style:    getBtnStyle(currentMax == 5),
					CustomID: "set_concurrency_5",
					Emoji:    &discordgo.ComponentEmoji{Name: "🚀"},
				},
			},
		},
		// Row 2: Subtitle Preference
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Indonesian",
					Style:    getBtnStyle(subLang == "Indonesian"),
					CustomID: "set_sublang_Indonesian",
					Emoji:    &discordgo.ComponentEmoji{Name: "🇮🇩"},
				},
				discordgo.Button{
					Label:    "English",
					Style:    getBtnStyle(subLang == "English"),
					CustomID: "set_sublang_English",
					Emoji:    &discordgo.ComponentEmoji{Name: "🇬🇧"},
				},
				discordgo.Button{
					Label:    "Semua Bahasa",
					Style:    getBtnStyle(subLang == "All"),
					CustomID: "set_sublang_All",
					Emoji:    &discordgo.ComponentEmoji{Name: "🌐"},
				},
			},
		},
		// Row 3: Mirror Preference
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Mirror: z2.idlixku.com",
					Style:    getBtnStyle(strings.Contains(baseURL, "idlixku")),
					CustomID: "set_mirror_z2",
					Emoji:    &discordgo.ComponentEmoji{Name: "🔗"},
				},
				discordgo.Button{
					Label:    "Mirror: idlix.asia",
					Style:    getBtnStyle(strings.Contains(baseURL, "idlix.asia")),
					CustomID: "set_mirror_asia",
					Emoji:    &discordgo.ComponentEmoji{Name: "🔗"},
				},
			},
		},
		// Row 4: Directory & Close Actions
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Ubah Folder Penyimpanan",
					Style:    discordgo.PrimaryButton,
					CustomID: "btn_open_change_dirs_modal",
					Emoji:    &discordgo.ComponentEmoji{Name: "📁"},
				},
				discordgo.Button{
					Label:    "Tutup Pengaturan",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_close_session_settings",
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}

	if initialOpen {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: comps,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: comps,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func getBtnStyle(active bool) discordgo.ButtonStyle {
	if active {
		return discordgo.SuccessButton
	}
	return discordgo.SecondaryButton
}
