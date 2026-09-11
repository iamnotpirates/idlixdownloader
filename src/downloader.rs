use regex::Regex;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::io::AsyncReadExt;
use tokio::process::Command;
use tokio::sync::{broadcast, Mutex, RwLock};

use crate::bin_manager::BinPaths;
use crate::extractor::IdlixClient;
use crate::models::{AppConfig, DownloadStatus, DownloadTask};

pub struct DownloadManager {
    pub tasks: Arc<RwLock<Vec<DownloadTask>>>,
    pub tx: broadcast::Sender<DownloadTask>,
    pub bin_paths: BinPaths,
    pub config: Arc<RwLock<AppConfig>>,
    pub extractor: Arc<IdlixClient>,
    active_workers: Arc<Mutex<usize>>,
}

impl DownloadManager {
    pub fn new(bin_paths: BinPaths, config: AppConfig, extractor: Arc<IdlixClient>) -> Self {
        let (tx, _) = broadcast::channel(100);
        Self {
            tasks: Arc::new(RwLock::new(Vec::new())),
            tx,
            bin_paths,
            config: Arc::new(RwLock::new(config)),
            extractor,
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
            created_at: now,
        };

        {
            let mut tasks = self.tasks.write().await;
            tasks.push(task.clone());
        }

        let _ = self.tx.send(task.clone());

        // Trigger worker pool check
        self.spawn_queue_worker();

        Ok(task)
    }

    pub fn spawn_queue_worker(&self) {
        let tasks_arc = Arc::clone(&self.tasks);
        let active_workers = Arc::clone(&self.active_workers);
        let config_arc = Arc::clone(&self.config);
        let tx = self.tx.clone();
        let bin_paths = self.bin_paths.clone();
        let extractor = Arc::clone(&self.extractor);

        Self::spawn_worker_loop(tasks_arc, active_workers, config_arc, tx, bin_paths, extractor);
    }

