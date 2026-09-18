use rusqlite::{params, Connection, Result};
use std::path::PathBuf;
use std::sync::{Arc, Mutex};

use crate::models::{DownloadStatus, DownloadTask};

#[derive(Clone)]
pub struct Database {
    conn: Arc<Mutex<Connection>>,
}

impl Database {

    #[allow(dead_code)]
    pub fn in_memory() -> Result<Self> {
        let conn = Connection::open_in_memory()?;
        conn.execute_batch("
            PRAGMA journal_mode = WAL;
            PRAGMA synchronous = NORMAL;

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
                progress REAL NOT NULL,
                speed TEXT NOT NULL,
                eta TEXT NOT NULL,
                error_msg TEXT,
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
        ")?;

        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
        })
    }

    pub fn init() -> Result<Self> {
        let db_path = Self::get_db_path();
        let conn = Connection::open(&db_path)?;

        // WAL mode for fast concurrent access
        conn.execute_batch("
            PRAGMA journal_mode = WAL;
            PRAGMA synchronous = NORMAL;

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
                progress REAL NOT NULL,
                speed TEXT NOT NULL,
                eta TEXT NOT NULL,
                error_msg TEXT,
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
        ")?;

        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
        })
    }

    pub fn get_db_path() -> PathBuf {
        if let Ok(exe) = std::env::current_exe() {
            if let Some(parent) = exe.parent() {
                return parent.join("idlix.db");
            }
        }
        PathBuf::from("idlix.db")
    }

    pub fn save_task(&self, task: &DownloadTask) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        let status_str = match task.status {
            DownloadStatus::Queued => "queued",
            DownloadStatus::Extracting => "extracting",
            DownloadStatus::Downloading => "downloading",
            DownloadStatus::Paused => "paused",
            DownloadStatus::Completed => "completed",
            DownloadStatus::Failed => "failed",
            DownloadStatus::Cancelled => "cancelled",
        };

        conn.execute(
            "INSERT OR REPLACE INTO tasks (
                id, title, media_type, year, season_num, episode_num,
                page_url, media_id, m3u8_url, subtitle_url, sub_lang,
                output_dir, file_name, status, progress, speed, eta, error_msg, created_at
            ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19)",
            params![
                task.id,
                task.title,
                task.media_type,
                task.year,
                task.season_num,
                task.episode_num,
                task.page_url,
                task.media_id,
                task.m3u8_url,
                task.subtitle_url,
                task.sub_lang,
                task.output_dir,
                task.file_name,
                status_str,
                task.progress as f64,
                task.speed,
                task.eta,
                task.error_msg,
                task.created_at,
            ],
        )?;
        Ok(())
    }

    pub fn get_all_tasks(&self) -> Result<Vec<DownloadTask>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(
            "SELECT id, title, media_type, year, season_num, episode_num,
                    page_url, media_id, m3u8_url, subtitle_url, sub_lang,
                    output_dir, file_name, status, progress, speed, eta, error_msg, created_at
             FROM tasks ORDER BY created_at DESC"
        )?;

        let task_iter = stmt.query_map([], |row| {
            let status_str: String = row.get(13)?;
            let status = match status_str.as_str() {
                "queued" => DownloadStatus::Queued,
                "extracting" => DownloadStatus::Extracting,
                "downloading" => DownloadStatus::Downloading,
                "paused" => DownloadStatus::Paused,
                "completed" => DownloadStatus::Completed,
                "failed" => DownloadStatus::Failed,
                "cancelled" => DownloadStatus::Cancelled,
                _ => DownloadStatus::Queued,
            };
            let progress_f64: f64 = row.get(14)?;

            Ok(DownloadTask {
                id: row.get(0)?,
                title: row.get(1)?,
                media_type: row.get(2)?,
                year: row.get(3)?,
                season_num: row.get(4)?,
                episode_num: row.get(5)?,
                page_url: row.get(6)?,
                media_id: row.get(7)?,
                m3u8_url: row.get(8)?,
                subtitle_url: row.get(9)?,
                sub_lang: row.get(10)?,
                output_dir: row.get(11)?,
                file_name: row.get(12)?,
                status,
                progress: progress_f64 as f32,
                speed: row.get(15)?,
                eta: row.get(16)?,
                error_msg: row.get(17)?,
                logs: Vec::new(),
                created_at: row.get(18)?,
            })
        })?;

        let mut tasks = Vec::new();
        for task in task_iter {
            tasks.push(task?);
        }
        Ok(tasks)
    }

    pub fn append_task_log(&self, task_id: &str, timestamp: &str, message: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;

        conn.execute(
            "INSERT INTO task_logs (task_id, timestamp, message, created_at) VALUES (?1, ?2, ?3, ?4)",
            params![task_id, timestamp, message, now],
        )?;
        Ok(())
    }

    pub fn get_task_logs(&self, task_id: &str) -> Result<Vec<String>> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(
            "SELECT timestamp, message FROM task_logs WHERE task_id = ?1 ORDER BY id ASC"
        )?;

        let log_iter = stmt.query_map(params![task_id], |row| {
            let timestamp: String = row.get(0)?;
            let message: String = row.get(1)?;
            Ok(format!("[{}] {}", timestamp, message))
        })?;

        let mut logs = Vec::new();
        for log in log_iter {
            logs.push(log?);
        }
        Ok(logs)
    }

    #[allow(dead_code)]
    pub fn delete_task(&self, task_id: &str) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        conn.execute("DELETE FROM task_logs WHERE task_id = ?1", params![task_id])?;
        conn.execute("DELETE FROM tasks WHERE id = ?1", params![task_id])?;
        Ok(())
    }
}
