package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
	mu sync.Mutex
}

func Open() (*DB, error) {
	dbPath := filepath.Join(config.GetAppDataDir(), "idlix.db")
	
	// Ensure parent dir exists
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Enable WAL mode & busy timeout
	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
	`); err != nil {
		return nil, fmt.Errorf("failed to set pragma: %w", err)
	}

	d := &DB{db: db}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return d, nil
}

func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		media_type TEXT NOT NULL,
		year TEXT,
		season_num INTEGER,
		episode_num INTEGER,
		page_url TEXT,
		media_id TEXT,
		m3u8_url TEXT NOT NULL,
		subtitle_url TEXT,
		sub_lang TEXT,
		output_dir TEXT NOT NULL,
		file_name TEXT NOT NULL,
		status TEXT NOT NULL,
		progress REAL NOT NULL DEFAULT 0.0,
		speed TEXT NOT NULL DEFAULT '',
		eta TEXT NOT NULL DEFAULT '',
		error_msg TEXT,
		discord_thread_id TEXT,
		discord_message_id TEXT,
		discord_guild_id TEXT,
		discord_channel_id TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS task_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_task_logs_task_id ON task_logs (task_id);
	`
	if _, err := d.db.Exec(schema); err != nil {
		return err
	}

	return d.ensureTaskColumns()
}

