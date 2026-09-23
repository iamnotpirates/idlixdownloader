export interface MediaItem {
  title: string
  url: string
  slug: string
  rating: string
  type: 'Movie' | 'TV Series'
  poster: string
  year?: string
}

export interface EpisodeInfo {
  season_num: number
  episode_num: number
  title: string
  media_id: string
  slug: string
}

export interface SeasonInfo {
  season_num: number
  episodes: EpisodeInfo[]
}

export interface SeriesDetails {
  title: string
  slug: string
  year: string
  synopsis?: string
  poster?: string
  seasons: SeasonInfo[]
}

export interface MovieDetails {
  id: string
  title: string
  slug: string
  year: string
  synopsis?: string
  poster?: string
  runtime?: number
  quality?: string
  genres: string[]
}

export interface SubtitleTrack {
  lang: string
  url: string
}

export interface StreamSources {
  title: string
  m3u8_url: string
  subtitles: SubtitleTrack[]
}

export interface DownloadTask {
  id: string
  title: string
  media_type: string
  year?: string
  season_num?: number
  episode_num?: number
  page_url?: string
  media_id?: string
  m3u8_url: string
  subtitle_url?: string
  sub_lang?: string
  output_dir: string
  file_name: string
  status: 'queued' | 'extracting' | 'downloading' | 'paused' | 'completed' | 'failed' | 'cancelled'
  progress: number
  speed: string
  eta: string
  error_msg?: string
  logs?: string[]
  created_at: number
}

export interface AppConfig {
  download_dir: string
  movies_dir: string
  series_dir: string
  base_url: string
  max_concurrent_downloads: number
  default_sub_lang: string
  ask_download_location?: boolean
}

export interface CreateDownloadPayload {
  page_url: string
  media_type: string
  media_id?: string
  title: string
  year?: string
  season_num?: number
  episode_num?: number
  sub_lang?: string
  custom_output_dir?: string
}
