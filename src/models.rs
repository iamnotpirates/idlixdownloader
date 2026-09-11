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
            max_concurrent_downloads: 2,
            default_sub_lang: "Indonesian".to_string(),
        }
    }
}

impl AppConfig {
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
