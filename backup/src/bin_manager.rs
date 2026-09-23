use serde_json::Value;
use std::fs::{self, File};
use std::io::{self, Cursor};
use std::path::PathBuf;
use zip::ZipArchive;

#[derive(Debug, Clone)]
pub struct BinPaths {
    pub ffmpeg: PathBuf,
    pub n_m3u8dl_re: PathBuf,
}

pub struct BinManager {
    bin_dir: PathBuf,
}

impl BinManager {
    pub fn new() -> Self {
        let bin_dir = std::env::current_exe()
            .ok()
            .and_then(|p| p.parent().map(|p| p.to_path_buf()))
            .unwrap_or_else(|| PathBuf::from("."))
            .join("bin");

        Self { bin_dir }
    }

    /// Checks if a command binary is in system PATH
    fn find_in_path(name: &str) -> Option<PathBuf> {
        let exe_name = if cfg!(windows) && !name.ends_with(".exe") {
            format!("{name}.exe")
        } else {
            name.to_string()
        };

        if let Ok(path_var) = std::env::var("PATH") {
            for dir in std::env::split_paths(&path_var) {
                let full = dir.join(&exe_name);
                if full.is_file() {
                    return Some(full);
                }
            }
        }
        None
    }

    /// Checks if binary exists in local ./bin/ or PATH
    pub fn get_n_m3u8dl_re_path(&self) -> Option<PathBuf> {
        let local = self.bin_dir.join(if cfg!(windows) { "N_m3u8DL-RE.exe" } else { "N_m3u8DL-RE" });
        if local.is_file() {
            return Some(local);
        }
        Self::find_in_path("N_m3u8DL-RE")
    }

    /// Checks if ffmpeg exists in local ./bin/ or PATH
    pub fn get_ffmpeg_path(&self) -> Option<PathBuf> {
        let local = self.bin_dir.join(if cfg!(windows) { "ffmpeg.exe" } else { "ffmpeg" });
        if local.is_file() {
            return Some(local);
        }
        Self::find_in_path("ffmpeg")
    }

    /// Ensures both required binaries are available. Downloads them automatically if missing.
    pub async fn ensure_binaries(&self) -> Result<BinPaths, String> {
        fs::create_dir_all(&self.bin_dir)
            .map_err(|e| format!("Failed to create bin directory: {e}"))?;

        let n_path = match self.get_n_m3u8dl_re_path() {
            Some(p) => p,
            None => {
                println!("[BinManager] N_m3u8DL-RE not found. Auto-downloading...");
                self.download_n_m3u8dl_re().await?
            }
        };

        let ffmpeg_path = match self.get_ffmpeg_path() {
            Some(p) => p,
            None => {
                println!("[BinManager] ffmpeg not found. Auto-downloading...");
                self.download_ffmpeg().await?
            }
        };

        Ok(BinPaths {
            ffmpeg: ffmpeg_path,
            n_m3u8dl_re: n_path,
        })
    }

    async fn download_n_m3u8dl_re(&self) -> Result<PathBuf, String> {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(120))
            .user_agent("IDLIXDownloader-Rust/1.0")
            .build()
            .map_err(|e| format!("HTTP error: {e}"))?;

        // 1. Fetch latest release URL from GitHub API
        let mut download_url = None;
        if let Ok(resp) = client
            .get("https://api.github.com/repos/nilaoda/N_m3u8DL-RE/releases/latest")
            .send()
            .await
        {
            if let Ok(rel) = resp.json::<Value>().await {
                if let Some(assets) = rel["assets"].as_array() {
                    for a in assets {
                        let name = a["name"].as_str().unwrap_or("");
                        let u = a["browser_download_url"].as_str().unwrap_or("");
                        if cfg!(windows) && name.contains("win-x64") && name.ends_with(".zip") {
                            download_url = Some(u.to_string());
                            break;
                        }
                    }
                }
            }
        }

        let url = download_url.unwrap_or_else(|| {
            "https://github.com/nilaoda/N_m3u8DL-RE/releases/download/v0.6.0-beta/N_m3u8DL-RE_v0.6.0-beta_win-x64_20260629.zip".to_string()
        });

        let resp = client
            .get(&url)
            .send()
            .await
            .map_err(|e| format!("Failed to download N_m3u8DL-RE: {e}"))?;

        if !resp.status().is_success() {
            return Err(format!("Download failed with status {}", resp.status()));
        }

        let bytes = resp.bytes().await.map_err(|e| format!("Read bytes error: {e}"))?;
        let target_exe = self.bin_dir.join(if cfg!(windows) { "N_m3u8DL-RE.exe" } else { "N_m3u8DL-RE" });

        // Extract zip
        let mut archive = ZipArchive::new(Cursor::new(bytes))
            .map_err(|e| format!("Invalid zip archive: {e}"))?;

        for i in 0..archive.len() {
            let mut file = archive.by_index(i).map_err(|e| format!("Zip entry error: {e}"))?;
            let name = file.name().to_string();
            if name.ends_with("N_m3u8DL-RE.exe") || name == "N_m3u8DL-RE" {
                let mut out = File::create(&target_exe).map_err(|e| format!("Failed to create exe: {e}"))?;
                io::copy(&mut file, &mut out).map_err(|e| format!("Copy failed: {e}"))?;
                break;
            }
        }

        if target_exe.is_file() {
            println!("[BinManager] N_m3u8DL-RE installed to {:?}", target_exe);
            Ok(target_exe)
        } else {
            Err("Failed to extract N_m3u8DL-RE binary from zip".to_string())
        }
    }

    async fn download_ffmpeg(&self) -> Result<PathBuf, String> {
        let target_exe = self.bin_dir.join(if cfg!(windows) { "ffmpeg.exe" } else { "ffmpeg" });

        let url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip";

        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(180))
            .user_agent("IDLIXDownloader-Rust/1.0")
            .build()
            .map_err(|e| format!("HTTP error: {e}"))?;

        let resp = client
            .get(url)
            .send()
            .await
            .map_err(|e| format!("Failed to download ffmpeg: {e}"))?;

        if !resp.status().is_success() {
            return Err(format!("Download ffmpeg failed with status {}", resp.status()));
        }

        let bytes = resp.bytes().await.map_err(|e| format!("Read ffmpeg bytes error: {e}"))?;

        let mut archive = ZipArchive::new(Cursor::new(bytes))
            .map_err(|e| format!("Invalid ffmpeg zip: {e}"))?;

        for i in 0..archive.len() {
            let mut file = archive.by_index(i).map_err(|e| format!("Zip entry error: {e}"))?;
            let name = file.name().to_string();
            if name.ends_with("bin/ffmpeg.exe") || name.ends_with("ffmpeg.exe") {
                let mut out = File::create(&target_exe).map_err(|e| format!("Failed to create ffmpeg.exe: {e}"))?;
                io::copy(&mut file, &mut out).map_err(|e| format!("Copy ffmpeg failed: {e}"))?;
                break;
            }
        }

        if target_exe.is_file() {
            println!("[BinManager] ffmpeg installed to {:?}", target_exe);
            Ok(target_exe)
        } else {
            Err("Failed to extract ffmpeg binary from zip".to_string())
        }
    }
}
