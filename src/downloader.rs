use regex::Regex;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::io::AsyncReadExt;
use tokio::process::Command;
use tokio::sync::{broadcast, Mutex, RwLock};

use crate::bin_manager::BinPaths;
use crate::db::Database;
use crate::extractor::IdlixClient;
use crate::models::{AppConfig, DownloadStatus, DownloadTask};

pub async fn log_task_event(
    task_id: &str,
    message: &str,
    tasks_arc: &Arc<RwLock<Vec<DownloadTask>>>,
    tx: &broadcast::Sender<DownloadTask>,
    db: &Database,
) {
    let now_secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    let sec = now_secs % 60;
    let min = (now_secs / 60) % 60;
    let hour = (now_secs / 3600) % 24;
    let timestamp = format!("{:02}:{:02}:{:02}", hour, min, sec);
    let formatted_line = format!("[{}] {}", timestamp, message);

    // 1. Write to SQLite database
    let _ = db.append_task_log(task_id, &timestamp, message);

    // 2. Append to memory task.logs buffer (keep last 150 lines for instant UI streaming)
    let mut updated_task = None;
    {
        let mut tasks = tasks_arc.write().await;
        if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
            t.logs.push(formatted_line);
            if t.logs.len() > 150 {
                let excess = t.logs.len() - 150;
                t.logs.drain(0..excess);
            }
            updated_task = Some(t.clone());
        }
    }

    if let Some(t) = updated_task {
        let _ = tx.send(t);
    }
}

pub struct DownloadManager {
    pub tasks: Arc<RwLock<Vec<DownloadTask>>>,
    pub tx: broadcast::Sender<DownloadTask>,
    pub bin_paths: BinPaths,
    pub config: Arc<RwLock<AppConfig>>,
    pub extractor: Arc<IdlixClient>,
    pub db: Database,
    active_workers: Arc<Mutex<usize>>,
}

impl DownloadManager {
    pub fn new(bin_paths: BinPaths, config: AppConfig, extractor: Arc<IdlixClient>, db: Database) -> Self {
        let (tx, _) = broadcast::channel(100);
        let mut existing_tasks = db.get_all_tasks().unwrap_or_default();
        for t in &mut existing_tasks {
            if t.status == DownloadStatus::Downloading || t.status == DownloadStatus::Extracting {
                t.status = DownloadStatus::Paused;
                let _ = db.save_task(t);
            }
        }

        Self {
            tasks: Arc::new(RwLock::new(existing_tasks)),
            tx,
            bin_paths,
            config: Arc::new(RwLock::new(config)),
            extractor,
            db,
            active_workers: Arc::new(Mutex::new(0)),
        }
    }

    pub fn subscribe(&self) -> broadcast::Receiver<DownloadTask> {
        self.tx.subscribe()
    }

    pub async fn add_task(
        &self,
        title: String,
        media_type: String,
        year: Option<String>,
        season_num: Option<u32>,
        episode_num: Option<u32>,
        page_url: Option<String>,
        media_id: Option<String>,
        m3u8_url: String,
        subtitle_url: Option<String>,
        sub_lang: Option<String>,
        custom_out_dir: Option<String>,
    ) -> Result<DownloadTask, String> {
        let conf = self.config.read().await;

        let (out_dir, file_name) = if media_type == "series" || media_type == "tv" {
            let s_num = season_num.unwrap_or(1);
            let ep_num = episode_num.unwrap_or(1);
            let clean_title = sanitize_filename(&title);
            let series_folder_name = format_jellyfin_title(&clean_title, year.as_deref(), page_url.as_deref());
            let target_series_dir = custom_out_dir.unwrap_or_else(|| {
                if !conf.series_dir.is_empty() {
                    conf.series_dir.clone()
                } else {
                    PathBuf::from(&conf.download_dir)
                        .join("TV Series")
                        .to_string_lossy()
                        .to_string()
                }
            });
            let folder = PathBuf::from(&target_series_dir)
                .join(&series_folder_name)
                .join(format!("Season {:02}", s_num));
            let name = format!("{} - S{:02}E{:02}", clean_title, s_num, ep_num);
            (folder.to_string_lossy().to_string(), name)
        } else {
            let movie_name = format_jellyfin_title(&title, year.as_deref(), page_url.as_deref());
            let target_movies_dir = custom_out_dir.unwrap_or_else(|| {
                if !conf.movies_dir.is_empty() {
                    conf.movies_dir.clone()
                } else {
                    PathBuf::from(&conf.download_dir)
                        .join("Movies")
                        .to_string_lossy()
                        .to_string()
                }
            });
            let folder = PathBuf::from(&target_movies_dir).join(&movie_name);
            let name = movie_name;
            (folder.to_string_lossy().to_string(), name)
        };

        // Guard against duplicate active tasks targeting the exact same output path
        {
            let tasks = self.tasks.read().await;
            if let Some(existing) = tasks.iter().find(|t| {
                t.output_dir == out_dir
                    && t.file_name == file_name
                    && (t.status == DownloadStatus::Queued
                        || t.status == DownloadStatus::Extracting
                        || t.status == DownloadStatus::Downloading)
            }) {
                return Err(format!(
                    "Download untuk '{}' sudah berjalan atau berada dalam antrean (Status: {:?})",
                    existing.title, existing.status
                ));
            }
        }

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs() as i64;

        let task = DownloadTask {
            id: uuid::Uuid::new_v4().to_string(),
            title,
            media_type,
            year,
            season_num,
            episode_num,
            page_url,
            media_id,
            m3u8_url,
            subtitle_url,
            sub_lang,
            output_dir: out_dir,
            file_name,
            status: DownloadStatus::Queued,
            progress: 0.0,
            speed: "0 KB/s".to_string(),
            eta: "Queued...".to_string(),
            error_msg: None,
            logs: Vec::new(),
            created_at: now,
        };

        let _ = self.db.save_task(&task);

        {
            let mut tasks = self.tasks.write().await;
            tasks.push(task.clone());
        }

        let _ = self.tx.send(task.clone());

        // Trigger worker pool check
        self.spawn_queue_worker();

        Ok(task)
    }

