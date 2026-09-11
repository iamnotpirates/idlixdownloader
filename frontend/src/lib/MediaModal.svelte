<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import type { AppConfig, MediaItem, MovieDetails, SeriesDetails } from './types'
  import { fetchMovieDetails, fetchSeriesDetails, fetchSettings, pickFolder, startDownload } from './api'
  import {
    X,
    Download,
    Star,
    Tv,
    Film,
    CheckCircle2,
    AlertCircle,
    Loader2,
    Clock,
    Sparkles,
    FolderDown,
    FolderSearch,
    FolderCheck,
    Subtitles,
  } from 'lucide-svelte'

  export let item: MediaItem
  export let onClose: () => void
  export let onDownloadStarted: (title: string) => void

  interface PendingDownload {
    type: 'movie' | 'series'
    title: string
    year?: string
    mediaId?: string
    seasonNum?: number
    episodeNum?: number
    defaultDir: string
    customDir: string | null
    useCustomDir: boolean
    subLang: string
  }

  let seriesDetails: SeriesDetails | null = null
  let movieDetails: MovieDetails | null = null
  let appConfig: AppConfig | null = null
  let selectedSeasonIndex = 0
  let loading = false
  let errorMsg = ''
  let subtitleLang = 'Indonesian'
  let downloadingEpisodes: Record<string, boolean> = {}
  let isDownloadingMovie = false
  let successMsg = ''

  let pendingDownload: PendingDownload | null = null
  let isSubmitting = false
  let isBrowsingFolder = false

  onMount(async () => {
    // Lock background body scroll
    document.body.style.overflow = 'hidden'

    loading = true
    errorMsg = ''
    try {
      const [configRes, detailsRes] = await Promise.all([
        fetchSettings().catch(() => null),
        item.type === 'TV Series' ? fetchSeriesDetails(item.slug) : fetchMovieDetails(item.slug),
      ])

      if (configRes) {
        appConfig = configRes
        if (configRes.default_sub_lang) {
          subtitleLang = configRes.default_sub_lang
        }
      }

      if (item.type === 'TV Series') {
        seriesDetails = detailsRes as SeriesDetails
      } else {
        movieDetails = detailsRes as MovieDetails
      }
    } catch (err: any) {
      errorMsg = err.message || 'Failed to load details'
    } finally {
      loading = false
    }
  })

  onDestroy(() => {
    // Restore background body scroll
    document.body.style.overflow = ''
  })

  function sanitizeTitle(t: string): string {
    return t.replace(/[\\/:*?"<>|]/g, '').trim()
  }

  function getYear(): string {
    return movieDetails?.year || seriesDetails?.year || item.year || ''
  }

  function getFolderPreview(p: PendingDownload): string {
    const clean = sanitizeTitle(p.title)
    const yr = p.year || getYear()
    const folderTitle = yr ? `${clean} (${yr})` : clean
    const baseDir = (p.useCustomDir && p.customDir ? p.customDir : p.defaultDir) || 'Downloads'

    if (p.type === 'series') {
      const s = String(p.seasonNum || 1).padStart(2, '0')
      const e = String(p.episodeNum || 1).padStart(2, '0')
      return `${baseDir}\\${folderTitle}\\Season ${s}\\${clean} - S${s}E${e}.mp4`
    } else {
      return `${baseDir}\\${folderTitle}\\${folderTitle}.mp4`
    }
  }

  function requestDownloadMovie() {
    const yr = movieDetails?.year || item.year
    const defDir = appConfig?.movies_dir || './downloads/Movies'

    if (appConfig?.ask_download_location !== false) {
      pendingDownload = {
        type: 'movie',
        title: item.title,
        year: yr,
        defaultDir: defDir,
        customDir: null,
        useCustomDir: false,
        subLang: subtitleLang,
      }
    } else {
      executeDownload({
        page_url: item.url,
        media_type: 'movie',
        title: item.title,
        year: yr,
        sub_lang: subtitleLang,
      })
    }
  }

  function requestDownloadEpisode(mediaId: string, seasonNum: number, episodeNum: number, _epTitle: string) {
    const yr = seriesDetails?.year || item.year
    const defDir = appConfig?.series_dir || './downloads/TV Series'

    if (appConfig?.ask_download_location !== false) {
      pendingDownload = {
        type: 'series',
        title: item.title,
        year: yr,
        mediaId,
        seasonNum,
        episodeNum,
        defaultDir: defDir,
        customDir: null,
        useCustomDir: false,
        subLang: subtitleLang,
      }
    } else {
      executeDownload({
        page_url: item.url,
        media_type: 'series',
        media_id: mediaId,
        title: item.title,
        year: yr,
        season_num: seasonNum,
        episode_num: episodeNum,
        sub_lang: subtitleLang,
      })
    }
  }

  async function handleBrowseCustomFolder() {
    if (!pendingDownload) return
    isBrowsingFolder = true
    try {
      const chosen = await pickFolder()
      if (chosen) {
        pendingDownload.customDir = chosen
        pendingDownload.useCustomDir = true
      }
    } catch (err: any) {
      errorMsg = `Gagal membuka folder picker: ${err.message || err}`
    } finally {
      isBrowsingFolder = false
    }
  }

  async function handleConfirmDownload() {
    if (!pendingDownload) return

    const p = pendingDownload
    const customOut = p.useCustomDir && p.customDir ? p.customDir : undefined

    if (p.type === 'movie') {
      await executeDownload({
        page_url: item.url,
        media_type: 'movie',
        title: p.title,
        year: p.year,
        sub_lang: p.subLang,
        custom_output_dir: customOut,
      })
    } else {
      await executeDownload({
        page_url: item.url,
        media_type: 'series',
        media_id: p.mediaId,
        title: p.title,
        year: p.year,
        season_num: p.seasonNum,
        episode_num: p.episodeNum,
        sub_lang: p.subLang,
        custom_output_dir: customOut,
      })
    }

    pendingDownload = null
  }

  async function executeDownload(params: {
    page_url: string
    media_type: 'movie' | 'series'
    media_id?: string
    title: string
    year?: string
    season_num?: number
    episode_num?: number
    sub_lang?: string
    custom_output_dir?: string
  }) {
    isSubmitting = true
    errorMsg = ''

    const isMovie = params.media_type === 'movie'
    const epKey = `${params.season_num}-${params.episode_num}`

    if (isMovie) {
      isDownloadingMovie = true
    } else {
      downloadingEpisodes[epKey] = true
    }

    try {
      await startDownload(params)
      const displayTitle = isMovie
        ? params.title
        : `${params.title} - S${String(params.season_num).padStart(2, '0')}E${String(params.episode_num).padStart(2, '0')}`

      successMsg = `Antrean download ditambahkan: "${displayTitle}"`
      onDownloadStarted(displayTitle)

      if (isMovie) {
        setTimeout(() => {
          onClose()
        }, 1200)
      }
    } catch (err: any) {
      errorMsg = err.message || 'Gagal memulai download'
    } finally {
      isSubmitting = false
      if (isMovie) {
        isDownloadingMovie = false
      } else {
        downloadingEpisodes[epKey] = false
      }
    }
  }
</script>

<div
  class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-zinc-950/80 p-0 sm:p-4 backdrop-blur-md transition-all"
  on:click|self={onClose}
>
  <div class="relative flex max-h-[92vh] sm:max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-t-3xl sm:rounded-3xl border-t sm:border border-zinc-800 bg-zinc-900 shadow-2xl">
    <!-- Drag Handle indicator for mobile bottom sheet -->
    <div class="mx-auto mt-2.5 h-1 w-12 rounded-full bg-zinc-700 sm:hidden" />

    <!-- Close Button -->
    <button
      class="absolute top-3 right-3 sm:top-4 sm:right-4 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-zinc-800/80 text-zinc-400 hover:bg-zinc-700 hover:text-white active:scale-95 transition-all"
      on:click={onClose}
    >
      <X class="h-4 w-4" />
    </button>

    <!-- Header Section -->
    <div class="flex gap-3 sm:gap-4 p-4 sm:p-6 border-b border-zinc-800/60 bg-gradient-to-b from-zinc-800/40 to-transparent">
      <div class="h-28 w-20 sm:h-32 sm:w-24 shrink-0 overflow-hidden rounded-xl bg-zinc-950 shadow-md">
        <img src={movieDetails?.poster || seriesDetails?.poster || item.poster || ''} alt={item.title} class="h-full w-full object-cover" />
      </div>
      <div class="flex flex-col justify-between overflow-hidden pr-6 sm:pr-0">
        <div>
          <div class="flex flex-wrap items-center gap-1.5 sm:gap-2">
            <span class="inline-flex items-center gap-1 rounded bg-indigo-500/20 px-2 py-0.5 text-[9px] sm:text-[10px] font-semibold text-indigo-400">
              {#if item.type === 'TV Series'}
                <Tv class="h-3 w-3" />
              {:else}
                <Film class="h-3 w-3" />
              {/if}
              {item.type}
            </span>
            {#if item.rating && item.rating !== 'N/A'}
              <span class="inline-flex items-center gap-1 text-[11px] sm:text-xs font-semibold text-amber-400">
                <Star class="h-3 w-3 fill-amber-400" />
                {item.rating}
              </span>
            {/if}
            {#if movieDetails?.year || seriesDetails?.year || item.year}
              <span class="text-[11px] sm:text-xs text-zinc-400">{movieDetails?.year || seriesDetails?.year || item.year}</span>
            {/if}
            {#if movieDetails?.runtime}
              <span class="inline-flex items-center gap-1 text-[11px] sm:text-xs text-zinc-400">
                <Clock class="h-3 w-3" />
                {movieDetails.runtime}m
              </span>
            {/if}
            {#if movieDetails?.quality}
              <span class="rounded bg-emerald-500/20 px-1.5 py-0.5 text-[9px] sm:text-[10px] font-semibold text-emerald-400">
                {movieDetails.quality}
              </span>
            {/if}
          </div>
          <h2 class="mt-1 text-base sm:text-lg font-bold text-white line-clamp-1">{item.title}</h2>
          <p class="mt-1 text-[11px] sm:text-xs text-zinc-400 line-clamp-2">
            {movieDetails?.synopsis || seriesDetails?.synopsis || 'Ready to download with high-speed multi-threaded engine.'}
          </p>
          {#if movieDetails?.genres && movieDetails.genres.length > 0}
            <div class="mt-1.5 flex flex-wrap gap-1">
              {#each movieDetails.genres.slice(0, 3) as genre}
                <span class="rounded-md bg-zinc-800/80 px-1.5 py-0.5 text-[9px] sm:text-[10px] text-zinc-400">
                  {genre}
                </span>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Subtitle selector -->
        <div class="mt-2.5 flex items-center gap-2">
          <span class="text-[10px] sm:text-[11px] font-medium text-zinc-400">Sub:</span>
          <select
            bind:value={subtitleLang}
            class="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs font-medium text-zinc-200 focus:border-indigo-500 focus:outline-none"
          >
            <option value="Indonesian">Indonesian</option>
            <option value="English">English</option>
            <option value="None">None</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Body / Content -->
    <div class="flex-1 overflow-y-auto p-4 sm:p-6">
      {#if errorMsg}
        <div class="mb-4 flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/30 p-3 text-xs text-red-400">
          <AlertCircle class="h-4 w-4 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      {/if}

      {#if successMsg}
        <div class="mb-4 flex items-center gap-2 rounded-xl bg-emerald-500/10 border border-emerald-500/30 p-3 text-xs text-emerald-400">
          <CheckCircle2 class="h-4 w-4 shrink-0" />
          <span>{successMsg}</span>
        </div>
      {/if}

      {#if item.type === 'Movie'}
        {#if loading}
          <div class="flex flex-col items-center justify-center py-10">
            <Loader2 class="h-8 w-8 animate-spin text-indigo-500" />
            <p class="mt-2 text-xs text-zinc-400">Memeriksa link stream movie...</p>
          </div>
        {:else}
          <!-- Movie Stream Card -->
          <div class="flex flex-col gap-3">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-2xl border border-zinc-800/80 bg-zinc-800/40 p-4 hover:bg-zinc-800/60 transition-all">
              <div class="flex items-center gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-600/20 text-indigo-400">
                  <Film class="h-5 w-5" />
                </div>
                <div>
                  <h4 class="text-sm font-semibold text-zinc-100">{item.title}</h4>
                  <div class="mt-0.5 flex items-center gap-2 text-[11px] text-zinc-400">
                    <span>Full Movie</span>
                    <span>•</span>
                    <span class="text-emerald-400 font-medium">{movieDetails?.quality || 'Full HD / HLS'}</span>
                    {#if movieDetails?.runtime}
                      <span>•</span>
                      <span>{movieDetails.runtime}m</span>
                    {/if}
                  </div>
                </div>
              </div>

              <button
                class="flex items-center justify-center gap-2 rounded-xl bg-indigo-600 px-5 py-3 text-xs font-bold text-white shadow-lg shadow-indigo-600/25 hover:bg-indigo-500 active:scale-95 transition-all disabled:opacity-50 w-full sm:w-auto"
                on:click={requestDownloadMovie}
                disabled={isDownloadingMovie}
              >
                {#if isDownloadingMovie}
                  <Loader2 class="h-4 w-4 animate-spin" />
                  <span>Mengantrekan...</span>
                {:else}
                  <Download class="h-4 w-4" />
                  <span>Download Movie</span>
                {/if}
              </button>
            </div>

            <div class="rounded-xl border border-zinc-800/50 bg-zinc-950/40 p-3 text-[11px] text-zinc-400 flex items-start gap-2">
              <Sparkles class="h-4 w-4 text-indigo-400 shrink-0 mt-0.5" />
              <span>
                Stream akan diunduh dengan multi-threaded engine dan subtitle {subtitleLang}.
              </span>
            </div>
          </div>
        {/if}
      {:else}
        <!-- TV Series View -->
        {#if loading}
          <div class="flex flex-col items-center justify-center py-10">
            <Loader2 class="h-8 w-8 animate-spin text-indigo-500" />
            <p class="mt-2 text-xs text-zinc-400">Memuat season & episode...</p>
          </div>
        {:else if seriesDetails && seriesDetails.seasons.length > 0}
          <!-- Season Tabs (Horizontal swipeable on mobile) -->
          <div class="flex items-center gap-2 overflow-x-auto pb-2 scrollbar-none">
            {#each seriesDetails.seasons as season, idx}
              <button
                class="rounded-xl px-3.5 py-2 text-xs font-semibold whitespace-nowrap active:scale-95 transition-all {selectedSeasonIndex === idx
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'bg-zinc-800 text-zinc-400 hover:text-zinc-200'}"
                on:click={() => (selectedSeasonIndex = idx)}
              >
                Season {season.season_num}
              </button>
            {/each}
          </div>

          <!-- Episode List -->
          <div class="mt-3 flex flex-col gap-2">
            {#each seriesDetails.seasons[selectedSeasonIndex].episodes as ep}
              {@const isDownloading = downloadingEpisodes[`${ep.season_num}-${ep.episode_num}`]}
              <div class="flex items-center justify-between gap-3 rounded-xl border border-zinc-800/80 bg-zinc-800/40 p-3 hover:bg-zinc-800/80 transition-all">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-zinc-700/60 text-xs font-bold text-zinc-300">
                    {ep.episode_num}
                  </div>
                  <div class="min-w-0">
                    <h4 class="text-xs font-semibold text-zinc-100 truncate">{ep.title}</h4>
                    <p class="text-[10px] text-zinc-400">S{ep.season_num} E{ep.episode_num}</p>
                  </div>
                </div>

                <button
                  class="flex shrink-0 items-center gap-1.5 rounded-lg bg-indigo-600/20 px-3 py-2 text-xs font-semibold text-indigo-400 hover:bg-indigo-600 hover:text-white active:scale-95 transition-all disabled:opacity-50"
                  on:click={() => requestDownloadEpisode(ep.media_id, ep.season_num, ep.episode_num, ep.title)}
                  disabled={isDownloading}
                >
                  {#if isDownloading}
                    <Loader2 class="h-3.5 w-3.5 animate-spin" />
                    <span>Queuing...</span>
                  {:else}
                    <Download class="h-3.5 w-3.5" />
                    <span>Download</span>
                  {/if}
                </button>
              </div>
            {/each}
          </div>
        {:else}
          <div class="py-8 text-center text-xs text-zinc-400">
            Tidak ada episode yang ditemukan untuk serial ini.
          </div>
        {/if}
      {/if}
    </div>
  </div>

  <!-- Download Confirmation Dialog Popup (Same dark theme) -->
  {#if pendingDownload}
    <div
      class="fixed inset-0 z-60 flex items-center justify-center bg-zinc-950/85 p-3 sm:p-4 backdrop-blur-md animate-in fade-in duration-150"
      on:click|self={() => (pendingDownload = null)}
    >
      <div class="relative flex w-full max-w-lg flex-col overflow-hidden rounded-2xl sm:rounded-3xl border border-zinc-800 bg-zinc-900 p-5 sm:p-6 shadow-2xl">
        <!-- Close Button -->
        <button
          class="absolute top-4 right-4 flex h-7 w-7 items-center justify-center rounded-full bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white active:scale-95 transition-all"
          on:click={() => (pendingDownload = null)}
        >
          <X class="h-4 w-4" />
        </button>

        <!-- Header -->
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-600/20 text-indigo-400">
            <FolderDown class="h-5 w-5" />
          </div>
          <div>
            <h3 class="text-sm sm:text-base font-bold text-white">Konfirmasi Download</h3>
            <p class="text-[11px] text-zinc-400">
              {pendingDownload.type === 'movie' ? 'Film' : 'Episode Serial'}: <span class="font-medium text-zinc-200">{pendingDownload.title}</span>
            </p>
          </div>
        </div>

        <!-- Folder Selection Options -->
        <div class="mt-4 flex flex-col gap-2.5">
          <span class="text-xs font-semibold text-zinc-300">Pilih Lokasi Penyimpanan:</span>

          <!-- Option 1: Default Folder -->
          <label
            class="flex items-start gap-3 rounded-xl border p-3 cursor-pointer transition-all {
              !pendingDownload.useCustomDir
                ? 'border-indigo-500/80 bg-indigo-500/10'
                : 'border-zinc-800 bg-zinc-950/40 hover:bg-zinc-800/40'
            }"
          >
            <input
              type="radio"
              name="dest_choice"
              checked={!pendingDownload.useCustomDir}
              on:change={() => {
                if (pendingDownload) pendingDownload.useCustomDir = false
              }}
              class="mt-0.5 text-indigo-600 focus:ring-indigo-500"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-1.5 text-xs font-semibold text-zinc-200">
                <FolderCheck class="h-3.5 w-3.5 text-indigo-400" />
                Folder Default
              </div>
              <p class="mt-0.5 text-[11px] text-zinc-400 break-all font-mono">
                {pendingDownload.defaultDir || './downloads'}
              </p>
            </div>
          </label>

          <!-- Option 2: Custom Folder -->
          <label
            class="flex items-start gap-3 rounded-xl border p-3 cursor-pointer transition-all {
              pendingDownload.useCustomDir
                ? 'border-indigo-500/80 bg-indigo-500/10'
                : 'border-zinc-800 bg-zinc-950/40 hover:bg-zinc-800/40'
            }"
          >
            <input
              type="radio"
              name="dest_choice"
              checked={pendingDownload.useCustomDir}
              on:change={() => {
                if (pendingDownload) pendingDownload.useCustomDir = true
              }}
              class="mt-0.5 text-indigo-600 focus:ring-indigo-500"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-1.5 text-xs font-semibold text-zinc-200">
                  <FolderSearch class="h-3.5 w-3.5 text-amber-400" />
                  Folder Lain (Pilih Manual)
                </div>
                <button
                  type="button"
                  class="flex items-center gap-1 rounded-lg bg-zinc-800 px-2.5 py-1 text-[11px] font-semibold text-zinc-200 hover:bg-zinc-700 hover:text-white active:scale-95 transition-all border border-zinc-700 shrink-0"
                  on:click|stopPropagation={handleBrowseCustomFolder}
                  disabled={isBrowsingFolder}
                >
                  {#if isBrowsingFolder}
                    <Loader2 class="h-3 w-3 animate-spin" />
                    <span>Membuka...</span>
                  {:else}
                    <FolderSearch class="h-3 w-3 text-amber-400" />
                    <span>Browse...</span>
                  {/if}
                </button>
              </div>
              <p class="mt-1 text-[11px] text-zinc-400 break-all font-mono">
                {pendingDownload.customDir || '(Klik Browse... untuk memilih folder root baru)'}
              </p>
            </div>
          </label>
        </div>

        <!-- Subtitle selection in confirmation modal -->
        <div class="mt-3 flex items-center justify-between rounded-xl border border-zinc-800 bg-zinc-950/40 p-3">
          <div class="flex items-center gap-2">
            <Subtitles class="h-4 w-4 text-emerald-400 shrink-0" />
            <span class="text-xs font-medium text-zinc-300">Subtitle Bahasa:</span>
          </div>
          <select
            bind:value={pendingDownload.subLang}
            class="rounded-lg border border-zinc-700 bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-200 focus:border-indigo-500 focus:outline-none"
          >
            <option value="Indonesian">Indonesian</option>
            <option value="English">English</option>
            <option value="None">None</option>
          </select>
        </div>

        <!-- Structure Preview (Jellyfin / Plex format) -->
        <div class="mt-3 rounded-xl border border-zinc-800/80 bg-zinc-950/70 p-3">
          <div class="flex items-center gap-1.5 text-[10px] font-semibold text-zinc-400 uppercase tracking-wider">
            <Sparkles class="h-3 w-3 text-indigo-400" />
            Struktur Folder Otomatis (Jellyfin / Plex):
          </div>
          <p class="mt-1 text-[11px] text-indigo-300 font-mono break-all leading-relaxed">
            {getFolderPreview(pendingDownload)}
          </p>
          {#if pendingDownload.subLang && pendingDownload.subLang !== 'None'}
            <p class="mt-0.5 text-[10px] text-emerald-400/90 font-mono">
              + Subtitle: {pendingDownload.subLang}.srt
            </p>
          {/if}
        </div>

        <!-- Modal Actions -->
        <div class="mt-5 flex items-center justify-end gap-2.5 border-t border-zinc-800/80 pt-4">
          <button
            type="button"
            class="rounded-xl border border-zinc-700 bg-zinc-800/60 px-4 py-2.5 text-xs font-semibold text-zinc-300 hover:bg-zinc-800 hover:text-white active:scale-95 transition-all"
            on:click={() => (pendingDownload = null)}
            disabled={isSubmitting}
          >
            Batal
          </button>
          <button
            type="button"
            class="flex items-center justify-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-xs font-bold text-white shadow-lg shadow-indigo-600/25 hover:bg-indigo-500 active:scale-95 transition-all disabled:opacity-50"
            on:click={handleConfirmDownload}
            disabled={isSubmitting || (pendingDownload.useCustomDir && !pendingDownload.customDir)}
          >
            {#if isSubmitting}
              <Loader2 class="h-4 w-4 animate-spin" />
              <span>Memproses...</span>
            {:else}
              <Download class="h-4 w-4" />
              <span>Mulai Download</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
