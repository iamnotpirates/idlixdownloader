<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import type { MediaItem, MovieDetails, SeriesDetails } from './types'
  import { fetchMovieDetails, fetchSeriesDetails, startDownload } from './api'
  import { X, Download, Star, Tv, Film, CheckCircle2, AlertCircle, Loader2, Clock, Sparkles } from 'lucide-svelte'

  export let item: MediaItem
  export let onClose: () => void
  export let onDownloadStarted: (title: string) => void

  let seriesDetails: SeriesDetails | null = null
  let movieDetails: MovieDetails | null = null
  let selectedSeasonIndex = 0
  let loading = false
  let errorMsg = ''
  let subtitleLang = 'Indonesian'
  let downloadingEpisodes: Record<string, boolean> = {}
  let isDownloadingMovie = false
  let successMsg = ''

  onMount(async () => {
    // Lock background body scroll
    document.body.style.overflow = 'hidden'

    loading = true
    errorMsg = ''
    try {
      if (item.type === 'TV Series') {
        seriesDetails = await fetchSeriesDetails(item.slug)
      } else {
        movieDetails = await fetchMovieDetails(item.slug)
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

  async function handleDownloadMovie() {
    isDownloadingMovie = true
    errorMsg = ''
    try {
      const year = movieDetails?.year || item.year
      await startDownload({
        page_url: item.url,
        media_type: 'movie',
        title: item.title,
        year,
        sub_lang: subtitleLang,
      })
      successMsg = `Queued "${item.title}" for download!`
      onDownloadStarted(item.title)
      setTimeout(() => {
        onClose()
      }, 1200)
    } catch (err: any) {
      errorMsg = err.message || 'Failed to start download'
    } finally {
      isDownloadingMovie = false
    }
  }

  async function handleDownloadEpisode(mediaId: string, seasonNum: number, episodeNum: number, epTitle: string) {
    const key = `${seasonNum}-${episodeNum}`
    downloadingEpisodes[key] = true
    errorMsg = ''

    try {
      const year = seriesDetails?.year || item.year
      const fullTitle = `${item.title} - S${String(seasonNum).padStart(2, '0')}E${String(episodeNum).padStart(2, '0')}`
      await startDownload({
        page_url: item.url,
        media_type: 'series',
        media_id: mediaId,
        title: item.title,
        year,
        season_num: seasonNum,
        episode_num: episodeNum,
        sub_lang: subtitleLang,
      })
      successMsg = `Queued "${fullTitle}"!`
      onDownloadStarted(fullTitle)
    } catch (err: any) {
      errorMsg = err.message || 'Failed to download episode'
    } finally {
      downloadingEpisodes[key] = false
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
                on:click={handleDownloadMovie}
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
                  on:click={() => handleDownloadEpisode(ep.media_id, ep.season_num, ep.episode_num, ep.title)}
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
</div>