    pub async fn retry_task(&self, task_id: &str) -> Result<DownloadTask, String> {
        let mut target_task = None;
        {
            let mut tasks = self.tasks.write().await;
            if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                if t.status == DownloadStatus::Downloading || t.status == DownloadStatus::Extracting {
                    return Err("Download sedang aktif berjalan".to_string());
                }
                clean_leftover_artifacts(&t.output_dir, &t.file_name);
                t.status = DownloadStatus::Queued;
                t.progress = 0.0;
                t.speed = "0 KB/s".to_string();
                t.eta = "Queued...".to_string();
                t.error_msg = None;
                let _ = self.db.save_task(t);
                target_task = Some(t.clone());
            }
        }

        if let Some(task) = target_task {
            log_task_event(
                &task.id,
                "[Init] Task diulang (retry) oleh user.",
                &self.tasks,
                &self.tx,
                &self.db,
            )
            .await;
            let _ = self.tx.send(task.clone());
            self.spawn_queue_worker();
            Ok(task)
        } else {
            Err("Task tidak ditemukan".to_string())
        }
    }

    pub async fn delete_task(&self, task_id: &str) -> Result<(), String> {
        {
            let mut tasks = self.tasks.write().await;
            if let Some(pos) = tasks.iter().position(|t| t.id == task_id) {
                let t = &tasks[pos];
                if t.status != DownloadStatus::Completed {
                    clean_leftover_artifacts(&t.output_dir, &t.file_name);
                }
                tasks.remove(pos);
            } else {
                return Err("Task tidak ditemukan".to_string());
            }
        }
        let _ = self.db.delete_task(task_id);
        Ok(())
    }

    pub fn spawn_queue_worker(&self) {
        let tasks_arc = Arc::clone(&self.tasks);
        let active_workers = Arc::clone(&self.active_workers);
        let config_arc = Arc::clone(&self.config);
        let tx = self.tx.clone();
        let bin_paths = self.bin_paths.clone();
        let extractor = Arc::clone(&self.extractor);
        let db = self.db.clone();

        Self::spawn_worker_loop(tasks_arc, active_workers, config_arc, tx, bin_paths, extractor, db);
    }

    fn spawn_worker_loop(
        tasks_arc: Arc<RwLock<Vec<DownloadTask>>>,
        active_workers: Arc<Mutex<usize>>,
        config_arc: Arc<RwLock<AppConfig>>,
        tx: broadcast::Sender<DownloadTask>,
        bin_paths: BinPaths,
        extractor: Arc<IdlixClient>,
        db: Database,
    ) {
        tokio::spawn(async move {
            let max_concurrent = {
                let conf = config_arc.read().await;
                conf.max_concurrent_downloads
            };

            let mut count = active_workers.lock().await;
            if *count >= max_concurrent {
                return;
            }

            // Find next queued task
            let mut next_task = None;
            {
                let mut tasks = tasks_arc.write().await;
                for t in tasks.iter_mut() {
                    if t.status == DownloadStatus::Queued {
                        t.status = DownloadStatus::Extracting;
                        t.eta = "Preparing stream...".to_string();
                        let _ = db.save_task(t);
                        next_task = Some(t.clone());
                        let _ = tx.send(t.clone());
                        break;
                    }
                }
            }

            if let Some(task) = next_task {
                *count += 1;
                drop(count);

                let tasks_inner = Arc::clone(&tasks_arc);
                let tx_inner = tx.clone();
                let bins = bin_paths.clone();
                let workers_inner = Arc::clone(&active_workers);
                let extractor_inner = Arc::clone(&extractor);
                let config_inner = Arc::clone(&config_arc);
                let db_inner = db.clone();

                tokio::spawn(async move {
                    Self::execute_task(
                        task,
                        tasks_inner.clone(),
                        tx_inner.clone(),
                        bins.clone(),
                        extractor_inner.clone(),
                        db_inner.clone(),
                    )
                    .await;

                    {
                        let mut c = workers_inner.lock().await;
                        if *c > 0 {
                            *c -= 1;
                        }
                    }

                    // Trigger next queued task
                    Self::spawn_worker_loop(
                        tasks_inner,
                        workers_inner,
                        config_inner,
                        tx_inner,
                        bins,
                        extractor_inner,
                        db_inner,
                    );
                });
            }
        });
    }

    async fn execute_task(
        mut task: DownloadTask,
        tasks_arc: Arc<RwLock<Vec<DownloadTask>>>,
        tx: broadcast::Sender<DownloadTask>,
        bin_paths: BinPaths,
        extractor: Arc<IdlixClient>,
        db: Database,
    ) {
        // 0. Pre-flight check: verify if destination file already exists and is valid (> 1MB)
        if let Some(size) = check_existing_valid_file(&task.output_dir, &task.file_name) {
            let size_mb = size as f64 / (1024.0 * 1024.0);
            log_task_event(
                &task.id,
                &format!(
                    "[SUCCESS] File video .mp4 sudah ada di folder tujuan ({:.2} MB). Melewati proses download.",
                    size_mb
                ),
                &tasks_arc,
                &tx,
                &db,
            )
            .await;

            let mut tasks = tasks_arc.write().await;
            if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                t.status = DownloadStatus::Completed;
                t.progress = 100.0;
                t.speed = "Existing File".to_string();
                t.eta = "00:00:00".to_string();
                t.error_msg = None;
                let _ = db.save_task(t);
                task = t.clone();
            }
            let _ = tx.send(task);
            return;
        }

        // Clean any corrupted or partial leftover artifacts before starting
        clean_leftover_artifacts(&task.output_dir, &task.file_name);

        log_task_event(
            &task.id,
            &format!("Initializing download task: '{}' ({})", task.title, task.media_type),
            &tasks_arc,
            &tx,
            &db,
        )
        .await;

        // 1. If m3u8_url is empty, extract stream sources now
        if task.m3u8_url.is_empty() {
            {
                let mut tasks = tasks_arc.write().await;
                if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                    t.status = DownloadStatus::Extracting;
                    t.eta = "Unlocking stream (~15s)...".to_string();
                    let _ = db.save_task(t);
                    task = t.clone();
                }
            }
            let _ = tx.send(task.clone());

            let page_url = task.page_url.clone().unwrap_or_default();
            let slug_or_id = if let Some(ref mid) = task.media_id {
                mid.as_str()
            } else if !page_url.is_empty() {
                page_url.trim_matches('/').split('/').last().unwrap_or(&task.title)
            } else {
                &task.title
            };

            log_task_event(
                &task.id,
                &format!("Extracting stream sources for '{}' (URL: '{}')...", slug_or_id, page_url),
                &tasks_arc,
                &tx,
                &db,
            )
            .await;

            match extractor.extract_stream(&task.media_type, slug_or_id, &page_url).await {
                Ok(sources) => {
                    log_task_event(
                        &task.id,
                        &format!("Stream extracted successfully. M3U8: {}", sources.m3u8_url),
                        &tasks_arc,
                        &tx,
                        &db,
                    )
                    .await;

                    let mut chosen_sub = None;
                    if let Some(ref target_lang) = task.sub_lang {
                        for s in &sources.subtitles {
                            if s.lang.to_lowercase().contains(&target_lang.to_lowercase()) {
                                chosen_sub = Some(s.url.clone());
                                break;
                            }
                        }
                        if chosen_sub.is_none() && !sources.subtitles.is_empty() {
                            chosen_sub = Some(sources.subtitles[0].url.clone());
                        }
                    }

                    if let Some(ref s_url) = chosen_sub {
                        log_task_event(
                            &task.id,
                            &format!("Subtitle chosen: {}", s_url),
                            &tasks_arc,
                            &tx,
                            &db,
                        )
                        .await;
                    }

                    {
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                            t.m3u8_url = sources.m3u8_url.clone();
                            t.subtitle_url = chosen_sub;
                            t.status = DownloadStatus::Downloading;
                            t.eta = "Starting download...".to_string();
                            let _ = db.save_task(t);
                            task = t.clone();
                        }
                    }
                    let _ = tx.send(task.clone());
                }
                Err(e) => {
                    let err_str = format!("Extraction error: {e}");
                    log_task_event(
                        &task.id,
                        &format!("[ERROR] {}", err_str),
                        &tasks_arc,
                        &tx,
                        &db,
                    )
                    .await;
                    {
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                            t.status = DownloadStatus::Failed;
                            t.error_msg = Some(err_str);
                            let _ = db.save_task(t);
                            task = t.clone();
                        }
                    }
                    let _ = tx.send(task);
                    return;
                }
            }
        }

        let _ = tokio::fs::create_dir_all(&task.output_dir).await;

        // Download subtitle if present
        if let Some(ref sub_url) = task.subtitle_url {
            let srt_path = PathBuf::from(&task.output_dir).join(format!("{}.srt", task.file_name));
            log_task_event(
                &task.id,
                &format!("Downloading subtitle to: {:?}", srt_path),
                &tasks_arc,
                &tx,
                &db,
            )
            .await;
            if let Err(e) = download_and_convert_vtt_to_srt(sub_url, &srt_path).await {
                log_task_event(
                    &task.id,
                    &format!("[WARN] Subtitle download/conversion failed: {}", e),
                    &tasks_arc,
                    &tx,
                    &db,
                )
                .await;
            } else {
                log_task_event(
                    &task.id,
                    "Subtitle downloaded and converted to SRT successfully",
                    &tasks_arc,
                    &tx,
                    &db,
                )
                .await;
            }
        }

        // Run N_m3u8DL-RE command with automatic MP4 remuxing
        let ref_header = format!("Referer: {}/", extractor.base_url.trim_end_matches('/'));
        let mut cmd = Command::new(&bin_paths.n_m3u8dl_re);
        cmd.arg(&task.m3u8_url)
            .arg("--save-dir")
            .arg(&task.output_dir)
            .arg("--save-name")
            .arg(&task.file_name)
            .arg("--auto-select")
            .arg("--force-ansi-console")
            .arg("--thread-count")
            .arg("16")
            .arg("--download-retry-count")
            .arg("3")
            .arg("-M")
            .arg("format=mp4:muxer=ffmpeg")
            .arg("-H")
            .arg("User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
            .arg("-H")
            .arg(&ref_header)
            .arg("--ffmpeg-binary-path")
            .arg(&bin_paths.ffmpeg)
            .arg("--log-level")
            .arg("INFO")
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped());

        #[cfg(windows)]
        {
            // CREATE_NO_WINDOW flag on Windows
            cmd.creation_flags(0x08000000);
        }

        log_task_event(
            &task.id,
            &format!(
                "Executing N_m3u8DL-RE command for '{}' -> Output: '{}/{}'",
                task.title, task.output_dir, task.file_name
            ),
            &tasks_arc,
            &tx,
            &db,
        )
        .await;

        let mut child = match cmd.spawn() {
            Ok(c) => c,
            Err(e) => {
                let err_str = format!("Failed to spawn downloader binary: {e}");
                log_task_event(
                    &task.id,
                    &format!("[ERROR] {}", err_str),
                    &tasks_arc,
                    &tx,
                    &db,
                )
                .await;
                task.status = DownloadStatus::Failed;
                task.error_msg = Some(err_str);
                let _ = db.save_task(&task);
                let mut tasks = tasks_arc.write().await;
                if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                    *t = task.clone();
                }
                let _ = tx.send(task);
                return;
            }
        };

        let stdout = child.stdout.take();
        let stderr = child.stderr.take();

        let out_handle = if let Some(out) = stdout {
            let tasks_clone = Arc::clone(&tasks_arc);
            let tx_clone = tx.clone();
            let task_id = task.id.clone();
            let db_clone = db.clone();
            Some(tokio::spawn(process_stream_output(out, task_id, tasks_clone, tx_clone, db_clone)))
        } else {
            None
        };

        let err_handle = if let Some(err) = stderr {
            let tasks_clone = Arc::clone(&tasks_arc);
            let tx_clone = tx.clone();
            let task_id = task.id.clone();
            let db_clone = db.clone();
            Some(tokio::spawn(process_stream_output(err, task_id, tasks_clone, tx_clone, db_clone)))
        } else {
            None
        };

        let status = child.wait().await;
        let success = status.map(|s| s.success()).unwrap_or(false);

        if let Some(h) = out_handle {
            let _ = h.await;
        }
        if let Some(h) = err_handle {
            let _ = h.await;
        }

        if success {
            if let Some(size) = check_existing_valid_file(&task.output_dir, &task.file_name) {
                let size_mb = size as f64 / (1024.0 * 1024.0);
                log_task_event(
                    &task.id,
                    &format!(
                        "[SUCCESS] Download dan remuxing MP4 selesai dengan sukses! Ukuran file: {:.2} MB",
                        size_mb
                    ),
                    &tasks_arc,
                    &tx,
                    &db,
                )
                .await;
                let mut tasks = tasks_arc.write().await;
                if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                    t.status = DownloadStatus::Completed;
                    t.progress = 100.0;
                    t.speed = "Done".to_string();
                    t.eta = "00:00:00".to_string();
                    t.error_msg = None;
                    let _ = db.save_task(t);
                    task = t.clone();
                }
            } else {
                let err_msg = "Download selesai namun file video .mp4 tidak ditemukan atau corrupt (<= 1MB).".to_string();
                log_task_event(
                    &task.id,
                    &format!("[ERROR] {}", err_msg),
                    &tasks_arc,
                    &tx,
                    &db,
                )
                .await;
                let mut tasks = tasks_arc.write().await;
                if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                    t.status = DownloadStatus::Failed;
                    t.error_msg = Some(err_msg);
                    let _ = db.save_task(t);
                    task = t.clone();
                }
            }
        } else {
            let captured_logs = {
                let tasks = tasks_arc.read().await;
                tasks
                    .iter()
                    .find(|t| t.id == task.id)
                    .map(|t| t.logs.clone())
                    .unwrap_or_default()
            };
            let reason = extract_error_reason(&captured_logs);
            log_task_event(
                &task.id,
                &format!("[ERROR] Download process failed: {}", reason),
                &tasks_arc,
                &tx,
                &db,
            )
            .await;
            let mut tasks = tasks_arc.write().await;
            if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                t.status = DownloadStatus::Failed;
                t.error_msg = Some(reason);
                let _ = db.save_task(t);
                task = t.clone();
            }
        }

        let _ = tx.send(task);
    }
}

