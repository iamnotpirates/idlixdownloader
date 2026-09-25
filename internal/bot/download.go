package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

func (b *Bot) handleDownloaderEvent(event string, task *models.DownloadTask) {
	if task == nil {
		return
	}

	switch event {
	case "task_created":
		b.onTaskCreated(task)
	case "task_updated":
		b.onTaskUpdated(task)
	}
}

func (b *Bot) onTaskCreated(task *models.DownloadTask) {
	if b.downloadsChannelID == "" {
		return
	}

	b.threadsMu.Lock()
	state, exists := b.activeThreads[task.ID]
	if !exists {
		state = &DownloadThreadState{
			Task: task,
		}
		b.activeThreads[task.ID] = state
	}
	b.threadsMu.Unlock()

	// Create Thread in downloads channel
	threadName := fmt.Sprintf("🧵 [DL] %s", task.FileName)
	if len(threadName) > 90 {
		threadName = threadName[:87] + "..."
	}

	// 1. Post Starter Message
	starterMsg, err := b.session.ChannelMessageSend(b.downloadsChannelID, fmt.Sprintf("🎬 **New Download Task**: `%s`", task.Title))
	if err != nil {
		log.Printf("⚠️ Failed to send starter message in downloads channel: %v\n", err)
		return
	}

	// 2. Create Thread from Starter Message
	th, err := b.session.MessageThreadStartComplex(b.downloadsChannelID, starterMsg.ID, &discordgo.ThreadStart{
		Name:                threadName,
		AutoArchiveDuration: 1440, // 24 hours
		Type:                discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		log.Printf("⚠️ Failed to create thread for task %s: %v\n", task.ID, err)
		return
	}

	b.threadsMu.Lock()
	state.ThreadID = th.ID
	currentTask := state.Task
	if currentTask == nil {
		currentTask = task
	}
	b.threadsMu.Unlock()

	// 3. Post Live Progress Embed inside Thread
	embed := b.buildProgressEmbed(currentTask)
	components := b.buildProgressComponents(currentTask)

	msg, err := b.session.ChannelMessageSendComplex(th.ID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err == nil && msg != nil {
		b.threadsMu.Lock()
		state.MessageID = msg.ID
		state.LastUpdated = time.Now()
		latestTask := state.Task
		b.threadsMu.Unlock()

		// If task advanced while thread was being created, sync immediately
		if latestTask != nil && latestTask.Status != currentTask.Status {
			b.onTaskUpdated(latestTask)
		}
	}
}

func (b *Bot) onTaskUpdated(task *models.DownloadTask) {
	b.threadsMu.Lock()
	state, exists := b.activeThreads[task.ID]
	if !exists {
		b.threadsMu.Unlock()
		b.onTaskCreated(task)
		return
	}
	state.Task = task
	threadID := state.ThreadID
	messageID := state.MessageID
	lastUpdated := state.LastUpdated
	b.threadsMu.Unlock()

	if threadID == "" || messageID == "" {
		// Thread message creation still in progress; onTaskCreated will pick up state.Task
		return
	}

	// If task is completed or failed/cancelled, update immediately
	isFinal := task.Status == models.StatusCompleted || task.Status == models.StatusFailed || task.Status == models.StatusCancelled

	// Throttle in-progress updates to once every 2 seconds to respect Discord rate limits
	if !isFinal && time.Since(lastUpdated) < 2000*time.Millisecond {
		return
	}

	embed := b.buildProgressEmbed(task)
	components := b.buildProgressComponents(task)
	embedsSlice := []*discordgo.MessageEmbed{embed}

	_, err := b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    threadID,
		ID:         messageID,
		Embeds:     &embedsSlice,
		Components: &components,
	})
	if err != nil {
		log.Printf("⚠️ Failed to update progress in thread %s: %v\n", threadID, err)
	}

	b.threadsMu.Lock()
	state.LastUpdated = time.Now()
	b.threadsMu.Unlock()

	// If finished, post to separate completed channel if configured
	if isFinal {
		if task.Status == models.StatusCompleted && b.historyChannelID != "" && b.historyChannelID != b.downloadsChannelID {
			b.postHistoryCard(task)
		}
		b.refreshDashboard()
	}
}

