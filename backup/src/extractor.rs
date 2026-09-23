use serde_json::Value;
use std::time::Duration;
use tokio::time::sleep;

use crate::http_client::CurlClient;
use crate::models::{EpisodeInfo, MovieDetails, SeasonInfo, SeriesDetails, StreamSources, SubtitleTrack};

pub struct IdlixClient {
    pub base_url: String,
    pub client: CurlClient,
}

impl IdlixClient {
    pub fn new(base_url: Option<String>) -> Self {
        let base = base_url.unwrap_or_else(|| "https://z2.idlixku.com".to_string());
        Self {
            base_url: base,
            client: CurlClient::new(),
        }
    }

    /// Fetches series metadata, seasons, and episodes from IDLIX API.
    pub async fn fetch_series_details(&self, slug: &str) -> Result<SeriesDetails, String> {
        let clean_slug = slug.trim_matches('/').split('/').last().unwrap_or(slug);
        let endpoint = format!("{}/api/series/{}", self.base_url, clean_slug);
        let referer = format!("{}/series/{}", self.base_url, clean_slug);

        let body = self
            .client
            .get(&endpoint, Some(&referer), true)
            .await
            .map_err(|e| format!("Network request failed: {e}"))?;

        let data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Failed to parse JSON: {e} (snippet: {})", body.chars().take(200).collect::<String>()))?;

        let title = data["title"]
            .as_str()
            .unwrap_or(clean_slug)
            .to_string();
        let year = data["firstAirDate"]
            .as_str()
            .or_else(|| data["year"].as_str())
            .map(|y| if y.len() >= 4 { &y[0..4] } else { y })
            .unwrap_or("N/A")
            .to_string();
        let synopsis = data["overview"].as_str().map(|s| s.to_string());
        let poster = data["posterPath"].as_str().or_else(|| data["poster"].as_str()).map(|p| {
            if p.starts_with("http") {
                p.to_string()
            } else {
                format!("https://image.tmdb.org/t/p/w500{}", p)
            }
        });

        let mut seasons = Vec::new();
        if let Some(seasons_array) = data["seasons"].as_array() {
            for s in seasons_array {
                let season_num = s["seasonNumber"]
                    .as_u64()
                    .or_else(|| s["season"].as_u64())
                    .unwrap_or(1) as u32;

                let mut episodes = Vec::new();
                if let Some(eps_array) = s["episodes"].as_array() {
                    for ep in eps_array {
                        let ep_num = ep["episodeNumber"]
                            .as_u64()
                            .or_else(|| ep["episode"].as_u64())
                            .unwrap_or(1) as u32;
                        let ep_title = ep["name"]
                            .as_str()
                            .or_else(|| ep["title"].as_str())
                            .unwrap_or(&format!("Episode {ep_num}"))
                            .to_string();
                        let media_id = ep["id"]
                            .as_str()
                            .map(|s| s.to_string())
                            .or_else(|| ep["id"].as_i64().map(|n| n.to_string()))
                            .unwrap_or_default();
                        let ep_slug = ep["slug"].as_str().unwrap_or_default().to_string();

                        episodes.push(EpisodeInfo {
                            season_num,
                            episode_num: ep_num,
                            title: ep_title,
                            media_id,
                            slug: ep_slug,
                        });
                    }
                } else {
                    // Fetch season episodes explicitly if not present in main JSON
                    let season_endpoint = format!(
                        "{}/api/series/{}/season/{}",
                        self.base_url, clean_slug, season_num
                    );
                    if let Ok(s_body) = self.client.get(&season_endpoint, Some(&referer), true).await {
                        if let Ok(s_data) = serde_json::from_str::<Value>(&s_body) {
                            if let Some(eps_array) = s_data["season"]["episodes"].as_array() {
                                for ep in eps_array {
                                    let ep_num = ep["episodeNumber"]
                                        .as_u64()
                                        .or_else(|| ep["episode"].as_u64())
                                        .unwrap_or(1) as u32;
                                    let ep_title = ep["name"]
                                        .as_str()
                                        .or_else(|| ep["title"].as_str())
                                        .unwrap_or(&format!("Episode {ep_num}"))
                                        .to_string();
                                    let media_id = ep["id"]
                                        .as_str()
                                        .map(|s| s.to_string())
                                        .or_else(|| ep["id"].as_i64().map(|n| n.to_string()))
                                        .unwrap_or_default();
                                    let ep_slug = ep["slug"].as_str().unwrap_or_default().to_string();

                                    episodes.push(EpisodeInfo {
                                        season_num,
                                        episode_num: ep_num,
                                        title: ep_title,
                                        media_id,
                                        slug: ep_slug,
                                    });
                                }
                            }
                        }
                    }
                }

                seasons.push(SeasonInfo {
                    season_num,
                    episodes,
                });
            }
        }

        Ok(SeriesDetails {
            title,
            slug: clean_slug.to_string(),
            year,
            synopsis,
            poster,
            seasons,
        })
    }