pub fn extract_error_reason(logs: &[String]) -> String {
    for line in logs.iter().rev() {
        let trimmed = line.trim();
        let clean = if let Some(idx) = trimmed.find("] ") {
            &trimmed[idx + 2..]
        } else {
            trimmed
        };

        if clean.contains("403") && (clean.contains("Forbidden") || clean.contains("status code")) {
            return "HTTP 403 Forbidden: Link stream telah kedaluwarsa atau diblokir server".to_string();
        }
        if clean.contains("404") && (clean.contains("Not Found") || clean.contains("status code")) {
            return "HTTP 404 Not Found: File segmen video tidak ditemukan di server sumber".to_string();
        }
        if clean.contains("HttpRequestException") {
            let msg = clean.split("HttpRequestException:").last().unwrap_or(clean).trim();
            return format!("Network Error: {}", msg);
        }
        if let Some(pos) = clean.find("ERROR :") {
            let msg = clean[pos + 7..].trim();
            if !msg.is_empty() {
                return msg.to_string();
            }
        }
        if let Some(pos) = clean.find("[ERROR]") {
            let msg = clean[pos + 7..].trim();
            if !msg.is_empty() {
                return msg.to_string();
            }
        }
        if clean.contains("Exception:") {
            return clean.to_string();
        }
        if (clean.contains("muxer") || clean.contains("ffmpeg")) && clean.to_lowercase().contains("error") {
            return format!("Remux Error: {}", clean);
        }
    }
    "Downloader process exited with error (Exit code != 0)".to_string()
}

