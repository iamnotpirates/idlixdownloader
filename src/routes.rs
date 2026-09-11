use axum::{
    extract::{
        ws::{Message, WebSocket, WebSocketUpgrade},
        Query, State,
    },
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::Deserialize;
use std::sync::Arc;

use crate::downloader::DownloadManager;
use crate::extractor::IdlixClient;
use crate::models::{
    AppConfig, CreateDownloadRequest, DownloadTask, MediaItem, MovieDetails, SeriesDetails,
    StreamSources,
};
use crate::scraper::Scraper;

#[derive(Clone)]
pub struct AppState {
    pub downloader: Arc<DownloadManager>,
}

#[derive(Debug)]
pub struct AppError(pub StatusCode, pub String);

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        (self.0, self.1).into_response()
    }
}

#[derive(Deserialize)]
pub struct SearchParams {
    pub q: String,
}

#[derive(Deserialize)]
pub struct SeriesParams {
    pub slug: String,
}

#[derive(Deserialize)]
pub struct MovieParams {
    pub slug: String,
}

#[derive(Deserialize)]
pub struct ExtractRequest {
    pub content_type: String,
    pub slug_or_id: String,
    pub page_url: String,
}

pub async fn get_catalog(
    State(state): State<AppState>,
) -> Result<Json<Vec<MediaItem>>, AppError> {
    let conf = state.downloader.config.read().await;
    let scraper = Scraper::new(Some(conf.base_url.clone()));
    let items = scraper
        .fetch_featured()
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;
    Ok(Json(items))
}

pub async fn search_media(
    State(state): State<AppState>,
    Query(params): Query<SearchParams>,
) -> Result<Json<Vec<MediaItem>>, AppError> {
    let conf = state.downloader.config.read().await;
    let scraper = Scraper::new(Some(conf.base_url.clone()));
    let items = scraper
        .search_content(&params.q)
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;
    Ok(Json(items))
}

pub async fn get_series_details(
    State(state): State<AppState>,
    Query(params): Query<SeriesParams>,
) -> Result<Json<SeriesDetails>, AppError> {
    let conf = state.downloader.config.read().await;
    let extractor = IdlixClient::new(Some(conf.base_url.clone()));
    let details = extractor
        .fetch_series_details(&params.slug)
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;
    Ok(Json(details))
}

pub async fn get_movie_details(
    State(state): State<AppState>,
    Query(params): Query<MovieParams>,
) -> Result<Json<MovieDetails>, AppError> {
    let conf = state.downloader.config.read().await;
    let extractor = IdlixClient::new(Some(conf.base_url.clone()));
    let details = extractor
        .fetch_movie_details(&params.slug)
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;
    Ok(Json(details))
}

pub async fn extract_stream_sources(
    State(state): State<AppState>,
    Json(payload): Json<ExtractRequest>,
) -> Result<Json<StreamSources>, AppError> {
    let conf = state.downloader.config.read().await;
    let extractor = IdlixClient::new(Some(conf.base_url.clone()));
    let sources = extractor
        .extract_stream(&payload.content_type, &payload.slug_or_id, &payload.page_url)
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;
    Ok(Json(sources))
}

pub async fn get_downloads(State(state): State<AppState>) -> Json<Vec<DownloadTask>> {
    let tasks = state.downloader.tasks.read().await;
    Json(tasks.clone())
}

pub async fn start_download(
    State(state): State<AppState>,
    Json(req): Json<CreateDownloadRequest>,
) -> Result<Json<DownloadTask>, AppError> {
    let task = state
        .downloader
        .add_task(
            req.title,
            req.media_type,
            req.year,
            req.season_num,
            req.episode_num,
            Some(req.page_url),
            req.media_id,
            String::new(), // extracted in background queue worker
            None,
            req.sub_lang,
            None,
        )
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;

    Ok(Json(task))
}

#[derive(Deserialize)]
pub struct OpenFolderParams {
    pub target: String,
}

pub async fn open_folder_handler(
    State(state): State<AppState>,
    Query(params): Query<OpenFolderParams>,
) -> Result<Json<serde_json::Value>, AppError> {
    let conf = state.downloader.config.read().await;
    let target_dir = match params.target.as_str() {
        "movies" => &conf.movies_dir,
        "series" => &conf.series_dir,
        _ => {
            return Err(AppError(
                StatusCode::BAD_REQUEST,
                "Invalid folder target. Must be 'movies' or 'series'".to_string(),
            ))
        }
    };

    crate::tray::open_folder_path(target_dir);
    Ok(Json(serde_json::json!({ "status": "ok", "path": target_dir })))
}

pub async fn get_settings(State(state): State<AppState>) -> Json<AppConfig> {
    let conf = state.downloader.config.read().await;
    Json(conf.clone())
}

pub async fn update_settings(
    State(state): State<AppState>,
    Json(new_conf): Json<AppConfig>,
) -> Result<Json<AppConfig>, AppError> {
    if let Err(err_msg) = new_conf.validate() {
        return Err(AppError(StatusCode::BAD_REQUEST, err_msg));
    }

    let mut conf = state.downloader.config.write().await;
    *conf = new_conf.clone();
    if let Err(e) = conf.save() {
        return Err(AppError(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("Failed to save config: {e}"),
        ));
    }
    Ok(Json(conf.clone()))
}

pub async fn ws_handler(ws: WebSocketUpgrade, State(state): State<AppState>) -> impl IntoResponse {
    ws.on_upgrade(|socket| handle_socket(socket, state))
}

async fn handle_socket(mut socket: WebSocket, state: AppState) {
    let mut rx = state.downloader.subscribe();

    // Send initial snapshot of all tasks
    {
        let tasks = state.downloader.tasks.read().await;
        if let Ok(json) = serde_json::to_string(&*tasks) {
            let _ = socket.send(Message::Text(json.into())).await;
        }
    }

    // Stream updates
    while let Ok(task) = rx.recv().await {
        if let Ok(json) = serde_json::to_string(&task) {
            if socket.send(Message::Text(json.into())).await.is_err() {
                break;
            }
        }
    }
}
