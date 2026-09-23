package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/iamnotpirates/idlixdownloader/internal/config"
	"github.com/iamnotpirates/idlixdownloader/internal/db"
	"github.com/iamnotpirates/idlixdownloader/internal/downloader"
	"github.com/iamnotpirates/idlixdownloader/internal/extractor"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
	"github.com/iamnotpirates/idlixdownloader/internal/scraper"
	"github.com/iamnotpirates/idlixdownloader/internal/sse"
	"github.com/iamnotpirates/idlixdownloader/web"
)

type Server struct {
	router     *chi.Mux
	db         *db.DB
	scraper    *scraper.Scraper
	extractor  *extractor.IdlixClient
	downloader *downloader.Manager
	sseHub     *sse.Hub
	cfg        models.AppConfig
}

func New(
	database *db.DB,
	sc *scraper.Scraper,
	ext *extractor.IdlixClient,
	dl *downloader.Manager,
	hub *sse.Hub,
) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		db:         database,
		scraper:    sc,
		extractor:  ext,
		downloader: dl,
		sseHub:     hub,
		cfg:        config.LoadConfig(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(corsMiddleware)

	// Web UI
	s.router.Get("/", web.Handler())

	// SSE Events
	s.router.Get("/api/events", s.sseHub.ServeHTTP)

	// API Routes
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/catalog", s.handleCatalog)
		r.Get("/search", s.handleSearch)
		r.Get("/movie/{slug}", s.handleMovieDetails)
		r.Get("/series/{slug}", s.handleSeriesDetails)

		r.Get("/downloads", s.handleGetDownloads)
		r.Post("/download", s.handleCreateDownload)
		r.Post("/download/season", s.handleCreateSeasonDownload)
		r.Post("/download/cancel", s.handleCancelDownload)
		r.Delete("/download/{id}", s.handleDeleteDownload)
		r.Get("/download/{id}/logs", s.handleGetLogs)

		r.Get("/config", s.handleGetConfig)
		r.Post("/config", s.handleSaveConfig)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	items, err := s.scraper.FetchFeatured()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	items, err := s.scraper.SearchContent(q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleMovieDetails(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	details, err := s.extractor.FetchMovieDetails(slug)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, details)
}

func (s *Server) handleSeriesDetails(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	details, err := s.extractor.FetchSeriesDetails(slug)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, details)
}

func (s *Server) handleGetDownloads(w http.ResponseWriter, r *http.Request) {
	tasks := s.downloader.GetAllTasks()
	respondJSON(w, http.StatusOK, tasks)
}

type DownloadRequest struct {
	Title       string  `json:"title"`
	MediaType   string  `json:"media_type"` // "movie" or "series"
	Year        *string `json:"year,omitempty"`
	SeasonNum   *int    `json:"season_num,omitempty"`
	EpisodeNum  *int    `json:"episode_num,omitempty"`
	EpisodeTitle string `json:"episode_title,omitempty"`
	PageURL     *string `json:"page_url,omitempty"`
	MediaID     *string `json:"media_id,omitempty"`
	OutputDir   string  `json:"output_dir,omitempty"`
}