pub fn format_speed_display(raw: &str) -> String {
    let s = raw.trim().trim_start_matches('-').trim();
    if s.is_empty() {
        return "0 B/s".to_string();
    }
    let mut formatted = s.to_string();
    if formatted.ends_with("MBps") {
        formatted = formatted.replace("MBps", " MB/s");
    } else if formatted.ends_with("KBps") {
        formatted = formatted.replace("KBps", " KB/s");
    } else if formatted.ends_with("GBps") {
        formatted = formatted.replace("GBps", " GB/s");
    } else if formatted.ends_with("Bps") {
        formatted = formatted.replace("Bps", " B/s");
    }
    formatted
}

async fn process_stream_output<R: tokio::io::AsyncRead + Unpin>(
    mut reader: R,
    task_id: String,
    tasks_arc: Arc<RwLock<Vec<DownloadTask>>>,
    tx: broadcast::Sender<DownloadTask>,
    db: Database,
) {
    let progress_regex = Regex::new(
        r"(\d+)\s*/\s*(\d+)\s+(\d+(?:\.\d+)?)%(?:\s+[\d\.]+[kKMmGg]?[bB]/[\d\.]+[kKMmGg]?[bB])?\s*-?(\d+(?:\.\d+)?\s*(?:[kKMmGg][iI]?[bB](?:/s|ps)|[bB]ps))?\s*(\d{1,2}:\d{2}:\d{2}|--:--:--)?"
    ).unwrap();
    let fallback_progress_regex = Regex::new(r"(\d+(?:\.\d+)?)%").unwrap();
    let fallback_seg_regex = Regex::new(r"(\d+)\s*/\s*(\d+)").unwrap();
    let fallback_speed_regex = Regex::new(r"(?:^|\s)-?(\d+(?:\.\d+)?\s*(?:[kKMmGg][iI]?[bB](?:/s|ps)|[bB]ps))").unwrap();
    let fallback_eta_regex = Regex::new(r"(\d{1,2}:\d{2}:\d{2})").unwrap();
    let ansi_regex = Regex::new(r"(?:[@-Z\-_]|\[[0-?]*[ -/]*[@-~])").unwrap();

    let mut buf = [0u8; 4096];
    let mut rolling_buf = String::new();
    let mut last_emit = std::time::Instant::now();

    while let Ok(n) = reader.read(&mut buf).await {
        if n == 0 {
            break;
        }

        let raw_str = String::from_utf8_lossy(&buf[..n]);
        let clean = ansi_regex.replace_all(&raw_str, "").to_string();
        rolling_buf.push_str(&clean);

        // 1. Process complete lines delimited by \r or \n
        while let Some(pos) = rolling_buf.find(['\r', '\n']) {
            let line = rolling_buf[..pos].trim().to_string();
            rolling_buf = rolling_buf[pos + 1..].to_string();

            if line.is_empty() {
                continue;
            }

            // Check if line contains a progress update
            if let Some(cap) = progress_regex.captures_iter(&line).last() {
                let mut updated = false;
                let mut current_task = None;

                {
                    let mut tasks = tasks_arc.write().await;
                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                        if let Some(pct_m) = cap.get(3) {
                            if let Ok(val) = pct_m.as_str().parse::<f32>() {
                                if (val - t.progress).abs() >= 0.05 || val >= 100.0 {
                                    t.progress = val;
                                    t.status = DownloadStatus::Downloading;
                                    updated = true;
                                }
                            }
                        }
                        if let Some(sp_m) = cap.get(4) {
                            let sp = format_speed_display(sp_m.as_str());
                            if !sp.is_empty() && t.speed != sp {
                                t.speed = sp;
                                updated = true;
                            }
                        }
                        if let Some(eta_m) = cap.get(5) {
                            let eta = eta_m.as_str().trim().to_string();
                            if eta != "--:--:--" && !eta.is_empty() && t.eta != eta {
                                t.eta = eta;
                                updated = true;
                            }
                        }
                        if updated {
                            current_task = Some(t.clone());
                        }
                    }
                }

                if let Some(t) = current_task {
                    if last_emit.elapsed().as_millis() >= 80 || t.progress >= 100.0 {
                        let _ = tx.send(t);
                        last_emit = std::time::Instant::now();
                    }
                }
            }

            // Check for log messages
            let is_log_line = line.contains("INFO :")
                || line.contains("WARN :")
                || line.contains("ERROR :")
                || line.contains("DEBUG :")
                || line.contains("Exception")
                || line.contains("Error")
                || line.contains("ffmpeg")
                || line.contains("Selected")
                || line.contains("Merging")
                || line.contains("Writing")
                || line.contains("Finished")
                || line.contains("Done");

            if is_log_line {
                // If progress was attached to the end of a log line, extract only the log portion
                let clean_log = if let Some(m) = progress_regex.find(&line) {
                    line[..m.start()].trim()
                } else {
                    line.as_str()
                };
                if !clean_log.is_empty() {
                    log_task_event(&task_id, clean_log, &tasks_arc, &tx, &db).await;
                }
            }
        }

        // 2. Also check un-delimited rolling_buf for live real-time progress updates!
        if !rolling_buf.is_empty() {
            let mut updated = false;
            let mut current_task = None;

            if let Some(cap) = progress_regex.captures_iter(&rolling_buf).last() {
                {
                    let mut tasks = tasks_arc.write().await;
                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                        if let Some(pct_m) = cap.get(3) {
                            if let Ok(val) = pct_m.as_str().parse::<f32>() {
                                if (val - t.progress).abs() >= 0.05 || val >= 100.0 {
                                    t.progress = val;
                                    t.status = DownloadStatus::Downloading;
                                    updated = true;
                                }
                            }
                        }
                        if let Some(sp_m) = cap.get(4) {
                            let sp = format_speed_display(sp_m.as_str());
                            if !sp.is_empty() && t.speed != sp {
                                t.speed = sp;
                                updated = true;
                            }
                        }
                        if let Some(eta_m) = cap.get(5) {
                            let eta = eta_m.as_str().trim().to_string();
                            if eta != "--:--:--" && !eta.is_empty() && t.eta != eta {
                                t.eta = eta;
                                updated = true;
                            }
                        }
                        if updated {
                            current_task = Some(t.clone());
                        }
                    }
                }
            } else if let Some(cap) = fallback_progress_regex.captures_iter(&rolling_buf).last() {
                if let Ok(val) = cap[1].parse::<f32>() {
                    let mut tasks = tasks_arc.write().await;
                    if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                        if (val - t.progress).abs() >= 0.05 || val >= 100.0 {
                            t.progress = val;
                            t.status = DownloadStatus::Downloading;
                            updated = true;
                        }
                        if let Some(sp_cap) = fallback_speed_regex.captures_iter(&rolling_buf).last() {
                            let sp = format_speed_display(&sp_cap[1]);
                            if !sp.is_empty() && t.speed != sp {
                                t.speed = sp;
                                updated = true;
                            }
                        }
                        if let Some(eta_cap) = fallback_eta_regex.captures_iter(&rolling_buf).last() {
                            let eta = eta_cap[1].trim().to_string();
                            if !eta.is_empty() && t.eta != eta {
                                t.eta = eta;
                                updated = true;
                            }
                        }
                        if updated {
                            current_task = Some(t.clone());
                        }
                    }
                }
            } else if let Some(cap) = fallback_seg_regex.captures_iter(&rolling_buf).last() {
                if let (Ok(curr), Ok(total)) = (cap[1].parse::<f32>(), cap[2].parse::<f32>()) {
                    if total > 0.0 && curr <= total {
                        let val = ((curr / total) * 100.0).min(100.0);
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                            if (val - t.progress).abs() >= 0.05 {
                                t.progress = val;
                                t.status = DownloadStatus::Downloading;
                                current_task = Some(t.clone());
                            }
                        }
                    }
                }
            }

            if let Some(t) = current_task {
                if last_emit.elapsed().as_millis() >= 80 || t.progress >= 100.0 {
                    let _ = tx.send(t);
                    last_emit = std::time::Instant::now();
                }
            }

            // Check if rolling_buf has log lines preceding stream progress markers
            for marker in ["Vid ", "Aud ", "Sub ", "-----"] {
                if let Some(idx) = rolling_buf.find(marker) {
                    let log_candidate = rolling_buf[..idx].trim().to_string();
                    if log_candidate.contains("INFO :")
                        || log_candidate.contains("WARN :")
                        || log_candidate.contains("ERROR :")
                        || log_candidate.contains("DEBUG :")
                    {
                        log_task_event(&task_id, &log_candidate, &tasks_arc, &tx, &db).await;
                        rolling_buf = rolling_buf[idx..].to_string();
                        break;
                    }
                }
            }

            // Keep rolling_buf bounded
            if rolling_buf.len() > 1024 {
                let keep_len = 512.min(rolling_buf.len());
                rolling_buf = rolling_buf[rolling_buf.len() - keep_len..].to_string();
            }
        }
    }
}