    /// Fetches movie metadata from IDLIX API.
    pub async fn fetch_movie_details(&self, slug: &str) -> Result<MovieDetails, String> {
        let clean_slug = slug.trim_matches('/').split('/').last().unwrap_or(slug);
        let endpoint = format!("{}/api/movies/{}", self.base_url, clean_slug);
        let referer = format!("{}/movie/{}", self.base_url, clean_slug);

        let body = self
            .client
            .get(&endpoint, Some(&referer), true)
            .await
            .map_err(|e| format!("Movie metadata request failed: {e}"))?;

        let data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Failed to parse movie JSON: {e} (snippet: {})", body.chars().take(200).collect::<String>()))?;

        let id = data["id"]
            .as_str()
            .map(|s| s.to_string())
            .or_else(|| data["id"].as_i64().map(|n| n.to_string()))
            .unwrap_or_default();
        let title = data["title"]
            .as_str()
            .unwrap_or(clean_slug)
            .to_string();
        let year = data["releaseDate"]
            .as_str()
            .or_else(|| data["year"].as_str())
            .map(|y| if y.len() >= 4 { &y[0..4] } else { y })
            .unwrap_or("N/A")
            .to_string();
        let synopsis = data["overview"].as_str().map(|s| s.to_string());
        let poster = data["posterPath"].as_str().or_else(|| data["poster"].as_str()).map(|p| {
            if p.starts_with("http") {
                p.to_string()
            } else {
                format!("https://image.tmdb.org/t/p/w500{}", p)
            }
        });
        let runtime = data["runtime"].as_u64().map(|r| r as u32);
        let quality = data["quality"].as_str().map(|q| q.to_string());

        let mut genres = Vec::new();
        if let Some(genres_arr) = data["genres"].as_array() {
            for g in genres_arr {
                if let Some(name) = g["name"].as_str() {
                    genres.push(name.to_string());
                }
            }
        }

        Ok(MovieDetails {
            id,
            title,
            slug: clean_slug.to_string(),
            year,
            synopsis,
            poster,
            runtime,
            quality,
            genres,
        })
    }

