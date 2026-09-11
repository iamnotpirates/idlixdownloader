import type { AppConfig, DownloadTask, MediaItem, MovieDetails, SeriesDetails, StreamSources } from './types'

const API_BASE = '/api'

export async function fetchCatalog(): Promise<MediaItem[]> {
  const res = await fetch(`${API_BASE}/catalog`)
  if (!res.ok) throw new Error(`Failed to load catalog: ${res.statusText}`)
  return res.json()
}

export async function searchMedia(query: string): Promise<MediaItem[]> {
  const res = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}`)
  if (!res.ok) throw new Error(`Search failed: ${res.statusText}`)
  return res.json()
}

export async function fetchSeriesDetails(slug: string): Promise<SeriesDetails> {
  const res = await fetch(`${API_BASE}/series?slug=${encodeURIComponent(slug)}`)
  if (!res.ok) throw new Error(`Failed to fetch series: ${res.statusText}`)
  return res.json()
}

export async function fetchMovieDetails(slug: string): Promise<MovieDetails> {
  const res = await fetch(`${API_BASE}/movies?slug=${encodeURIComponent(slug)}`)
  if (!res.ok) throw new Error(`Failed to fetch movie: ${res.statusText}`)
  return res.json()
}

export async function extractStream(
  contentType: 'movie' | 'series',
  slugOrId: string,
  pageUrl: string
): Promise<StreamSources> {
  const res = await fetch(`${API_BASE}/extract`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      content_type: contentType,
      slug_or_id: slugOrId,
      page_url: pageUrl,
    }),
  })
  if (!res.ok) {
    const errText = await res.text()
    throw new Error(errText || 'Failed to extract stream')
  }
  return res.json()
}

export async function startDownload(params: {
  page_url: string
  media_type: 'movie' | 'series'
  media_id?: string
  title: string
  year?: string
  season_num?: number
  episode_num?: number
  sub_lang?: string
}): Promise<DownloadTask> {
  const res = await fetch(`${API_BASE}/downloads`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  })
  if (!res.ok) {
    const err = await res.text()
    throw new Error(err || 'Failed to start download')
  }
  return res.json()
}

export async function fetchDownloads(): Promise<DownloadTask[]> {
  const res = await fetch(`${API_BASE}/downloads`)
  if (!res.ok) throw new Error('Failed to fetch downloads')
  return res.json()
}

export async function fetchSettings(): Promise<AppConfig> {
  const res = await fetch(`${API_BASE}/settings`)
  if (!res.ok) throw new Error('Failed to load settings')
  return res.json()
}

export async function saveSettings(config: AppConfig): Promise<AppConfig> {
  const res = await fetch(`${API_BASE}/settings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
  if (!res.ok) {
    const errText = await res.text()
    throw new Error(errText || 'Failed to save settings')
  }
  return res.json()
}

export async function openFolder(target: 'movies' | 'series'): Promise<void> {
  const res = await fetch(`${API_BASE}/open-folder?target=${target}`, {
    method: 'POST',
  })
  if (!res.ok) {
    const errText = await res.text()
    throw new Error(errText || 'Failed to open folder')
  }
}

export function createDownloadSocket(onMessage: (tasks: DownloadTask | DownloadTask[]) => void): () => void {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws`
  let ws: WebSocket | null = null
  let reconnectTimer: number | null = null

  function connect() {
    ws = new WebSocket(wsUrl)
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        onMessage(data)
      } catch (e) {
        console.error('WS Parse Error', e)
      }
    }
    ws.onclose = () => {
      reconnectTimer = window.setTimeout(connect, 2000)
    }
  }

  connect()

  return () => {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws) ws.close()
  }
}
