package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/iamnotpirates/idlixdownloader/internal/binmanager"
	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/db"
	"github.com/iamnotpirates/idlixdownloader/internal/extractor"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
	"github.com/iamnotpirates/idlixdownloader/internal/sse"
)

var (
	ansiRegex           = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
	streamProgressRegex = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)%\s*(?:[\d\.]+\s*[kKMmGg]?[bB]/[\d\.]+\s*[kKMmGg]?[bB])?\s*-?([\d\.]+\s*(?:[kKMmGg]?[bB](?:ps|/s|/sec)|[kKMmGg]bps))\s*(\d{2}:\d{2}:\d{2}|--:--:--)?`)
	fallbackPercentRegex = regexp.MustCompile(`(?i)(?:Progress:)?\s*(\d+(?:\.\d+)?)%`)
	fallbackSpeedRegex   = regexp.MustCompile(`(?i)-?([\d\.]+\s*(?:[kKMmGg]?[bB](?:ps|/s|/sec)|[kKMmGg]bps))`)
	fallbackETARegex     = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})`)
)

type Manager struct {
	db         *db.DB
	extractor  *extractor.IdlixClient
	binPaths   *binmanager.BinPaths
	sseHub     *sse.Hub
	queueChan  chan string
	tasks      map[string]*models.DownloadTask
	activeProcs map[string]*exec.Cmd
	mu         sync.RWMutex
}

func New(database *db.DB, ext *extractor.IdlixClient, bins *binmanager.BinPaths, hub *sse.Hub) *Manager {
	m := &Manager{
		db:          database,
		extractor:   ext,
		binPaths:    bins,
		sseHub:      hub,
		queueChan:   make(chan string, 100),
		tasks:       make(map[string]*models.DownloadTask),
		activeProcs: make(map[string]*exec.Cmd),
	}

	// Load existing tasks from DB
	if existing, err := database.GetTasks(); err == nil {
		for _, t := range existing {
			task := t
			m.tasks[task.ID] = &task
		}
	}

	return m
}

func (m *Manager) Start() {
	cfg := config.LoadConfig()
	workers := cfg.MaxConcurrentTasks
	if workers <= 0 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go m.workerLoop(i)
	}

	// Queue any tasks that were left queued or downloading
	m.mu.RLock()
	for _, t := range m.tasks {
		if t.Status == models.StatusQueued || t.Status == models.StatusDownloading || t.Status == models.StatusExtracting {
			t.Status = models.StatusQueued
			_ = m.db.SaveTask(t)
			m.queueChan <- t.ID
		}
	}
	m.mu.RUnlock()
}

func (m *Manager) Enqueue(task *models.DownloadTask) error {
	m.mu.Lock()
	m.tasks[task.ID] = task
	m.mu.Unlock()

	if err := m.db.SaveTask(task); err != nil {
		return err
	}

	m.sseHub.Broadcast("task_created", task)
	m.queueChan <- task.ID
	return nil
}

func (m *Manager) Cancel(taskID string) error {
	m.mu.Lock()
	if cmd, ok := m.activeProcs[taskID]; ok && cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if task, ok := m.tasks[taskID]; ok {
		task.Status = models.StatusCancelled
		_ = m.db.SaveTask(task)
		m.sseHub.Broadcast("task_updated", task)
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) Delete(taskID string) error {
	_ = m.Cancel(taskID)
	m.mu.Lock()
	delete(m.tasks, taskID)
	m.mu.Unlock()

	if err := m.db.DeleteTask(taskID); err != nil {
		return err
	}
	m.sseHub.Broadcast("task_deleted", map[string]string{"id": taskID})
	return nil
}

func (m *Manager) GetAllTasks() []models.DownloadTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]models.DownloadTask, 0, len(m.tasks))
	for _, t := range m.tasks {
		list = append(list, *t)
	}
	return list
}

func (m *Manager) workerLoop(workerID int) {
	for taskID := range m.queueChan {
		m.mu.RLock()
		task, ok := m.tasks[taskID]
		m.mu.RUnlock()

		if !ok || task.Status == models.StatusCancelled {
			continue
		}

		m.executeTask(task)
	}
}

