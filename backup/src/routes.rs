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
            req.custom_output_dir,
        )
        .await
        .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e))?;

    Ok(Json(task))
}

#[cfg(windows)]
struct ForegroundWindowWrapper(isize);

#[cfg(windows)]
impl raw_window_handle::HasWindowHandle for ForegroundWindowWrapper {
    fn window_handle(&self) -> Result<raw_window_handle::WindowHandle<'_>, raw_window_handle::HandleError> {
        let non_zero = std::num::NonZeroIsize::new(self.0)
            .ok_or(raw_window_handle::HandleError::Unavailable)?;
        let handle = raw_window_handle::Win32WindowHandle::new(non_zero);
        unsafe { Ok(raw_window_handle::WindowHandle::borrow_raw(raw_window_handle::RawWindowHandle::Win32(handle))) }
    }
}

#[cfg(windows)]
impl raw_window_handle::HasDisplayHandle for ForegroundWindowWrapper {
    fn display_handle(&self) -> Result<raw_window_handle::DisplayHandle<'_>, raw_window_handle::HandleError> {
        let handle = raw_window_handle::WindowsDisplayHandle::new();
        unsafe { Ok(raw_window_handle::DisplayHandle::borrow_raw(raw_window_handle::RawDisplayHandle::Windows(handle))) }
    }
}

pub async fn pick_folder_handler() -> Result<Json<serde_json::Value>, AppError> {
    let chosen = tokio::task::spawn_blocking(|| {
        let mut dialog = rfd::FileDialog::new().set_title("Pilih Folder Penyimpanan");

        #[cfg(windows)]
        {
            unsafe {
                let hwnd = windows_sys::Win32::UI::WindowsAndMessaging::GetForegroundWindow();
                if !hwnd.is_null() {
                    let wrapper = ForegroundWindowWrapper(hwnd as isize);
                    dialog = dialog.set_parent(&wrapper);
                }
            }
        }

        dialog.pick_folder()
    })
    .await
    .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    match chosen {
        Some(path) => Ok(Json(serde_json::json!({
            "status": "ok",
            "path": path.to_string_lossy().to_string()
        }))),
        None => Ok(Json(serde_json::json!({
            "status": "cancelled",
            "path": null
        }))),
    }
}

#[derive(Deserialize)]
pub struct OpenFolderParams {
    pub target: Option<String>,
    pub path: Option<String>,
}

pub async fn open_folder_handler(
    State(state): State<AppState>,
    Query(params): Query<OpenFolderParams>,
) -> Result<Json<serde_json::Value>, AppError> {
    let target_path = if let Some(ref p) = params.path {
        p.clone()
    } else if let Some(ref target) = params.target {
        let conf = state.downloader.config.read().await;
        match target.as_str() {
            "movies" => conf.movies_dir.clone(),
            "series" => conf.series_dir.clone(),
            _ => {
                return Err(AppError(
                    StatusCode::BAD_REQUEST,
                    "Invalid folder target. Must be 'movies' or 'series'".to_string(),
                ))
            }
        }
    } else {
        return Err(AppError(
            StatusCode::BAD_REQUEST,
            "Missing 'target' or 'path' query parameter".to_string(),
        ));
    };

    crate::tray::open_folder_path(&target_path);
    Ok(Json(serde_json::json!({ "status": "ok", "path": target_path })))
}

#[derive(Deserialize)]
pub struct TaskActionParams {
    pub id: String,
}

pub async fn retry_download(
    State(state): State<AppState>,
    Query(params): Query<TaskActionParams>,
) -> Result<Json<DownloadTask>, AppError> {
    let task = state
        .downloader
        .retry_task(&params.id)
        .await
        .map_err(|e| AppError(StatusCode::BAD_REQUEST, e))?;
    Ok(Json(task))
}

pub async fn delete_download(
    State(state): State<AppState>,
    Query(params): Query<TaskActionParams>,
) -> Result<Json<serde_json::Value>, AppError> {
    state
        .downloader
        .delete_task(&params.id)
        .await
        .map_err(|e| AppError(StatusCode::BAD_REQUEST, e))?;
    Ok(Json(serde_json::json!({ "status": "ok", "id": params.id })))
}

#[derive(Deserialize)]
pub struct TaskLogParams {
    pub id: String,
}

pub async fn get_task_log(
    State(state): State<AppState>,
    Query(params): Query<TaskLogParams>,
) -> Result<Json<serde_json::Value>, AppError> {
    let db = state.downloader.db.clone();
    let task_id = params.id.clone();
    let logs = tokio::task::spawn_blocking(move || {
        db.get_task_logs(&task_id).unwrap_or_default()
    })
    .await
    .map_err(|e| AppError(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let content = if logs.is_empty() {
        "No logs recorded in database for this task yet.".to_string()
    } else {
        logs.join("\n")
    };

    Ok(Json(serde_json::json!({
        "task_id": params.id,
        "log": content
    })))
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
