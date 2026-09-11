<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import type { DownloadTask, MediaItem } from './lib/types'
  import { fetchCatalog, searchMedia, createDownloadSocket, fetchDownloads } from './lib/api'
  import { getRouteFromPath, navigateTo, type Route } from './lib/router'
  import Navbar from './lib/Navbar.svelte'
  import MediaCard from './lib/MediaCard.svelte'
  import MediaModal from './lib/MediaModal.svelte'
  import DownloadsTab from './lib/DownloadsTab.svelte'
  import SettingsTab from './lib/SettingsTab.svelte'
  import { Search, Loader2, Sparkles, AlertCircle, Settings, RefreshCw } from 'lucide-svelte'

  let activeTab: Route = getRouteFromPath()
  let catalogItems: MediaItem[] = []
  let searchResults: MediaItem[] = []
  let searchQuery = ''
  let isSearching = false
  let loadingCatalog = false
  let errorMsg = ''
  let selectedMedia: MediaItem | null = null
  let downloads: DownloadTask[] = []
  let cleanupSocket: (() => void) | null = null

  $: activeDownloadsCount = downloads.filter((d) => d.status === 'downloading' || d.status === 'queued' || d.status === 'extracting').length
  $: displayItems = searchQuery.trim() ? searchResults : catalogItems

  function syncRoute() {
    activeTab = getRouteFromPath()
  }

  onMount(async () => {
    // 0. Listen to popstate for browser Back/Forward navigation
    window.addEventListener('popstate', syncRoute)

    // 1. Fetch initial catalog
    loadCatalog()

    // 2. Fetch initial downloads list
    try {
      downloads = await fetchDownloads()
    } catch (e) {
      console.error('Failed to load initial downloads', e)
    }

    // 3. Connect real-time WebSocket
    cleanupSocket = createDownloadSocket((data) => {
      if (Array.isArray(data)) {
        downloads = data
      } else {
        const index = downloads.findIndex((d) => d.id === data.id)
        if (index >= 0) {
          downloads[index] = data
          downloads = [...downloads]
        } else {
          downloads = [data, ...downloads]
        }
      }
    })
  })

  onDestroy(() => {
    window.removeEventListener('popstate', syncRoute)
    if (cleanupSocket) cleanupSocket()
  })

  async function loadCatalog() {
    loadingCatalog = true
    errorMsg = ''
    try {
      catalogItems = await fetchCatalog()
    } catch (err: any) {
      errorMsg = err.message || 'Failed to connect to IDLIX catalog'
    } finally {
      loadingCatalog = false
    }
  }

  async function handleSearch() {
    const q = searchQuery.trim()
    if (!q) {
      searchResults = []
      return
    }

    isSearching = true
    errorMsg = ''
    try {
      searchResults = await searchMedia(q)
    } catch (err: any) {
      errorMsg = err.message || 'Search request failed'
    } finally {
      isSearching = false
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  function handleDownloadStarted(title: string) {
    // Switch to downloads tab after starting download
    setTimeout(() => {
      navigateTo('downloads')
    }, 500)
  }
</script>

<div class="min-h-screen bg-zinc-950 text-zinc-100 flex flex-col selection:bg-indigo-500 selection:text-white">
  <!-- Top Navigation (and Mobile Bottom Nav) -->
  <Navbar bind:activeTab {activeDownloadsCount} />

  <!-- Main View Container (Extra bottom padding on mobile for bottom bar) -->
  <main class="flex-1 pb-24 sm:pb-12">
    {#if activeTab === 'browse'}
      <div class="mx-auto max-w-7xl px-3 sm:px-6 py-4 sm:py-8">
        <!-- Hero Search Section -->
        <div class="relative mb-6 sm:mb-10 overflow-hidden rounded-2xl sm:rounded-3xl border border-zinc-800 bg-gradient-to-b from-indigo-950/40 via-zinc-900/60 to-zinc-900/20 p-4 sm:p-8 text-center backdrop-blur-md">
          <div class="mx-auto max-w-2xl">
            <div class="inline-flex items-center gap-1.5 rounded-full bg-indigo-500/10 border border-indigo-500/20 px-2.5 py-0.5 sm:px-3 sm:py-1 text-[10px] sm:text-xs font-semibold text-indigo-400 mb-2 sm:mb-4">
              <Sparkles class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
              <span>Rust Multi-Threaded Engine</span>
            </div>
            <h1 class="text-xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-white">
              Search Movies & Series
            </h1>
            <p class="mt-1 sm:mt-2 text-[11px] sm:text-xs text-zinc-400">
              Download movies, full seasons, and subtitles directly to local storage.
            </p>

            <!-- Search Bar -->
            <div class="mt-4 sm:mt-6 flex items-center gap-2">
              <div class="relative flex-1">
                <Search class="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
                <input
                  type="text"
                  bind:value={searchQuery}
                  on:keydown={handleKeyDown}
                  placeholder="Cari judul (contoh: Moana, Avatar, Breaking Bad)..."
                  class="w-full rounded-xl sm:rounded-2xl border border-zinc-700/80 bg-zinc-900/90 pl-10 pr-3 sm:pr-4 py-2.5 sm:py-3 text-xs sm:text-sm text-white placeholder-zinc-500 shadow-inner focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <button
                class="flex items-center gap-1.5 sm:gap-2 rounded-xl sm:rounded-2xl bg-indigo-600 px-4 sm:px-6 py-2.5 sm:py-3 text-xs sm:text-sm font-bold text-white shadow-lg shadow-indigo-600/20 hover:bg-indigo-500 active:scale-95 transition-all disabled:opacity-50 shrink-0"
                on:click={handleSearch}
                disabled={isSearching}
              >
                {#if isSearching}
                  <Loader2 class="h-4 w-4 animate-spin" />
                {:else}
                  <Search class="h-4 w-4" />
                  <span>Search</span>
                {/if}
              </button>
            </div>
          </div>
        </div>

        <!-- Error State -->
        {#if errorMsg}
          <div class="mb-4 sm:mb-6 flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-2xl bg-red-500/10 border border-red-500/30 p-3.5 sm:p-4 text-xs text-red-300">
            <div class="flex items-center gap-3">
              <AlertCircle class="h-5 w-5 shrink-0 text-red-400" />
              <div>
                <p class="font-semibold text-red-200">Gagal terhubung ke IDLIX</p>
                <p class="text-[11px] text-red-400/90 mt-0.5">{errorMsg}. Cek koneksi internet atau perbarui Base URL di Settings jika domain IDLIX berganti.</p>
              </div>
            </div>
            <div class="flex items-center gap-2 self-end sm:self-auto">
              <button
                class="flex items-center gap-1.5 rounded-lg bg-zinc-800 border border-zinc-700 px-3 py-1.5 text-xs font-semibold text-zinc-200 hover:bg-zinc-700 active:scale-95"
                on:click={() => navigateTo('settings')}
              >
                <Settings class="h-3.5 w-3.5 text-zinc-400" />
                <span>Ganti Domain</span>
              </button>
              <button
                class="flex items-center gap-1.5 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-500 active:scale-95 shadow-sm"
                on:click={loadCatalog}
              >
                <RefreshCw class="h-3.5 w-3.5" />
                <span>Coba Lagi</span>
              </button>
            </div>
          </div>
        {/if}

        <!-- Section Header -->
        <div class="mb-4 sm:mb-6 flex items-center justify-between px-1">
          <div>
            <h2 class="text-base sm:text-lg font-bold text-white">
              {searchQuery.trim() ? `Hasil Pencarian "${searchQuery}"` : 'Popular & Featured'}
            </h2>
            <p class="text-[11px] sm:text-xs text-zinc-400">
              {displayItems.length} judul ditemukan
            </p>
          </div>

          {#if searchQuery.trim()}
            <button
              class="text-xs font-medium text-indigo-400 hover:underline p-1"
              on:click={() => {
                searchQuery = ''
                searchResults = []
              }}
            >
              Reset Pencarian
            </button>
          {/if}
        </div>

        <!-- Media Grid -->
        {#if loadingCatalog || isSearching}
          <div class="grid grid-cols-2 gap-2.5 sm:gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
            {#each Array(12) as _}
              <div class="flex flex-col gap-2 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-2 animate-pulse">
                <div class="aspect-[2/3] w-full rounded-xl bg-zinc-800" />
                <div class="h-3 w-3/4 rounded bg-zinc-800" />
              </div>
            {/each}
          </div>
        {:else if displayItems.length === 0}
          <div class="flex flex-col items-center justify-center rounded-3xl border border-zinc-800/80 bg-zinc-900/40 py-16 text-center">
            <p class="text-sm font-semibold text-zinc-300">Tidak ada film atau series ditemukan</p>
            <p class="text-xs text-zinc-500 mt-1">Coba kata kunci pencarian yang lain.</p>
          </div>
        {:else}
          <div class="grid grid-cols-2 gap-2.5 sm:gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
            {#each displayItems as item (item.url || item.slug)}
              <MediaCard {item} onSelect={(selected) => (selectedMedia = selected)} />
            {/each}
          </div>
        {/if}
      </div>
    {:else if activeTab === 'downloads'}
      <DownloadsTab {downloads} />
    {:else if activeTab === 'settings'}
      <SettingsTab />
    {/if}
  </main>

  <!-- Media Details & Episode Selector Modal -->
  {#if selectedMedia}
    <MediaModal
      item={selectedMedia}
      onClose={() => (selectedMedia = null)}
      onDownloadStarted={handleDownloadStarted}
    />
  {/if}
</div>
