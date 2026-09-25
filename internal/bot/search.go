package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

func (b *Bot) createExploreSession(items []models.MediaItem, query, category string) string {
	b.exploreMu.Lock()
	defer b.exploreMu.Unlock()

	// Clean up stale sessions (> 1 hour)
	now := time.Now()
	for k, v := range b.exploreSessions {
		if now.Sub(v.CreatedAt) > 1*time.Hour {
			delete(b.exploreSessions, k)
		}
	}

	sessionID := uuid.New().String()[:8]
	b.exploreSessions[sessionID] = &ExploreSession{
		Items:        items,
		CurrentIndex: 0,
		Query:        query,
		Category:     category,
		CreatedAt:    now,
	}
	return sessionID
}

func (b *Bot) getExploreSession(sessionID string) *ExploreSession {
	b.exploreMu.RLock()
	defer b.exploreMu.RUnlock()
	return b.exploreSessions[sessionID]
}

func (b *Bot) buildExploreEmbed(sessionID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	session := b.getExploreSession(sessionID)
	if session == nil || len(session.Items) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🔍 Tidak Ada Hasil",
			Description: "Tidak ditemukan film atau series dengan kata kunci tersebut.",
			Color:       0xED4245,
		}
		return embed, nil
	}

	idx := session.CurrentIndex
	if idx < 0 {
		idx = 0
		session.CurrentIndex = 0
	}
	if idx >= len(session.Items) {
		idx = len(session.Items) - 1
		session.CurrentIndex = idx
	}

	item := session.Items[idx]
	yearStr := "N/A"
	if item.Year != nil && *item.Year != "" {
		yearStr = *item.Year
	}

	ratingStr := item.Rating
	if ratingStr == "" {
		ratingStr = "N/A"
	}

	typeIcon := "🎬"
	if strings.EqualFold(item.MediaType, "TV Series") || strings.EqualFold(item.MediaType, "series") {
		typeIcon = "📺"
	}

	titleText := fmt.Sprintf("%s %s (%s)", typeIcon, item.Title, yearStr)
	categoryLabel := session.Category
	if categoryLabel == "" {
		categoryLabel = "Search Result"
	}

	embed := &discordgo.MessageEmbed{
		Title:       titleText,
		URL:         item.URL,
		Description: fmt.Sprintf("**Kategori**: %s\n**Rating**: ⭐ `%s` | **Tipe**: `%s`", categoryLabel, ratingStr, item.MediaType),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📌 Info",
				Value:  fmt.Sprintf("Item **%d** dari **%d** hasil", idx+1, len(session.Items)),
				Inline: true,
			},
			{
				Name:   "🔗 Link",
				Value:  fmt.Sprintf("[Buka IDLIX Web](%s)", item.URL),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("IDLIX • Item %d/%d • ID: %s", idx+1, len(session.Items), item.Slug),
		},
	}

	if item.Poster != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: item.Poster,
		}
		embed.Image = &discordgo.MessageEmbedImage{
			URL: item.Poster,
		}
	}

	// Build Action Components
	var compRows []discordgo.MessageComponent

	// Row 1: Item Actions (Download / View Episodes)
	var actionRow1 []discordgo.MessageComponent
	if strings.EqualFold(item.MediaType, "TV Series") || strings.EqualFold(item.MediaType, "series") {
		actionRow1 = append(actionRow1, discordgo.Button{
			Label:    "📺 Pilih Season & Episode",
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("btn_series_view_%s_%s", sessionID, item.Slug),
			Emoji:    &discordgo.ComponentEmoji{Name: "📺"},
		})
	} else {
		actionRow1 = append(actionRow1, discordgo.Button{
			Label:    "⬇️ Unduh Film (1080p)",
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("btn_movie_dl_%s_%s", sessionID, item.Slug),
			Emoji:    &discordgo.ComponentEmoji{Name: "⬇️"},
		})
	}

	actionRow1 = append(actionRow1, discordgo.Button{
		Label:    "❌ Tutup",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("btn_close_session_%s", sessionID),
	})
	compRows = append(compRows, discordgo.ActionsRow{Components: actionRow1})

	// Row 2: Navigation Carousel Buttons
	if len(session.Items) > 1 {
		compRows = append(compRows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "◀️ Prev",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("btn_nav_prev_%s", sessionID),
					Disabled: idx == 0,
				},
				discordgo.Button{
					Label:    fmt.Sprintf("%d / %d", idx+1, len(session.Items)),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("btn_nav_pos_%s", sessionID),
					Disabled: true,
				},
				discordgo.Button{
					Label:    "Next ▶️",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("btn_nav_next_%s", sessionID),
					Disabled: idx >= len(session.Items)-1,
				},
			},
		})
	}

	return embed, compRows
}

func (b *Bot) buildGenreSelectComponent(sessionID string) []discordgo.MessageComponent {
	options := []discordgo.SelectMenuOption{
		{Label: "Action", Value: "action", Description: "Film & Series laga / aksi", Emoji: &discordgo.ComponentEmoji{Name: "💥"}},
		{Label: "Anime / Animation", Value: "anime", Description: "Anime jepang & animasi", Emoji: &discordgo.ComponentEmoji{Name: "🌸"}},
		{Label: "Drama / Korean", Value: "drama", Description: "Drakor & film drama", Emoji: &discordgo.ComponentEmoji{Name: "🎭"}},
		{Label: "Sci-Fi & Fantasy", Value: "sci-fi", Description: "Fiksi ilmiah & luar angkasa", Emoji: &discordgo.ComponentEmoji{Name: "🚀"}},
		{Label: "Horror & Thriller", Value: "horror", Description: "Film horor & mencekam", Emoji: &discordgo.ComponentEmoji{Name: "👻"}},
		{Label: "Comedy", Value: "comedy", Description: "Film & Series komedi", Emoji: &discordgo.ComponentEmoji{Name: "😂"}},
		{Label: "Romance", Value: "romance", Description: "Kisah cinta & romantis", Emoji: &discordgo.ComponentEmoji{Name: "💖"}},
		{Label: "Crime & Mystery", Value: "crime", Description: "Detektif & kriminal", Emoji: &discordgo.ComponentEmoji{Name: "🕵️"}},
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "select_genre_explore",
					Placeholder: "🎭 Pilih genre yang ingin dijelajahi...",
					Options:     options,
				},
			},
		},
	}
}