func (b *Bot) buildProgressEmbed(task *models.DownloadTask) *discordgo.MessageEmbed {
	color := 0x5865F2 // Blurple
	statusIcon := "📥"
	statusText := "Mengunduh stream..."

	switch task.Status {
	case models.StatusQueued:
		statusIcon = "⏳"
		statusText = "Dalam Antrean (Queued)"
		color = 0xFEE75C // Yellow
	case models.StatusExtracting:
		statusIcon = "🔍"
		statusText = "Mengekstrak M3U8 & Decrypt Stream..."
		color = 0xEB459E // Pink
	case models.StatusDownloading:
		statusIcon = "⚡"
		statusText = "Mengunduh video stream..."
		color = 0x5865F2 // Blurple
	case models.StatusCompleted:
		statusIcon = "✅"
		statusText = "Download Selesai (100%)"
		color = 0x57F287 // Green
	case models.StatusFailed:
		statusIcon = "❌"
		statusText = "Download Gagal"
		color = 0xED4245 // Red
	case models.StatusCancelled:
		statusIcon = "⏹️"
		statusText = "Dibatalkan oleh Pengguna"
		color = 0x95A5A6 // Grey
	}

	progressBar := generateProgressBar(task.Progress, 16)
	speedStr := task.Speed
	if speedStr == "" {
		speedStr = "-"
	}
	etaStr := task.ETA
	if etaStr == "" {
		etaStr = "-"
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "📊 Progress",
			Value:  fmt.Sprintf("`%s` **%.1f%%**", progressBar, task.Progress),
			Inline: false,
		},
		{
			Name:   "⚡ Kecepatan",
			Value:  fmt.Sprintf("`%s`", speedStr),
			Inline: true,
		},
		{
			Name:   "⏳ Estimasi (ETA)",
			Value:  fmt.Sprintf("`%s`", etaStr),
			Inline: true,
		},
		{
			Name:   "📁 Nama File",
			Value:  fmt.Sprintf("`%s.mp4`", task.FileName),
			Inline: false,
		},
		{
			Name:   "💾 Output Directory",
			Value:  fmt.Sprintf("`%s`", task.OutputDir),
			Inline: false,
		},
	}

	if task.ErrorMsg != nil && *task.ErrorMsg != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Error Reason",
			Value:  fmt.Sprintf("```\n%s\n```", *task.ErrorMsg),
			Inline: false,
		})
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", statusIcon, task.Title),
		Description: fmt.Sprintf("**Status**: %s", statusText),
		Color:       color,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Task ID: %s • %s", task.ID, time.Now().Format("15:04:05 WIB")),
		},
	}
}

func (b *Bot) buildProgressComponents(task *models.DownloadTask) []discordgo.MessageComponent {
	if task.Status == models.StatusCompleted || task.Status == models.StatusFailed || task.Status == models.StatusCancelled {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "⏹️ Batalkan Unduhan",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("btn_cancel_task_%s", task.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "⏹️",
					},
				},
			},
		},
	}
}

func (b *Bot) postHistoryCard(task *models.DownloadTask) {
	if b.historyChannelID == "" {
		return
	}

	typeIcon := "🎬"
	if strings.EqualFold(task.MediaType, "series") {
		typeIcon = "📺"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("✅ %s %s Siap Ditonton!", typeIcon, task.Title),
		Description: fmt.Sprintf("File video berhasil diunduh dan disimpan ke server library."),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📁 File Output",
				Value:  fmt.Sprintf("`%s.mp4`", task.FileName),
				Inline: false,
			},
			{
				Name:   "💾 Lokasi Folder",
				Value:  fmt.Sprintf("`%s`", task.OutputDir),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Completed at %s", time.Now().Format("15:04:05 WIB")),
		},
	}

	_, _ = b.session.ChannelMessageSendEmbed(b.historyChannelID, embed)
}

func generateProgressBar(percent float64, totalBlocks int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int((percent / 100.0) * float64(totalBlocks))
	if filled > totalBlocks {
		filled = totalBlocks
	}

	var sb strings.Builder
	for i := 0; i < filled; i++ {
		sb.WriteString("█")
	}
	for i := filled; i < totalBlocks; i++ {
		sb.WriteString("░")
	}
	return sb.String()
}