func (m *Manager) executeTask(task *models.DownloadTask) {
	// 0. Check if file already exists in output directory and validate integrity via FFmpeg
	existingFile := filepath.Join(task.OutputDir, task.FileName+".mp4")
	if isValid, sizeMB := validateMediaIntegrity(m.binPaths.FFmpeg, existingFile); isValid {
		m.mu.Lock()
		task.Status = models.StatusCompleted
		task.Progress = 100.0
		task.ETA = "File already exists"
		task.Speed = ""
		_ = m.db.SaveTask(task)
		m.mu.Unlock()
		m.log(task.ID, fmt.Sprintf("Valid media file already exists (%.2f MB), skipping download: %s", sizeMB, existingFile))
		m.sseHub.Broadcast("task_updated", task)
		return
	}

	// 1. Check if extraction is needed
	if task.M3U8URL == "" {
		m.updateStatus(task, models.StatusExtracting, "Extracting video stream...")
		m.log(task.ID, "Starting stream extraction...")

		contentType := task.MediaType
		slugOrID := ""
		if task.MediaID != nil && *task.MediaID != "" {
			slugOrID = *task.MediaID
		} else if task.PageURL != nil {
			slugOrID = *task.PageURL
		}

		pageURL := ""
		if task.PageURL != nil {
			pageURL = *task.PageURL
		}

		sources, err := m.extractor.ExtractStream(contentType, slugOrID, pageURL)
		if err != nil {
			m.failTask(task, fmt.Sprintf("Stream extraction failed: %v", err))
			return
		}

		task.M3U8URL = sources.M3U8URL

		// Choose subtitle (Indonesian default, fallback to first)
		cfg := config.LoadConfig()
		targetLang := strings.ToLower(cfg.SubLang)
		if targetLang == "" {
			targetLang = "indonesian"
		}

		var chosenSub *string
		for _, s := range sources.Subtitles {
			if strings.Contains(strings.ToLower(s.Lang), targetLang) || strings.Contains(strings.ToLower(s.Lang), "indo") {
				subURL := s.URL
				chosenSub = &subURL
				break
			}
		}
		if chosenSub == nil && len(sources.Subtitles) > 0 {
			subURL := sources.Subtitles[0].URL
			chosenSub = &subURL
		}

		task.SubtitleURL = chosenSub
		m.log(task.ID, fmt.Sprintf("Stream URL resolved: %s", task.M3U8URL))
	}

	// 2. Setup Staging Directory
	stagingDir := filepath.Join(os.TempDir(), "idlixdownloader", task.ID)
	chunksDir := filepath.Join(stagingDir, "chunks")
	_ = os.MkdirAll(chunksDir, 0755)
	defer os.RemoveAll(stagingDir)

	m.updateStatus(task, models.StatusDownloading, "Starting download...")
	m.log(task.ID, fmt.Sprintf("Working in staging dir: %s", stagingDir))

	// 3. Download and convert Subtitle if present
	if task.SubtitleURL != nil && *task.SubtitleURL != "" {
		srtPath := filepath.Join(stagingDir, fmt.Sprintf("%s.srt", task.FileName))
		m.log(task.ID, "Downloading and converting subtitle...")
		if err := downloadAndConvertVTTtoSRT(*task.SubtitleURL, srtPath); err != nil {
			m.log(task.ID, fmt.Sprintf("[WARN] Subtitle error: %v", err))
		} else {
			m.log(task.ID, "Subtitle converted to SRT successfully")
		}
	}

	// 4. Run N_m3u8DL-RE with exact flags for full real-time progress stream
	refHeader := fmt.Sprintf("Referer: %s/", strings.TrimRight(m.extractor.BaseURL, "/"))
	args := []string{
		task.M3U8URL,
		"--tmp-dir", chunksDir,
		"--save-dir", stagingDir,
		"--save-name", task.FileName,
		"--auto-select",
		"--force-ansi-console",
		"--thread-count", "16",
		"--download-retry-count", "3",
		"-M", "format=mp4:muxer=ffmpeg",
		"--auto-subtitle-fix",
		"--check-segments-count", "false",
		"-H", refHeader,
		"-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"--log-level", "INFO",
	}

	if m.binPaths.FFmpeg != "" {
		args = append(args, "--ffmpeg-binary-path", m.binPaths.FFmpeg)
	}

	cmd := exec.Command(m.binPaths.NM3U8DLRE, args...)
	cmd.Dir = stagingDir
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		m.failTask(task, fmt.Sprintf("Failed to open stdout pipe: %v", err))
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		m.failTask(task, fmt.Sprintf("Failed to open stderr pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		m.failTask(task, fmt.Sprintf("Failed to start N_m3u8DL-RE: %v", err))
		return
	}

	m.mu.Lock()
	m.activeProcs[task.ID] = cmd
	m.mu.Unlock()

	// Parse stdout & stderr
	go m.scanOutput(task.ID, stdoutPipe)
	go m.scanOutput(task.ID, stderrPipe)

	exitErr := cmd.Wait()

	m.mu.Lock()
	delete(m.activeProcs, task.ID)
	m.mu.Unlock()

	if exitErr != nil {
		m.mu.RLock()
		isCancelled := task.Status == models.StatusCancelled
		m.mu.RUnlock()
		if isCancelled {
			m.log(task.ID, "Task was cancelled")
			return
		}
		m.failTask(task, fmt.Sprintf("N_m3u8DL-RE exited with error: %v", exitErr))
		return
	}

	// 5. Move output files to destination
	_ = os.MkdirAll(task.OutputDir, 0755)

	entries, _ := os.ReadDir(stagingDir)
	foundMP4 := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".mp4" || ext == ".mkv" || ext == ".srt" {
			src := filepath.Join(stagingDir, name)
			dst := filepath.Join(task.OutputDir, name)
			_ = copyFile(src, dst)
			if ext == ".mp4" || ext == ".mkv" {
				foundMP4 = true
			}
		}
	}

	if !foundMP4 {
		m.failTask(task, "Download finished but output MP4 file was not found")
		return
	}

	m.mu.Lock()
	task.Status = models.StatusCompleted
	task.Progress = 100.0
	task.ETA = "Finished"
	task.Speed = ""
	_ = m.db.SaveTask(task)
	m.mu.Unlock()

	m.log(task.ID, fmt.Sprintf("Successfully saved to: %s", task.OutputDir))
	m.sseHub.Broadcast("task_updated", task)
}