func (d *DB) ensureTaskColumns() error {
	rows, err := d.db.Query("PRAGMA table_info(tasks);")
	if err != nil {
		return err
	}
	defer rows.Close()

	existingCols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
			existingCols[strings.ToLower(name)] = true
		}
	}

	colsToAdd := map[string]string{
		"discord_thread_id":  "TEXT",
		"discord_message_id": "TEXT",
		"discord_guild_id":   "TEXT",
		"discord_channel_id": "TEXT",
		"year":               "TEXT",
		"season_num":         "INTEGER",
		"episode_num":        "INTEGER",
		"page_url":           "TEXT",
		"media_id":           "TEXT",
		"subtitle_url":       "TEXT",
		"sub_lang":           "TEXT",
	}

	for col, colType := range colsToAdd {
		if !existingCols[col] {
			_, _ = d.db.Exec(fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s;", col, colType))
		}
	}
	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) SaveTask(task *models.DownloadTask) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
	INSERT INTO tasks (
		id, title, media_type, year, season_num, episode_num,
		page_url, media_id, m3u8_url, subtitle_url, sub_lang,
		output_dir, file_name, status, progress, speed, eta, error_msg,
		discord_thread_id, discord_message_id, discord_channel_id, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		media_type=excluded.media_type,
		year=excluded.year,
		season_num=excluded.season_num,
		episode_num=excluded.episode_num,
		page_url=excluded.page_url,
		media_id=excluded.media_id,
		m3u8_url=excluded.m3u8_url,
		subtitle_url=excluded.subtitle_url,
		sub_lang=excluded.sub_lang,
		output_dir=excluded.output_dir,
		file_name=excluded.file_name,
		status=excluded.status,
		progress=excluded.progress,
		speed=excluded.speed,
		eta=excluded.eta,
		error_msg=excluded.error_msg,
		discord_thread_id=coalesce(excluded.discord_thread_id, tasks.discord_thread_id),
		discord_message_id=coalesce(excluded.discord_message_id, tasks.discord_message_id),
		discord_channel_id=coalesce(excluded.discord_channel_id, tasks.discord_channel_id);
	`

	_, err := d.db.Exec(query,
		task.ID, task.Title, task.MediaType, task.Year, task.SeasonNum, task.EpisodeNum,
		task.PageURL, task.MediaID, task.M3U8URL, task.SubtitleURL, task.SubLang,
		task.OutputDir, task.FileName, string(task.Status), task.Progress, task.Speed, task.ETA, task.ErrorMsg,
		task.DiscordThreadID, task.DiscordMessageID, task.DiscordChannelID, task.CreatedAt,
	)
	return err
}

func (d *DB) GetTasks() ([]models.DownloadTask, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, title, media_type, year, season_num, episode_num,
		       page_url, media_id, m3u8_url, subtitle_url, sub_lang,
		       output_dir, file_name, status, progress, speed, eta, error_msg,
		       discord_thread_id, discord_message_id, discord_channel_id, created_at
		FROM tasks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.DownloadTask
	for rows.Next() {
		var t models.DownloadTask
		var statusStr string
		err := rows.Scan(
			&t.ID, &t.Title, &t.MediaType, &t.Year, &t.SeasonNum, &t.EpisodeNum,
			&t.PageURL, &t.MediaID, &t.M3U8URL, &t.SubtitleURL, &t.SubLang,
			&t.OutputDir, &t.FileName, &statusStr, &t.Progress, &t.Speed, &t.ETA, &t.ErrorMsg,
			&t.DiscordThreadID, &t.DiscordMessageID, &t.DiscordChannelID, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		t.Status = models.DownloadStatus(statusStr)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (d *DB) GetTask(id string) (*models.DownloadTask, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var t models.DownloadTask
	var statusStr string
	err := d.db.QueryRow(`
		SELECT id, title, media_type, year, season_num, episode_num,
		       page_url, media_id, m3u8_url, subtitle_url, sub_lang,
		       output_dir, file_name, status, progress, speed, eta, error_msg,
		       discord_thread_id, discord_message_id, discord_channel_id, created_at
		FROM tasks WHERE id = ?
	`, id).Scan(
		&t.ID, &t.Title, &t.MediaType, &t.Year, &t.SeasonNum, &t.EpisodeNum,
		&t.PageURL, &t.MediaID, &t.M3U8URL, &t.SubtitleURL, &t.SubLang,
		&t.OutputDir, &t.FileName, &statusStr, &t.Progress, &t.Speed, &t.ETA, &t.ErrorMsg,
		&t.DiscordThreadID, &t.DiscordMessageID, &t.DiscordChannelID, &t.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Status = models.DownloadStatus(statusStr)
	return &t, nil
}

func (d *DB) DeleteTask(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, err := d.db.Exec(`DELETE FROM tasks WHERE id = ?`, id); err != nil {
		return err
	}
	_, err := d.db.Exec(`DELETE FROM task_logs WHERE task_id = ?`, id)
	return err
}

func (d *DB) ClearFinishedTasks() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, _ = d.db.Exec(`
		DELETE FROM task_logs WHERE task_id IN (
			SELECT id FROM tasks WHERE status IN ('completed', 'failed', 'cancelled')
		)
	`)
	_, err := d.db.Exec(`DELETE FROM tasks WHERE status IN ('completed', 'failed', 'cancelled')`)
	return err
}

func (d *DB) AddTaskLog(taskID, message string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	timestamp := now.Format("15:04:05")
	createdAt := now.Unix()

	_, err := d.db.Exec(`
		INSERT INTO task_logs (task_id, timestamp, message, created_at)
		VALUES (?, ?, ?, ?)
	`, taskID, timestamp, message, createdAt)
	return err
}

func (d *DB) SetSetting(key, val string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, val, time.Now().Unix())
	return err
}

func (d *DB) GetSetting(key, defaultVal string) string {
	d.mu.Lock()
	defer d.mu.Unlock()

	var val string
	err := d.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return defaultVal
	}
	return val
}

func (d *DB) GetAllSettings() (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			res[k] = v
		}
	}
	return res, nil
}

func (d *DB) GetTaskLogs(taskID string) ([]models.TaskLog, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT id, task_id, timestamp, message, created_at
		FROM task_logs
		WHERE task_id = ?
		ORDER BY id ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TaskLog
	for rows.Next() {
		var l models.TaskLog
		if err := rows.Scan(&l.ID, &l.TaskID, &l.Timestamp, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}