    fn spawn_worker_loop(
        tasks_arc: Arc<RwLock<Vec<DownloadTask>>>,
        active_workers: Arc<Mutex<usize>>,
        config_arc: Arc<RwLock<AppConfig>>,
        tx: broadcast::Sender<DownloadTask>,
        bin_paths: BinPaths,
        extractor: Arc<IdlixClient>,
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

                tokio::spawn(async move {
                    Self::execute_task(
                        task,
                        tasks_inner.clone(),
                        tx_inner.clone(),
                        bins.clone(),
                        extractor_inner.clone(),
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
    ) {
        // 1. If m3u8_url is empty, extract stream sources now
        if task.m3u8_url.is_empty() {
            {
                let mut tasks = tasks_arc.write().await;
                if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                    t.status = DownloadStatus::Extracting;
                    t.eta = "Unlocking stream (~15s)...".to_string();
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

            match extractor.extract_stream(&task.media_type, slug_or_id, &page_url).await {
                Ok(sources) => {
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

                    {
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                            t.m3u8_url = sources.m3u8_url.clone();
                            t.subtitle_url = chosen_sub;
                            t.status = DownloadStatus::Downloading;
                            t.eta = "Starting download...".to_string();
                            task = t.clone();
                        }
                    }
                    let _ = tx.send(task.clone());
                }
                Err(e) => {
                    {
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                            t.status = DownloadStatus::Failed;
                            t.error_msg = Some(format!("Extraction error: {e}"));
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
            let _ = download_and_convert_vtt_to_srt(sub_url, &srt_path).await;
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

        let mut child = match cmd.spawn() {
            Ok(c) => c,
            Err(e) => {
                task.status = DownloadStatus::Failed;
                task.error_msg = Some(format!("Failed to spawn downloader: {e}"));
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

        if let Some(out) = stdout {
            let tasks_clone = Arc::clone(&tasks_arc);
            let tx_clone = tx.clone();
            let task_id = task.id.clone();
            tokio::spawn(process_stream_output(out, task_id, tasks_clone, tx_clone));
        }

        if let Some(err) = stderr {
            let tasks_clone = Arc::clone(&tasks_arc);
            let tx_clone = tx.clone();
            let task_id = task.id.clone();
            tokio::spawn(process_stream_output(err, task_id, tasks_clone, tx_clone));
        }

        let status = child.wait().await;
        let success = status.map(|s| s.success()).unwrap_or(false);

        {
            let mut tasks = tasks_arc.write().await;
            if let Some(t) = tasks.iter_mut().find(|t| t.id == task.id) {
                if success {
                    t.status = DownloadStatus::Completed;
                    t.progress = 100.0;
                    t.speed = "Done".to_string();
                    t.eta = "00:00:00".to_string();
                } else {
                    t.status = DownloadStatus::Failed;
                    t.error_msg = Some("Downloader process exited with error".to_string());
                }
                task = t.clone();
            }
        }

        let _ = tx.send(task);
    }
}

async fn process_stream_output<R: tokio::io::AsyncRead + Unpin>(
    mut reader: R,
    task_id: String,
    tasks_arc: Arc<RwLock<Vec<DownloadTask>>>,
    tx: broadcast::Sender<DownloadTask>,
) {
    let progress_regex = Regex::new(r"(\d+(?:\.\d+)?)%").unwrap();
    let seg_regex = Regex::new(r"(\d+)\s*/\s*(\d+)").unwrap();
    let speed_regex = Regex::new(r"%\s*(?:[\d\.]+[kKMmGg]?[bB]/[\d\.]+[kKMmGg]?[bB])?\s*(\d+(?:\.\d+)?\s*(?:[kKMmGg][iI]?[bB](?:/s|ps)|[bB]ps))").unwrap();
    let fallback_speed_regex = Regex::new(r"(?:^|\s)(\d+(?:\.\d+)?\s*(?:[kKMmGg][iI]?[bB]/s))").unwrap();
    let eta_regex = Regex::new(r"(\d{1,2}:\d{2}:\d{2})").unwrap();
    let ansi_regex = Regex::new(r"\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])").unwrap();

    let mut buf = [0u8; 4096];
    let mut line_buf = Vec::new();
    let mut last_emit = std::time::Instant::now();

    while let Ok(n) = reader.read(&mut buf).await {
        if n == 0 {
            break;
        }
        for &b in &buf[..n] {
            if b == b'\r' || b == b'\n' {
                if !line_buf.is_empty() {
                    let raw_str = String::from_utf8_lossy(&line_buf);
                    let clean = ansi_regex.replace_all(&raw_str, "").to_string();
                    line_buf.clear();

                    let trimmed = clean.trim();
                    if trimmed.is_empty() {
                        continue;
                    }

                    // Filter out standard log lines so their timestamps are never parsed as ETA
                    if trimmed.contains("INFO :") || trimmed.contains("WARN :") || trimmed.contains("ERROR :") || trimmed.contains("DEBUG :") {
                        continue;
                    }

                    let mut updated = false;
                    let mut current_task = None;

                    {
                        let mut tasks = tasks_arc.write().await;
                        if let Some(t) = tasks.iter_mut().find(|t| t.id == task_id) {
                            if let Some(cap) = progress_regex.captures(trimmed) {
                                if let Ok(val) = cap[1].parse::<f32>() {
                                    if (val - t.progress).abs() >= 0.05 || val >= 100.0 {
                                        t.progress = val;
                                        t.status = DownloadStatus::Downloading;
                                        updated = true;
                                    }
                                }
                            } else if let Some(cap) = seg_regex.captures(trimmed) {
                                if let (Ok(curr), Ok(total)) = (cap[1].parse::<f32>(), cap[2].parse::<f32>()) {
                                    if total > 0.0 && curr <= total {
                                        let val = ((curr / total) * 100.0).min(100.0);
                                        if (val - t.progress).abs() >= 0.05 {
                                            t.progress = val;
                                            t.status = DownloadStatus::Downloading;
                                            updated = true;
                                        }
                                    }
                                }
                            }

                            if let Some(cap) = speed_regex.captures(trimmed).or_else(|| fallback_speed_regex.captures(trimmed)) {
                                let mut s = cap[1].trim().to_string();
                                if s.ends_with("MBps") {
                                    s = s.replace("MBps", " MB/s");
                                } else if s.ends_with("KBps") {
                                    s = s.replace("KBps", " KB/s");
                                } else if s.ends_with("GBps") {
                                    s = s.replace("GBps", " GB/s");
                                }
                                if t.speed != s {
                                    t.speed = s;
                                    updated = true;
                                }
                            }

                            // Only parse ETA from actual progress lines containing % or ---
                            if (trimmed.contains('%') || trimmed.contains("---")) && !trimmed.contains("--:--:--") {
                                if let Some(m) = eta_regex.find_iter(trimmed).last() {
                                    let eta_str = m.as_str().trim().to_string();
                                    if t.eta != eta_str {
                                        t.eta = eta_str;
                                        updated = true;
                                    }
                                }
                            }

                            if updated {
                                current_task = Some(t.clone());
                            }
                        }
                    }

                    if let Some(t) = current_task {
                        if last_emit.elapsed().as_millis() >= 100 || t.progress >= 100.0 {
                            let _ = tx.send(t);
                            last_emit = std::time::Instant::now();
                        }
                    }
                }
            } else {
                line_buf.push(b);
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
            created_at: 0,
        };

        let tasks_arc = Arc::new(RwLock::new(vec![task]));
        let simulated_output = b"Vid 1920x1080 | 6221 Kbps ------------------------------ 1/64 1.56% 5.81MB/371.85MB5.81MBps00:00:32\rVid 1920x1080 | 6221 Kbps ------------------------------ 50/64 78.12% 423.30MB/564.40MB11.04MBps00:00:09\rVid 1920x1080 | 6221 Kbps ------------------------------64/64 100.00%478.94MB - 00:00:00\n";

        process_stream_output(
            &simulated_output[..],
            "task_1".to_string(),
            tasks_arc.clone(),
            tx,
        )
        .await;

        let final_task = {
            let tasks = tasks_arc.read().await;
            tasks[0].clone()
        };

        assert_eq!(final_task.progress, 100.0);
        assert_eq!(final_task.speed, "11.04 MB/s");
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