func (m *Manager) scanOutput(taskID string, r io.Reader) {
	buf := make([]byte, 4096)
	var rollingBuf strings.Builder
	var lastBroadcast time.Time

	for {
		n, err := r.Read(buf)
		if n > 0 {
			cleanChunk := ansiRegex.ReplaceAllString(string(buf[:n]), "")
			rollingBuf.WriteString(cleanChunk)

			content := rollingBuf.String()
			matches := streamProgressRegex.FindAllStringSubmatch(content, -1)
			if len(matches) > 0 {
				lastMatch := matches[len(matches)-1]
				percentStr := lastMatch[1]
				speedStr := lastMatch[2]
				etaStr := ""
				if len(lastMatch) > 3 {
					etaStr = lastMatch[3]
				}

				m.mu.Lock()
				task, ok := m.tasks[taskID]
				if ok && task.Status == models.StatusDownloading {
					updated := false
					if p, err := strconv.ParseFloat(percentStr, 64); err == nil {
						if p != task.Progress {
							task.Progress = p
							updated = true
						}
					}
					sp := formatSpeedDisplay(speedStr)
					if sp != "" && sp != task.Speed {
						task.Speed = sp
						updated = true
					}
					if etaStr != "" && etaStr != "--:--:--" && etaStr != task.ETA {
						task.ETA = etaStr
						updated = true
					}

					if updated && time.Since(lastBroadcast) > 100*time.Millisecond {
						lastBroadcast = time.Now()
						_ = m.db.SaveTask(task)
						m.sseHub.Broadcast("task_updated", task)
					}
				}
				m.mu.Unlock()
			} else {
				// Fallback if full regex didn't catch, try simple percent
				if pMatches := fallbackPercentRegex.FindAllStringSubmatch(content, -1); len(pMatches) > 0 {
					pLast := pMatches[len(pMatches)-1]
					if p, err := strconv.ParseFloat(pLast[1], 64); err == nil {
						m.mu.Lock()
						task, ok := m.tasks[taskID]
						if ok && task.Status == models.StatusDownloading && p != task.Progress {
							task.Progress = p
							if time.Since(lastBroadcast) > 100*time.Millisecond {
								lastBroadcast = time.Now()
								_ = m.db.SaveTask(task)
								m.sseHub.Broadcast("task_updated", task)
							}
						}
						m.mu.Unlock()
					}
				}
			}

			// Keep only tail in rollingBuf so memory stays tiny
			if rollingBuf.Len() > 300 {
				tail := rollingBuf.String()
				tail = tail[len(tail)-300:]
				rollingBuf.Reset()
				rollingBuf.WriteString(tail)
			}
		}
		if err != nil {
			break
		}
	}
}

