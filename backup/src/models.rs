use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MediaItem {
    pub title: String,
    pub url: String,
    pub slug: String,
    pub rating: String,
    #[serde(rename = "type")]
    pub media_type: String, // "Movie" or "TV Series"
    pub poster: String,
    pub year: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SeriesDetails {
    pub title: String,
    pub slug: String,
    pub year: String,
    pub synopsis: Option<String>,
    pub poster: Option<String>,
    pub seasons: Vec<SeasonInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MovieDetails {
    pub id: String,
    pub title: String,
    pub slug: String,
    pub year: String,
    pub synopsis: Option<String>,
    pub poster: Option<String>,
    pub runtime: Option<u32>,
    pub quality: Option<String>,
    pub genres: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SeasonInfo {
    pub season_num: u32,
    pub episodes: Vec<EpisodeInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EpisodeInfo {
    pub season_num: u32,
    pub episode_num: u32,
    pub title: String,
    pub media_id: String,
    pub slug: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubtitleTrack {
    pub lang: String,
    pub url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StreamSources {
    pub title: String,
    pub m3u8_url: String,
    pub subtitles: Vec<SubtitleTrack>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum DownloadStatus {
    Queued,
    Extracting,
    Downloading,
    Paused,
    Completed,
    Failed,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DownloadTask {
    pub id: String,
    pub title: String,
    pub media_type: String, // "movie" or "series"
    #[serde(default)]
    pub year: Option<String>,
    pub season_num: Option<u32>,
    pub episode_num: Option<u32>,
    #[serde(default)]
    pub page_url: Option<String>,
    #[serde(default)]
    pub media_id: Option<String>,
    pub m3u8_url: String,
    pub subtitle_url: Option<String>,
    #[serde(default)]
    pub sub_lang: Option<String>,
    pub output_dir: String,
    pub file_name: String,
    pub status: DownloadStatus,
    pub progress: f32, // 0.0 - 100.0
    pub speed: String,
    pub eta: String,
    pub error_msg: Option<String>,
    #[serde(default)]
    pub logs: Vec<String>,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDownloadRequest {
    pub page_url: String,
    pub media_type: String,
    pub media_id: Option<String>,
    pub title: String,
    #[serde(default)]
    pub year: Option<String>,
    pub season_num: Option<u32>,
    pub episode_num: Option<u32>,
    pub sub_lang: Option<String>,
    #[serde(default)]
    pub custom_output_dir: Option<String>,
}

fn default_ask_download_location() -> bool {
    true
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppConfig {
    #[serde(default)]
    pub download_dir: String,
    #[serde(default)]
    pub movies_dir: String,
    #[serde(default)]
    pub series_dir: String,
    pub base_url: String,
    pub max_concurrent_downloads: usize,
    pub default_sub_lang: String,
    #[serde(default = "default_ask_download_location")]
    pub ask_download_location: bool,
}

impl Default for AppConfig {
    fn default() -> Self {
        let base_download = dirs::download_dir()
            .unwrap_or_else(|| std::path::PathBuf::from("./downloads"));
        let movies_dir = base_download.join("Movies").to_string_lossy().to_string();
        let series_dir = base_download.join("TV Series").to_string_lossy().to_string();

        Self {
            download_dir: base_download.to_string_lossy().to_string(),
            movies_dir,
            series_dir,
            base_url: "https://z2.idlixku.com".to_string(),
            max_concurrent_downloads: 1,
            default_sub_lang: "Indonesian".to_string(),
            ask_download_location: true,
        }
    }
}

impl AppConfig {
    pub fn validate(&self) -> Result<(), String> {
        if self.movies_dir.trim().is_empty() {
            return Err("Movies directory path cannot be empty".to_string());
        }
        if self.series_dir.trim().is_empty() {
            return Err("TV Series directory path cannot be empty".to_string());
        }
        if self.base_url.trim().is_empty() {
            return Err("Base URL cannot be empty".to_string());
        }
        let parsed_url = url::Url::parse(&self.base_url)
            .map_err(|_| "Base URL must be a valid URL (e.g. https://z2.idlixku.com)".to_string())?;
        if parsed_url.scheme() != "http" && parsed_url.scheme() != "https" {
            return Err("Base URL must use http or https protocol".to_string());
        }
        if self.max_concurrent_downloads < 1 || self.max_concurrent_downloads > 10 {
            return Err("Max concurrent downloads must be between 1 and 10".to_string());
        }

        for (label, dir_str) in [("Movies", &self.movies_dir), ("TV Series", &self.series_dir)] {
            let s = dir_str.trim();
            #[cfg(windows)]
            {
                let without_drive = if s.len() >= 2 && s.chars().nth(1) == Some(':') {
                    &s[2..]
                } else {
                    s
                };
                if without_drive.contains(['<', '>', '"', '|', '?', '*', ':']) {
                    return Err(format!("{} folder path contains invalid characters (< > \" | ? * :)", label));
                }
            }

            let path = std::path::Path::new(s);
            let abs_path = if path.is_absolute() {
                path.to_path_buf()
            } else {
                std::env::current_dir().unwrap_or_default().join(path)
            };

            if let Err(e) = std::fs::create_dir_all(&abs_path) {
                return Err(format!("Cannot create/access {} folder at '{}': {}", label, dir_str, e));
            }
        }

        Ok(())
    }

    fn config_file_path() -> std::path::PathBuf {
        if let Ok(exe) = std::env::current_exe() {
            if let Some(parent) = exe.parent() {
                let p = parent.join("config.json");
                if p.exists() {
                    return p;
                }
            }
        }
        std::path::PathBuf::from("config.json")
    }

    pub fn load() -> Self {
        let p = Self::config_file_path();
        let content = std::fs::read_to_string(&p).or_else(|_| std::fs::read_to_string("config.json"));

        if let Ok(content) = content {
            if let Ok(mut cfg) = serde_json::from_str::<AppConfig>(&content) {
                let def = Self::default();
                if cfg.movies_dir.is_empty() {
                    cfg.movies_dir = if !cfg.download_dir.is_empty() {
                        std::path::PathBuf::from(&cfg.download_dir)
                            .join("Movies")
                            .to_string_lossy()
                            .to_string()
                    } else {
                        def.movies_dir
                    };
                }
                if cfg.series_dir.is_empty() {
                    cfg.series_dir = if !cfg.download_dir.is_empty() {
                        std::path::PathBuf::from(&cfg.download_dir)
                            .join("TV Series")
                            .to_string_lossy()
                            .to_string()
                    } else {
                        def.series_dir
                    };
                }
                return cfg;
            }
        }
        Self::default()
    }

    pub fn save(&self) -> Result<(), std::io::Error> {
        let json = serde_json::to_string_pretty(self)
            .map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e))?;
        let p = Self::config_file_path();
        std::fs::write(&p, &json)?;
        let _ = std::fs::write("config.json", &json);
        Ok(())
    }
}
