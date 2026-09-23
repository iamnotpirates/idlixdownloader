use scraper::{Html, Selector};
use serde_json::Value;

use crate::http_client::CurlClient;
use crate::models::MediaItem;

pub struct Scraper {
    pub base_url: String,
    pub client: CurlClient,
}

impl Scraper {
    pub fn new(base_url: Option<String>) -> Self {
        let base = base_url.unwrap_or_else(|| "https://z2.idlixku.com".to_string());
        Self {
            base_url: base,
            client: CurlClient::new(),
        }
    }

    /// Search content via IDLIX JSON API endpoint
    pub async fn search_content(&self, query: &str) -> Result<Vec<MediaItem>, String> {
        let q_clean = query.trim();
        if q_clean.is_empty() {
            return Ok(Vec::new());
        }

        let api_url = format!("{}/api/search?q={}", self.base_url, urlencoding::encode(q_clean));
        let referer = format!("{}/", self.base_url);

        let body = self
            .client
            .get(&api_url, Some(&referer), true)
            .await
            .map_err(|e| format!("Search request failed: {e}"))?;

        let data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Failed to parse search JSON: {e} (body snippet: {})", &body.chars().take(200).collect::<String>()))?;

        let results = data["results"]
            .as_array()
            .ok_or_else(|| "Missing results array in response".to_string())?;

        let mut items = Vec::new();
        for item in results {
            let title = item["title"]
                .as_str()
                .or_else(|| item["name"].as_str())
                .unwrap_or("Unknown")
                .to_string();

            let slug = item["slug"].as_str().unwrap_or_default().to_string();
            let content_type = item["contentType"].as_str().unwrap_or("");

            let is_tv = content_type == "tv_series"
                || content_type == "series"
                || content_type == "tvshows"
                || slug.contains("series/")
                || slug.contains("tvshows/");

            let media_type = if is_tv { "TV Series".to_string() } else { "Movie".to_string() };
            let url = if is_tv {
                format!("{}/series/{}", self.base_url, slug)
            } else {
                format!("{}/movie/{}", self.base_url, slug)
            };

            let rel_date = item["releaseDate"]
                .as_str()
                .or_else(|| item["firstAirDate"].as_str())
                .unwrap_or("");
            let year = if rel_date.len() >= 4 {
                Some(rel_date[0..4].to_string())
            } else {
                None
            };

            let rating = if let Some(vote) = item["voteAverage"].as_f64() {
                format!("{:.1}", vote)
            } else {
                "N/A".to_string()
            };

            let raw_poster = item["posterPath"].as_str().or_else(|| item["poster"].as_str()).unwrap_or("");
            let poster = if raw_poster.is_empty() {
                String::new()
            } else if raw_poster.starts_with("http") {
                raw_poster.to_string()
            } else {
                format!("https://image.tmdb.org/t/p/w500{}", raw_poster)
            };

            items.push(MediaItem {
                title,
                url,
                slug,
                rating,
                media_type,
                poster,
                year,
            });
        }

        Ok(items)
    }

    /// Fetches Featured / Popular items from home page JSON-LD schema & resolves metadata
    pub async fn fetch_featured(&self) -> Result<Vec<MediaItem>, String> {
        let html = match self.client.get(&self.base_url, None, false).await {
            Ok(body) => body,
            Err(_) => return self.search_content("2024").await,
        };

        let mut featured_targets = Vec::new();

        // 1. Parse JSON-LD structured data from IDLIX Next.js Homepage
        {
            let document = Html::parse_document(&html);
            if let Ok(script_selector) = Selector::parse("script[type='application/ld+json']") {
                for script_el in document.select(&script_selector) {
                    let text = script_el.text().collect::<Vec<_>>().join("");
                    if let Ok(json_val) = serde_json::from_str::<Value>(&text) {
                        if let Some(arr) = json_val.as_array() {
                            for item in arr {
                                if item["@type"] == "ItemList" {
                                    if let Some(elements) = item["itemListElement"].as_array() {
                                        for el in elements {
                                            let name = el["name"].as_str().unwrap_or_default().to_string();
                                            let url = el["url"].as_str().unwrap_or_default().to_string();
                                            if !name.is_empty() && !url.is_empty() {
                                                let slug = url.trim_matches('/').split('/').last().unwrap_or("").to_string();
                                                let is_tv = url.contains("/series/");
                                                featured_targets.push((name, url, slug, is_tv));
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        // 2. Resolve metadata (Posters, ratings, release date) for all featured titles concurrently
        if !featured_targets.is_empty() {
            let mut join_handles = Vec::new();
            for (name, url, slug, is_tv) in featured_targets {
                let client = self.client.clone();
                let base_url = self.base_url.clone();
                join_handles.push(tokio::spawn(async move {
                    let scraper = Scraper { base_url, client };
                    if let Ok(search_res) = scraper.search_content(&name).await {
                        if let Some(found) = search_res.into_iter().find(|r| r.slug == slug || r.title.eq_ignore_ascii_case(&name)) {
                            return found;
                        }
                    }
                    MediaItem {
                        title: name,
                        url,
                        slug,
                        rating: "Featured".to_string(),
                        media_type: if is_tv { "TV Series".to_string() } else { "Movie".to_string() },
                        poster: String::new(),
                        year: None,
                    }
                }));
            }

            let mut resolved_items = Vec::new();
            for handle in join_handles {
                if let Ok(item) = handle.await {
                    resolved_items.push(item);
                }
            }

            if !resolved_items.is_empty() {
                return Ok(resolved_items);
            }
        }

        self.search_content("2024").await
    }
}
