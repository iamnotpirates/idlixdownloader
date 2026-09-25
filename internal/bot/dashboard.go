package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/iamnotpirates/idlixdownloader/internal/system"
)

func (b *Bot) refreshDashboard() {
	if b.searchChannelID == "" {
		return
	}

	embed := b.buildDashboardEmbed()
	components := b.buildDashboardComponents()
	embedsSlice := []*discordgo.MessageEmbed{embed}

	b.mu.Lock()
	defer b.mu.Unlock()

	// If we already have a message ID, try to edit it
	if b.dashboardMsgID != "" {
		_, err := b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    b.searchChannelID,
			ID:         b.dashboardMsgID,
			Embeds:     &embedsSlice,
			Components: &components,
		})
		if err == nil {
			return
		}
	}

	// Otherwise, find existing bot message in channel or send a new one
	msgs, err := b.session.ChannelMessages(b.searchChannelID, 10, "", "", "")
	if err == nil {
		for _, m := range msgs {
			if m.Author.ID == b.session.State.User.ID && len(m.Embeds) > 0 && m.Embeds[0].Title == "🎬 IDLIX STREAM & DOWNLOADER HUB" {
				b.dashboardMsgID = m.ID
				_, err = b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
					Channel:    b.searchChannelID,
					ID:         m.ID,
					Embeds:     &embedsSlice,
					Components: &components,
				})
				if err == nil {
					return
				}
			}
		}
	}

	// Send new dashboard message
	msg, err := b.session.ChannelMessageSendComplex(b.searchChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("⚠️ Failed to send dashboard message: %v\n", err)
		return
	}
	b.dashboardMsgID = msg.ID
}

func (b *Bot) buildDashboardEmbed() *discordgo.MessageEmbed {
	storage, err := system.GetDiskSpace(b.cfg.MoviesDir)
	storageStr := "N/A"
	if err == nil && storage != nil {
		storageStr = fmt.Sprintf("%.1f GB Free / %.1f GB Total (%.0f%%)", storage.FreeGB, storage.TotalGB, storage.PercentUsed)
	}

	tasks, _ := b.db.GetTasks()
	activeCount := 0
	completedCount := 0
	for _, t := range tasks {
		if t.Status == "downloading" || t.Status == "extracting" || t.Status == "queued" {
			activeCount++
		} else if t.Status == "completed" {
			completedCount++
		}
	}

	return &discordgo.MessageEmbed{
		Title:       "🎬 IDLIX STREAM & DOWNLOADER HUB",
		Description: "Pusat pencarian, streaming discovery, dan antrean download film / series otomatis ke home storage server.",
		Color:       0x5865F2, // Discord Blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⚡ Active Tasks",
				Value:  fmt.Sprintf("`%d active` / `%d total`", activeCount, len(tasks)),
				Inline: true,
			},
			{
				Name:   "✅ Completed",
				Value:  fmt.Sprintf("`%d items`", completedCount),
				Inline: true,
			},
			{
				Name:   "💾 Storage Disk",
				Value:  fmt.Sprintf("`%s`", storageStr),
				Inline: false,
			},
			{
				Name:   "💡 Quick Instructions",
				Value:  "• Klik **🔍 Cari Film / Series** untuk mencari judul spesifik.\n• Klik **🔥 Featured / Trending** atau **🎭 Explore** untuk melihat rekomendasi film/series terbaru.\n• Download akan otomatis membuat live thread di <#" + b.downloadsChannelID + ">.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("IDLIX Downloader • Updated %s", time.Now().Format("15:04:05 WIB")),
		},
	}
}

func (b *Bot) buildDashboardComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "🔍 Cari Judul",
					Style:    discordgo.PrimaryButton,
					CustomID: "btn_search_modal",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔍",
					},
				},
				discordgo.Button{
					Label:    "🔥 Featured & Trending",
					Style:    discordgo.SuccessButton,
					CustomID: "btn_explore_featured",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔥",
					},
				},
				discordgo.Button{
					Label:    "🎬 Latest Movies",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_explore_movies",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🎬",
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "📺 Latest Series",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_explore_series",
					Emoji: &discordgo.ComponentEmoji{
						Name: "📺",
					},
				},
				discordgo.Button{
					Label:    "🎭 Explore Genre",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_explore_genres",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🎭",
					},
				},
				discordgo.Button{
					Label:    "⚙️ Settings",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_open_settings",
					Emoji: &discordgo.ComponentEmoji{
						Name: "⚙️",
					},
				},
				discordgo.Button{
					Label:    "🔄 Refresh",
					Style:    discordgo.SecondaryButton,
					CustomID: "btn_refresh_dash",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔄",
					},
				},
			},
		},
	}
}

func (b *Bot) buildStorageEmbed() *discordgo.MessageEmbed {
	storage, err := system.GetDiskSpace(b.cfg.MoviesDir)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       "💾 Status Storage",
			Description: fmt.Sprintf("Gagal membaca status storage: %v", err),
			Color:       0xED4245,
		}
	}

	return &discordgo.MessageEmbed{
		Title:       "💾 Status Storage Home Server",
		Color:       0x57F287,
		Description: fmt.Sprintf("Path Media: `%s`", storage.Path),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Kapasitas Total",
				Value:  fmt.Sprintf("**%.2f GB** (%s)", storage.TotalGB, humanize.Bytes(storage.TotalBytes)),
				Inline: true,
			},
			{
				Name:   "Tersedia (Free)",
				Value:  fmt.Sprintf("**%.2f GB** (%s)", storage.FreeGB, humanize.Bytes(storage.FreeBytes)),
				Inline: true,
			},
			{
				Name:   "Terpakai (Used)",
				Value:  fmt.Sprintf("**%.2f GB** (%.1f%%)", storage.UsedGB, storage.PercentUsed),
				Inline: true,
			},
			{
				Name:   "Folder Movies",
				Value:  fmt.Sprintf("`%s`", b.cfg.MoviesDir),
				Inline: false,
			},
			{
				Name:   "Folder TV Series",
				Value:  fmt.Sprintf("`%s`", b.cfg.SeriesDir),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Checked at %s", time.Now().Format("15:04:05")),
		},
	}
}
