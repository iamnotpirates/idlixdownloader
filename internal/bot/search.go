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
		Page:         0,
		Query:        query,
		Category:     category,
		ViewMode:     "list",
		CreatedAt:    now,
	}
	return sessionID
}

func (b *Bot) getExploreSession(sessionID string) *ExploreSession {
	b.exploreMu.RLock()
	defer b.exploreMu.RUnlock()
	return b.exploreSessions[sessionID]
}

func (b *Bot) buildExploreView(sessionID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	session := b.getExploreSession(sessionID)
	if session == nil || len(session.Items) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🔍 Tidak Ada Hasil",
			Description: "Tidak ditemukan film atau series dengan kata kunci tersebut.",
			Color:       0xED4245,
		}
		return embed, nil
	}

	if session.ViewMode == "detail" {
		return b.buildDetailView(sessionID)
	}
	return b.buildListView(sessionID)
}

func (b *Bot) buildListView(sessionID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	session := b.getExploreSession(sessionID)
	if session == nil || len(session.Items) == 0 {
		return nil, nil
	}

	pageSize := 25
	totalPages := (len(session.Items) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if session.Page < 0 {
		session.Page = 0
	}
	if session.Page >= totalPages {
		session.Page = totalPages - 1
	}

	startIdx := session.Page * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(session.Items) {
		endIdx = len(session.Items)
	}

	pageItems := session.Items[startIdx:endIdx]

	var sb strings.Builder
	var selectOptions []discordgo.SelectMenuOption

	for i, item := range pageItems {
		globalIdx := startIdx + i
		yearStr := "N/A"
		if item.Year != nil && *item.Year != "" {
			yearStr = *item.Year
		}

		typeIcon := "🎬"
		if strings.EqualFold(item.MediaType, "TV Series") || strings.EqualFold(item.MediaType, "series") {
			typeIcon = "📺"
		}

		ratingStr := item.Rating
		if ratingStr == "" {
			ratingStr = "N/A"
		}

		qualityBadge := item.Quality
		if qualityBadge == "" {
			qualityBadge = "HD"
		}

		sb.WriteString(fmt.Sprintf("`%2d.` %s **%s** (%s) `[%s]` • ⭐ `%s`\n",
			globalIdx+1, typeIcon, item.Title, yearStr, qualityBadge, ratingStr))

		label := fmt.Sprintf("%d. %s (%s)", globalIdx+1, item.Title, yearStr)
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		desc := fmt.Sprintf("%s • %s • Rating: %s", item.MediaType, qualityBadge, ratingStr)
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}

		selectOptions = append(selectOptions, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", globalIdx),
			Description: desc,
			Emoji: &discordgo.ComponentEmoji{
				Name: typeIcon,
			},
		})
	}

	categoryLabel := session.Category
	if categoryLabel == "" {
		categoryLabel = "Hasil Pencarian"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 %s (%d Total Ditemukan)", categoryLabel, len(session.Items)),
		Description: fmt.Sprintf("Menampilkan **%d-%d** dari **%d** hasil:\n\n%s\n👉 *Pilih nomor/judul dari dropdown di bawah untuk melihat poster, sinopsis, & unduh.*", startIdx+1, endIdx, len(session.Items), sb.String()),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Halaman %d dari %d • IDLIX Stream & Downloader", session.Page+1, totalPages),
		},
	}

	if len(pageItems) > 0 && pageItems[0].Poster != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: pageItems[0].Poster,
		}
	}

	var components []discordgo.MessageComponent

	// Dropdown Select Menu (up to 25 items)
	if len(selectOptions) > 0 {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("select_item_from_list_%s", sessionID),
					Placeholder: "👇 Pilih judul untuk buka menu download / episode...",
					Options:     selectOptions,
				},
			},
		})
	}

	// Pagination & Close Buttons
	var navRow []discordgo.MessageComponent
	if totalPages > 1 {
		navRow = append(navRow, discordgo.Button{
			Label:    "◀️ Halaman Sebelumnya",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("btn_page_prev_%s", sessionID),
			Disabled: session.Page == 0,
		})
		navRow = append(navRow, discordgo.Button{
			Label:    fmt.Sprintf("Hal %d / %d", session.Page+1, totalPages),
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("btn_page_info_%s", sessionID),
			Disabled: true,
		})
		navRow = append(navRow, discordgo.Button{
			Label:    "Halaman Berikutnya ▶️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("btn_page_next_%s", sessionID),
			Disabled: session.Page >= totalPages-1,
		})
	}

	navRow = append(navRow, discordgo.Button{
		Label:    "❌ Tutup",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("btn_close_session_%s", sessionID),
	})

	components = append(components, discordgo.ActionsRow{Components: navRow})

	return embed, components
}

func (b *Bot) buildDetailView(sessionID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	session := b.getExploreSession(sessionID)
	if session == nil || len(session.Items) == 0 {
		return nil, nil
	}

	idx := session.CurrentIndex
	if idx < 0 || idx >= len(session.Items) {
		idx = 0
		session.CurrentIndex = 0
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

	qualityBadge := item.Quality
	if qualityBadge == "" {
		qualityBadge = "HD"
	}

	titleText := fmt.Sprintf("%s %s (%s) [%s]", typeIcon, item.Title, yearStr, qualityBadge)

	embed := &discordgo.MessageEmbed{
		Title:       titleText,
		URL:         item.URL,
		Description: fmt.Sprintf("**Rating**: ⭐ `%s` | **Tipe**: `%s` | **Tahun**: `%s` | **Kualitas**: `[%s]`", ratingStr, item.MediaType, yearStr, qualityBadge),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔗 Link Web",
				Value:  fmt.Sprintf("[Buka IDLIX](%s)", item.URL),
				Inline: true,
			},
			{
				Name:   "📌 Urutan",
				Value:  fmt.Sprintf("Item **%d** dari **%d**", idx+1, len(session.Items)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("ID: %s • Pilih aksi unduh di bawah", item.Slug),
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

	var compRows []discordgo.MessageComponent

	// Row 1: Action Buttons (Download / Episodes)
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
			Label:    fmt.Sprintf("⬇️ Unduh Film (%s)", qualityBadge),
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("btn_movie_dl_%s_%s", sessionID, item.Slug),
			Emoji:    &discordgo.ComponentEmoji{Name: "⬇️"},
		})
	}

	compRows = append(compRows, discordgo.ActionsRow{Components: actionRow1})

	// Row 2: Back to List & Navigation
	var navRow []discordgo.MessageComponent
	navRow = append(navRow, discordgo.Button{
		Label:    "🔙 Kembali ke Daftar",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("btn_back_to_list_%s", sessionID),
		Emoji:    &discordgo.ComponentEmoji{Name: "🔙"},
	})

	if len(session.Items) > 1 {
		navRow = append(navRow, discordgo.Button{
			Label:    "◀️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("btn_detail_prev_%s", sessionID),
			Disabled: idx == 0,
		})
		navRow = append(navRow, discordgo.Button{
			Label:    "▶️",
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("btn_detail_next_%s", sessionID),
			Disabled: idx >= len(session.Items)-1,
		})
	}

	navRow = append(navRow, discordgo.Button{
		Label:    "❌",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("btn_close_session_%s", sessionID),
	})

	compRows = append(compRows, discordgo.ActionsRow{Components: navRow})

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
