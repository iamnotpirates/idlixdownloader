package models

type MediaItem struct {
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	Slug      string  `json:"slug"`
	Rating    string  `json:"rating"`
	MediaType string  `json:"media_type"` // "Movie" or "TV Series"
	Poster    string  `json:"poster"`
	Year      *string `json:"year,omitempty"`
}

type EpisodeInfo struct {
	SeasonNum  int    `json:"season_num"`
	EpisodeNum int    `json:"episode_num"`
	Title      string `json:"title"`
	MediaID    string `json:"media_id"`
	Slug       string `json:"slug"`
}

type SeasonInfo struct {
	SeasonNum int           `json:"season_num"`
	Episodes  []EpisodeInfo `json:"episodes"`
}

type SeriesDetails struct {
	Title    string       `json:"title"`
	Slug     string       `json:"slug"`
	Year     string       `json:"year"`
	Synopsis *string      `json:"synopsis,omitempty"`
	Poster   *string      `json:"poster,omitempty"`
	Seasons  []SeasonInfo `json:"seasons"`
}

type MovieDetails struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Slug     string   `json:"slug"`
	Year     string   `json:"year"`
	Synopsis *string  `json:"synopsis,omitempty"`
	Poster   *string  `json:"poster,omitempty"`
	Runtime  *int     `json:"runtime,omitempty"`
	Quality  *string  `json:"quality,omitempty"`
	Genres   []string `json:"genres"`
}

type SubtitleTrack struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

type StreamSources struct {
	Title     string          `json:"title"`
	M3U8URL   string          `json:"m3u8_url"`
	Subtitles []SubtitleTrack `json:"subtitles"`
}

type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusExtracting  DownloadStatus = "extracting"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
)

type DownloadTask struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	MediaType   string         `json:"media_type"` // "movie" or "series"
	Year        *string        `json:"year,omitempty"`
	SeasonNum   *int           `json:"season_num,omitempty"`
	EpisodeNum  *int           `json:"episode_num,omitempty"`
	PageURL     *string        `json:"page_url,omitempty"`
	MediaID     *string        `json:"media_id,omitempty"`
	M3U8URL     string         `json:"m3u8_url"`
	SubtitleURL *string        `json:"subtitle_url,omitempty"`
	SubLang     *string        `json:"sub_lang,omitempty"`
	OutputDir   string         `json:"output_dir"`
	FileName    string         `json:"file_name"`
	Status      DownloadStatus `json:"status"`
	Progress    float64        `json:"progress"`
	Speed       string         `json:"speed"`
	ETA         string         `json:"eta"`
	ErrorMsg    *string        `json:"error_msg,omitempty"`
	CreatedAt   int64          `json:"created_at"`
}

type TaskLog struct {
	ID        int64  `json:"id"`
	TaskID    string `json:"task_id"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

type AppConfig struct {
	MoviesDir           string `json:"movies_dir"`
	SeriesDir           string `json:"series_dir"`
	SubLang             string `json:"sub_lang"`
	ConfirmDownload     bool   `json:"confirm_download"`
	BaseURL             string `json:"base_url"`
	MaxConcurrentTasks  int    `json:"max_concurrent_tasks"`
}