func (s *Server) handleCreateDownload(w http.ResponseWriter, r *http.Request) {
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cfg := config.LoadConfig()
	taskID := uuid.New().String()[:8]
	createdAt := time.Now().Unix()

	cleanTitle := sanitizeFilename(req.Title)
	yearStr := "N/A"
	if req.Year != nil && *req.Year != "" {
		yearStr = *req.Year
	}

	var outputDir, fileName string

	if req.MediaType == "series" {
		sNum := 1
		if req.SeasonNum != nil {
			sNum = *req.SeasonNum
		}
		eNum := 1
		if req.EpisodeNum != nil {
			eNum = *req.EpisodeNum
		}

		epTitle := fmt.Sprintf("Episode %d", eNum)
		if req.EpisodeTitle != "" {
			epTitle = sanitizeFilename(req.EpisodeTitle)
		}

		baseSeriesDir := cfg.SeriesDir
		if req.OutputDir != "" {
			baseSeriesDir = req.OutputDir
		}

		// Jellyfin standard: {SeriesDir}/{Title} ({Year})/Season {N}/
		seriesFolder := fmt.Sprintf("%s (%s)", cleanTitle, yearStr)
		seasonFolder := fmt.Sprintf("Season %d", sNum)
		outputDir = filepath.Join(baseSeriesDir, seriesFolder, seasonFolder)

		// File: {Title} - S{N:02d}E{E:02d} - {EpisodeTitle}
		fileName = fmt.Sprintf("%s - S%02dE%02d - %s", cleanTitle, sNum, eNum, epTitle)
	} else {
		baseMoviesDir := cfg.MoviesDir
		if req.OutputDir != "" {
			baseMoviesDir = req.OutputDir
		}

		// Jellyfin standard: {MoviesDir}/{Title} ({Year})/
		movieFolder := fmt.Sprintf("%s (%s)", cleanTitle, yearStr)
		outputDir = filepath.Join(baseMoviesDir, movieFolder)

		// File: {Title} ({Year})
		fileName = fmt.Sprintf("%s (%s)", cleanTitle, yearStr)
	}

	task := &models.DownloadTask{
		ID:         taskID,
		Title:      req.Title,
		MediaType:  req.MediaType,
		Year:       req.Year,
		SeasonNum:  req.SeasonNum,
		EpisodeNum: req.EpisodeNum,
		PageURL:    req.PageURL,
		MediaID:    req.MediaID,
		OutputDir:  outputDir,
		FileName:   fileName,
		Status:     models.StatusQueued,
		Progress:   0.0,
		Speed:      "",
		ETA:        "Queued",
		CreatedAt:  createdAt,
	}

	if err := s.downloader.Enqueue(task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

type SeasonDownloadRequest struct {
	Title     string               `json:"title"`
	Year      *string              `json:"year,omitempty"`
	SeasonNum int                  `json:"season_num"`
	PageURL   *string              `json:"page_url,omitempty"`
	OutputDir string               `json:"output_dir,omitempty"`
	Episodes  []models.EpisodeInfo `json:"episodes"`
}

func (s *Server) handleCreateSeasonDownload(w http.ResponseWriter, r *http.Request) {
	var req SeasonDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cfg := config.LoadConfig()
	cleanTitle := sanitizeFilename(req.Title)
	yearStr := "N/A"
	if req.Year != nil && *req.Year != "" {
		yearStr = *req.Year
	}

	baseSeriesDir := cfg.SeriesDir
	if req.OutputDir != "" {
		baseSeriesDir = req.OutputDir
	}

	seriesFolder := fmt.Sprintf("%s (%s)", cleanTitle, yearStr)
	seasonFolder := fmt.Sprintf("Season %d", req.SeasonNum)
	outputDir := filepath.Join(baseSeriesDir, seriesFolder, seasonFolder)

	var createdTasks []*models.DownloadTask

	for _, ep := range req.Episodes {
		taskID := uuid.New().String()[:8]
		createdAt := time.Now().Unix()
		sNum := req.SeasonNum
		eNum := ep.EpisodeNum
		epTitle := sanitizeFilename(ep.Title)
		if epTitle == "" {
			epTitle = fmt.Sprintf("Episode %d", eNum)
		}

		fileName := fmt.Sprintf("%s - S%02dE%02d - %s", cleanTitle, sNum, eNum, epTitle)
		mediaID := ep.MediaID
		if mediaID == "" {
			mediaID = ep.Slug
		}

		task := &models.DownloadTask{
			ID:         taskID,
			Title:      req.Title,
			MediaType:  "series",
			Year:       req.Year,
			SeasonNum:  &sNum,
			EpisodeNum: &eNum,
			PageURL:    req.PageURL,
			MediaID:    &mediaID,
			OutputDir:  outputDir,
			FileName:   fileName,
			Status:     models.StatusQueued,
			Progress:   0.0,
			Speed:      "",
			ETA:        "Queued",
			CreatedAt:  createdAt,
		}

		_ = s.downloader.Enqueue(task)
		createdTasks = append(createdTasks, task)
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"queued_count": len(createdTasks),
		"tasks":        createdTasks,
	})
}

func (s *Server) handleCancelDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_ = s.downloader.Cancel(body.ID)
	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = s.downloader.Delete(id)
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logs, err := s.db.GetTaskLogs(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, logs)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	respondJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var cfg models.AppConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := config.SaveConfig(cfg); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update live scraper & extractor base URL if changed
	s.scraper.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	s.extractor.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	respondJSON(w, http.StatusOK, cfg)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func sanitizeFilename(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	clean := name
	for _, ch := range invalid {
		clean = strings.ReplaceAll(clean, ch, " ")
	}
	return strings.TrimSpace(strings.Join(strings.Fields(clean), " "))
}