pub fn check_existing_valid_file(output_dir: &str, file_name: &str) -> Option<u64> {
    let target_mp4 = PathBuf::from(output_dir).join(format!("{}.mp4", file_name));
    if target_mp4.exists() {
        if let Ok(meta) = std::fs::metadata(&target_mp4) {
            if meta.len() > 1024 * 1024 {
                return Some(meta.len());
            }
        }
    }
    None
}

pub fn clean_leftover_artifacts(output_dir: &str, file_name: &str) {
    let out_path = Path::new(output_dir);
    if !out_path.exists() {
        return;
    }
    let target_mp4 = out_path.join(format!("{}.mp4", file_name));
    if target_mp4.exists() {
        if let Ok(meta) = std::fs::metadata(&target_mp4) {
            if meta.len() <= 1024 * 1024 {
                let _ = std::fs::remove_file(&target_mp4);
            }
        }
    }
    if let Ok(entries) = std::fs::read_dir(out_path) {
        for entry in entries.flatten() {
            let path = entry.path();
            if let Some(name_str) = path.file_name().and_then(|n| n.to_str()) {
                if name_str.starts_with(file_name) {
                    let is_temp = name_str.ends_with(".tmp")
                        || name_str.ends_with(".part")
                        || name_str.ends_with(".m4s")
                        || name_str.ends_with(".ts")
                        || name_str.ends_with(".aria2")
                        || name_str.ends_with(".temp")
                        || name_str.ends_with(".m3u8");
                    if is_temp {
                        if path.is_file() {
                            let _ = std::fs::remove_file(&path);
                        } else if path.is_dir() {
                            let _ = std::fs::remove_dir_all(&path);
                        }
                    }
                }
            }
        }
    }
}