    /// Extracts video stream (.m3u8) & subtitles for a Movie or Episode.
    pub async fn extract_stream(
        &self,
        content_type: &str, // "movie" or "series"
        slug_or_id: &str,
        page_url: &str,
    ) -> Result<StreamSources, String> {
        let (media_id, title) = if content_type == "movie" {
            let slug_from_page = page_url.trim_matches('/').split('/').last().unwrap_or("");
            let clean_slug = if !slug_from_page.is_empty() {
                slug_from_page
            } else {
                slug_or_id.trim_matches('/').split('/').last().unwrap_or(slug_or_id)
            };

            let meta_endpoint = format!("{}/api/movies/{}", self.base_url, clean_slug);
            let body = self
                .client
                .get(&meta_endpoint, Some(page_url), true)
                .await
                .map_err(|e| format!("Failed to get movie metadata: {e}"))?;

            let meta: Value = serde_json::from_str(&body)
                .map_err(|e| format!("Invalid metadata JSON: {e}"))?;
            let id = meta["id"]
                .as_str()
                .map(|s| s.to_string())
                .or_else(|| meta["id"].as_i64().map(|n| n.to_string()))
                .ok_or_else(|| "Media ID not found in movie metadata".to_string())?;
            let title = meta["title"].as_str().unwrap_or(clean_slug).to_string();
            (id, title)
        } else {
            // For series episode, slug_or_id is the episode media_id
            let clean_slug = slug_or_id.trim_matches('/').split('/').last().unwrap_or(slug_or_id);
            (slug_or_id.to_string(), clean_slug.to_string())
        };

        // 0. Ensure session device cookies ('did') are present
        if !page_url.is_empty() {
            let _ = self.client.get(page_url, None, false).await;
        } else {
            let _ = self.client.get(&self.base_url, None, false).await;
        }

        // 1. Play-info gate token
        let info_content_type = if content_type == "movie" { "movie" } else { "episode" };
        let play_info_url = format!(
            "{}/api/watch/play-info/{}/{}",
            self.base_url,
            info_content_type,
            media_id
        );

        let body = self
            .client
            .get(&play_info_url, Some(page_url), true)
            .await
            .map_err(|e| format!("Failed to request play-info: {e}"))?;

        let info_data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Invalid play-info JSON: {e} (snippet: {})", body.chars().take(200).collect::<String>()))?;
        let gate_token = info_data["gateToken"]
            .as_str()
            .ok_or_else(|| format!("Missing gateToken in response: {}", body.chars().take(200).collect::<String>()))?;

        let unlock_at = info_data["unlockAt"].as_f64().unwrap_or(0.0) / 1000.0;
        let server_now = info_data["serverNow"].as_f64().unwrap_or(0.0) / 1000.0;
        let diff = unlock_at - server_now;
        let wait_sec = if diff > 0.0 { diff.min(30.0) + 1.0 } else { 0.5 };

        if wait_sec > 0.0 {
            println!("[Extractor] Waiting {:.1}s for gate countdown timer...", wait_sec);
            sleep(Duration::from_secs_f64(wait_sec)).await;
        }

        // 2. Claim session
        println!("[Extractor] Claiming playback session...");
        let claim_url = format!("{}/api/watch/session/claim", self.base_url);
        let claim_payload = serde_json::json!({ "gateToken": gate_token });
        let body = self
            .client
            .post_json(&claim_url, Some(page_url), &claim_payload)
            .await
            .map_err(|e| format!("Failed to claim session: {e}"))?;

        let claim_data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Invalid claim JSON: {e} (snippet: {})", body.chars().take(200).collect::<String>()))?;

        if let Some(err) = claim_data["error"].as_str() {
            return Err(format!("Session claim rejected: {err}"));
        }

        let claim_token = claim_data["claim"]
            .as_str()
            .ok_or_else(|| format!("Missing claim token in session response: {body}"))?;
        let redeem_url = claim_data["redeemUrl"]
            .as_str()
            .ok_or_else(|| format!("Missing redeemUrl in session response: {body}"))?;

        // 3. Redeem master playlist URL and subtitles
        println!("[Extractor] Redeeming stream sources from {redeem_url}...");
        let redeem_payload = serde_json::json!({ "claim": claim_token });
        let body = self
            .client
            .post_json(redeem_url, Some(page_url), &redeem_payload)
            .await
            .map_err(|e| format!("Failed to redeem master playlist: {e}"))?;

        let final_data: Value = serde_json::from_str(&body)
            .map_err(|e| format!("Invalid redeem JSON: {e} (snippet: {})", body.chars().take(200).collect::<String>()))?;

        if let Some(err) = final_data["error"].as_str() {
            return Err(format!("Master playlist redeem rejected: {err}"));
        }

        let m3u8_url = final_data["url"]
            .as_str()
            .ok_or_else(|| "No master m3u8 playlist found in stream response".to_string())?
            .to_string();

        let mut subtitles = Vec::new();
        if let Some(subs_array) = final_data["subtitles"].as_array() {
            for s in subs_array {
                let lang = s["label"]
                    .as_str()
                    .or_else(|| s["lang"].as_str())
                    .unwrap_or("Subtitle")
                    .to_string();
                let sub_url = s["path"].as_str().unwrap_or_default().to_string();
                if !sub_url.is_empty() {
                    subtitles.push(SubtitleTrack {
                        lang,
                        url: sub_url,
                    });
                }
            }
        }

        Ok(StreamSources {
            title,
            m3u8_url,
            subtitles,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_extract_moana() {
        let client = IdlixClient::new(None);
        let res = client.extract_stream("movie", "moana-2016", "https://z2.idlixku.com/movie/moana-2016").await;
        println!("Test extract result: {:?}", res);
        assert!(res.is_ok());
        let sources = res.unwrap();
        assert!(!sources.m3u8_url.is_empty());
        println!("Extracted M3U8 URL: {}", sources.m3u8_url);
    }
}