func formatSpeedDisplay(raw string) string {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "-"))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "MBps", " MB/s")
	s = strings.ReplaceAll(s, "KBps", " KB/s")
	s = strings.ReplaceAll(s, "GBps", " GB/s")
	s = strings.ReplaceAll(s, "Bps", " B/s")
	return s
}

func (m *Manager) updateStatus(task *models.DownloadTask, status models.DownloadStatus, eta string) {
	m.mu.Lock()
	task.Status = status
	task.ETA = eta
	_ = m.db.SaveTask(task)
	m.mu.Unlock()

	m.sseHub.Broadcast("task_updated", task)
}

func (m *Manager) failTask(task *models.DownloadTask, errMsg string) {
	m.mu.Lock()
	task.Status = models.StatusFailed
	task.ErrorMsg = &errMsg
	task.ETA = "Failed"
	task.Speed = ""
	_ = m.db.SaveTask(task)
	m.mu.Unlock()

	m.log(task.ID, fmt.Sprintf("[ERROR] %s", errMsg))
	m.sseHub.Broadcast("task_updated", task)
}

func (m *Manager) log(taskID, message string) {
	_ = m.db.AddTaskLog(taskID, message)
}

func downloadAndConvertVTTtoSRT(vttURL, srtPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(vttURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	lines := strings.Split(string(bodyBytes), "\n")
	var srtLines []string
	counter := 1
	inCue := false

	vttTimeRegex := regexp.MustCompile(`(\d{2}:\d{2}:\d{2})\.(\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2})\.(\d{3})`)
	vttShortTimeRegex := regexp.MustCompile(`(\d{2}:\d{2})\.(\d{3})\s*-->\s*(\d{2}:\d{2})\.(\d{3})`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "WEBVTT") || strings.HasPrefix(trimmed, "NOTE") {
			continue
		}

		if vttTimeRegex.MatchString(trimmed) {
			srtTimestamp := vttTimeRegex.ReplaceAllString(trimmed, "$1,$2 --> $3,$4")
			srtLines = append(srtLines, strconv.Itoa(counter), srtTimestamp)
			counter++
			inCue = true
			continue
		}

		if vttShortTimeRegex.MatchString(trimmed) {
			fixed := vttShortTimeRegex.ReplaceAllString(trimmed, "00:$1,$2 --> 00:$3,$4")
			srtLines = append(srtLines, strconv.Itoa(counter), fixed)
			counter++
			inCue = true
			continue
		}

		if trimmed == "" {
			if inCue {
				srtLines = append(srtLines, "")
				inCue = false
			}
		} else if inCue {
			srtLines = append(srtLines, trimmed)
		}
	}

	return os.WriteFile(srtPath, []byte(strings.Join(srtLines, "\n")), 0644)
}

func validateMediaIntegrity(ffmpegPath, filePath string) (bool, float64) {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() || info.Size() < 1024*1024 {
		return false, 0
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)

	// If FFmpeg is available, verify stream container and header parsing
	if ffmpegPath != "" {
		cmd := exec.Command(ffmpegPath, "-v", "error", "-i", filePath, "-t", "0.1", "-f", "null", "-")
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
		}
		if err := cmd.Run(); err != nil {
			// File corrupted / broken moov atom
			return false, sizeMB
		}
	}

	return true, sizeMB
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