pub fn sanitize_filename(name: &str) -> String {
    let invalid = ['\\', '/', ':', '*', '?', '"', '<', '>', '|'];
    name.chars()
        .map(|c| if invalid.contains(&c) { '_' } else { c })
        .collect::<String>()
        .trim()
        .to_string()
}

pub fn format_jellyfin_title(title: &str, year: Option<&str>, page_url: Option<&str>) -> String {
    let clean_title = sanitize_filename(title);

    let detected_year = year
        .filter(|y| !y.trim().is_empty() && y.trim() != "N/A")
        .map(|y| y.trim().to_string())
        .or_else(|| {
            if let Some(url) = page_url {
                let re = Regex::new(r"[-_/](\d{4})(?:/|$)").ok()?;
                re.captures(url).and_then(|c| c.get(1)).map(|m| m.as_str().to_string())
            } else {
                None
            }
        });

    if let Some(y) = detected_year {
        let year_pattern = format!("({})", y);
        if clean_title.contains(&year_pattern) {
            clean_title
        } else if clean_title.ends_with(&y) {
            let without_year = clean_title[..clean_title.len() - y.len()].trim_end_matches([' ', '-', '_']);
            format!("{} ({})", without_year, y)
        } else {
            format!("{} ({})", clean_title, y)
        }
    } else {
        clean_title
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_nm3u8dl_output() {
        use tokio::io::AsyncReadExt;
        let mut cmd = Command::new("./target/release/bin/N_m3u8DL-RE.exe");
        cmd.arg("https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8")
            .arg("--auto-select")
            .arg("--force-ansi-console")
            .arg("--save-dir")
            .arg("./tmp_test_dl")
            .arg("--save-name")
            .arg("test")
            .arg("-M")
            .arg("format=mp4:muxer=ffmpeg")
            .arg("--ffmpeg-binary-path")
            .arg("./target/release/bin/ffmpeg.exe")
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped());

        let mut child = cmd.spawn().expect("failed to spawn");
        let mut stdout = child.stdout.take().unwrap();
        let mut stderr = child.stderr.take().unwrap();

        let h1 = tokio::spawn(async move {
            let mut buf = [0u8; 1024];
            while let Ok(n) = stdout.read(&mut buf).await {
                if n == 0 { break; }
                let text = String::from_utf8_lossy(&buf[..n]);
                println!("[STDOUT RAW {:?}]: {}", n, text);
            }
        });

        let h2 = tokio::spawn(async move {
            let mut buf = [0u8; 1024];
            while let Ok(n) = stderr.read(&mut buf).await {
                if n == 0 { break; }
                let text = String::from_utf8_lossy(&buf[..n]);
                println!("[STDERR RAW {:?}]: {}", n, text);
            }
        });

        let _ = child.wait().await;
        let _ = tokio::join!(h1, h2);
        let _ = tokio::fs::remove_dir_all("./tmp_test_dl").await;
    }

    #[test]
    fn test_format_jellyfin_title() {
        assert_eq!(
            format_jellyfin_title("Moana", Some("2016"), None),
            "Moana (2016)"
        );
        assert_eq!(
            format_jellyfin_title("Moana (2016)", Some("2016"), None),
            "Moana (2016)"
        );
        assert_eq!(
            format_jellyfin_title("Moana 2016", Some("2016"), None),
            "Moana (2016)"
        );
        assert_eq!(
            format_jellyfin_title("Moana", None, Some("https://z2.idlixku.com/movie/moana-2016")),
            "Moana (2016)"
        );
        assert_eq!(
            format_jellyfin_title("Avatar: The Way of Water", Some("2022"), None),
            "Avatar_ The Way of Water (2022)"
        );
    }

    #[tokio::test]
    async fn test_process_stream_output_parsing() {
        let (tx, mut rx) = broadcast::channel(100);
        let db = Database::in_memory().unwrap();
        let task = DownloadTask {
            id: "task_1".to_string(),
            title: "Test".to_string(),
            media_type: "movie".to_string(),
            year: None,
            season_num: None,
            episode_num: None,
            page_url: None,
            media_id: None,
            m3u8_url: "".to_string(),
            subtitle_url: None,
            sub_lang: None,
            output_dir: "".to_string(),
            file_name: "".to_string(),
            status: DownloadStatus::Queued,
            progress: 0.0,
            speed: "".to_string(),
            eta: "".to_string(),
            error_msg: None,
            logs: Vec::new(),
            created_at: 0,
        };

        let tasks_arc = Arc::new(RwLock::new(vec![task]));
        // Continuous un-delimited output from N_m3u8DL-RE (no \r or \n during segment downloads)
        let simulated_output = b"Vid 1920x1080 | 6221 Kbps ------------------------------ 1/64 1.56% 5.81MB/371.85MB5.81MBps00:00:32Vid 1920x1080 | 6221 Kbps ------------------------------ 50/64 78.12% 423.30MB/564.40MB11.04MBps00:00:09Vid 1920x1080 | 6221 Kbps ------------------------------ 64/64 100.00% 478.94MB/478.94MB12.45MBps00:00:00";

        process_stream_output(
            &simulated_output[..],
            "task_1".to_string(),
            tasks_arc.clone(),
            tx,
            db,
        )
        .await;

        let final_task = {
            let tasks = tasks_arc.read().await;
            tasks[0].clone()
        };

        assert_eq!(final_task.progress, 100.0);
        assert_eq!(final_task.speed, "12.45 MB/s");
        assert_eq!(final_task.eta, "00:00:00");
        assert_eq!(final_task.status, DownloadStatus::Downloading);

        // Check that broadcast received updates
        let mut msg_count = 0;
        while let Ok(_t) = rx.try_recv() {
            msg_count += 1;
        }
        assert!(msg_count > 0, "Should have broadcasted progress updates");
    }
}

