#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bin_manager;
mod downloader;
mod extractor;
mod http_client;
mod models;
mod routes;
mod scraper;
mod tray;

use axum::{
    body::Body,
    http::{header, HeaderValue, Response, StatusCode, Uri},
    response::IntoResponse,
    routing::{get, post},
    Router,
};
use bin_manager::BinManager;
use downloader::DownloadManager;
use extractor::IdlixClient;
use models::AppConfig;
use routes::{
    extract_stream_sources, get_catalog, get_downloads, get_movie_details, get_series_details,
    get_settings, open_folder_handler, search_media, start_download, update_settings, ws_handler,
    AppState,
};
use rust_embed::RustEmbed;
use std::net::SocketAddr;
use std::sync::Arc;
use tower_http::cors::{Any, CorsLayer};

#[derive(RustEmbed)]
#[folder = "frontend/dist"]
struct FrontendAssets;

async fn static_handler(uri: Uri) -> impl IntoResponse {
    let mut path = uri.path().trim_start_matches('/').to_string();

    if path.is_empty() {
        path = "index.html".to_string();
    }

    match FrontendAssets::get(&path) {
        Some(content) => {
            let mime = mime_guess::from_path(&path).first_or_octet_stream();
            Response::builder()
                .status(StatusCode::OK)
                .header(header::CONTENT_TYPE, HeaderValue::from_str(mime.as_ref()).unwrap())
                .body(Body::from(content.data))
                .unwrap()
        }
        None => {
            // SPA fallback: Return index.html for client-side routing
            match FrontendAssets::get("index.html") {
                Some(content) => Response::builder()
                    .status(StatusCode::OK)
                    .header(header::CONTENT_TYPE, HeaderValue::from_static("text/html"))
                    .body(Body::from(content.data))
                    .unwrap(),
                None => Response::builder()
                    .status(StatusCode::NOT_FOUND)
                    .body(Body::from("Frontend build files not found. Run 'bun run build' inside frontend/"))
                    .unwrap(),
            }
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=======================================================");
    println!("🏴‍☠️ IDLIX Downloader (Rust + Svelte)");
    println!("=======================================================");

    // 1. Check / auto-download dependencies
    let bin_mgr = BinManager::new();
    println!("[Init] Checking required binaries (ffmpeg, N_m3u8DL-RE)...");
    let bin_paths = bin_mgr.ensure_binaries().await.unwrap_or_else(|e| {
        eprintln!("[Warning] Binary auto-download issue: {e}");
        bin_manager::BinPaths {
            ffmpeg: std::path::PathBuf::from("ffmpeg"),
            n_m3u8dl_re: std::path::PathBuf::from("N_m3u8DL-RE"),
        }
    });

    println!("[Init] FFmpeg: {:?}", bin_paths.ffmpeg);
    println!("[Init] N_m3u8DL-RE: {:?}", bin_paths.n_m3u8dl_re);

    let config = AppConfig::load();
    let extractor = Arc::new(IdlixClient::new(Some(config.base_url.clone())));
    let downloader = Arc::new(DownloadManager::new(bin_paths, config, Arc::clone(&extractor)));

    let state = AppState {
        downloader,
    };

    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    let app = Router::<AppState>::new()
        .route("/api/catalog", get(get_catalog))
        .route("/api/search", get(search_media))
        .route("/api/movies", get(get_movie_details))
        .route("/api/series", get(get_series_details))
        .route("/api/extract", post(extract_stream_sources))
        .route("/api/downloads", get(get_downloads).post(start_download))
        .route("/api/settings", get(get_settings).post(update_settings))
        .route("/api/open-folder", post(open_folder_handler))
        .route("/ws", get(ws_handler))
        .fallback(static_handler)
        .layer(cors)
        .with_state(state);

    let addr = SocketAddr::from(([0, 0, 0, 0], 8989));
    println!("🚀 Server running on all interfaces: http://0.0.0.0:8989 (port 8989)");
    println!("💡 Open http://localhost:8989 or http://<your-ip>:8989 in your browser.");

    let listener = tokio::net::TcpListener::bind(addr).await?;
    tokio::spawn(async move {
        if let Err(e) = axum::serve(listener, app).await {
            eprintln!("[Server Error] {e}");
        }
    });

    let server_url = "http://localhost:8989".to_string();
    tray::run_tray_loop(server_url)?;

    Ok(())
}