async fn download_and_convert_vtt_to_srt(vtt_url: &str, srt_path: &Path) -> Result<(), String> {
    let client = reqwest::Client::new();
    let body = client
        .get(vtt_url)
        .header("User-Agent", "Mozilla/5.0")
        .send()
        .await
        .map_err(|e| format!("Failed to fetch subtitle: {e}"))?
        .text()
        .await
        .map_err(|e| format!("Failed to read subtitle text: {e}"))?;

    let srt_content = vtt_to_srt(&body);
    tokio::fs::write(srt_path, srt_content)
        .await
        .map_err(|e| format!("Failed to write srt file: {e}"))?;

    Ok(())
}

fn vtt_to_srt(vtt: &str) -> String {
    let mut srt_lines = Vec::new();
    let lines: Vec<&str> = vtt.lines().collect();
    let mut counter = 1;
    let time_re = Regex::new(r"(\d{2}:\d{2}(?::\d{2})?\.\d{3})\s*-->\s*(\d{2}:\d{2}(?::\d{2})?\.\d{3})").unwrap();
    let tag_re = Regex::new(r"<[^>]+>").unwrap();

    let mut i = 0;
    while i < lines.len() {
        let line = lines[i].trim();
        if let Some(cap) = time_re.captures(line) {
            let mut start_t = cap[1].replace('.', ",");
            let mut end_t = cap[2].replace('.', ",");

            if start_t.len() <= 9 {
                start_t = format!("00:{start_t}");
            }
            if end_t.len() <= 9 {
                end_t = format!("00:{end_t}");
            }

            srt_lines.push(counter.to_string());
            counter += 1;
            srt_lines.push(format!("{start_t} --> {end_t}"));

            i += 1;
            while i < lines.len() && !lines[i].trim().is_empty() {
                let clean = tag_re.replace_all(lines[i].trim(), "").to_string();
                if !clean.is_empty() {
                    srt_lines.push(clean);
                }
                i += 1;
            }
            srt_lines.push(String::new());
        }
        i += 1;
    }

    srt_lines.join("\n")
}
