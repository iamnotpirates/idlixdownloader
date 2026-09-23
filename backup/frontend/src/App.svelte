<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import type { DownloadTask, MediaItem } from './lib/types'
  import { fetchCatalog, searchMedia, createDownloadSocket, fetchDownloads } from './lib/api'
  import { getRouteFromPath, navigateTo, type Route } from './lib/router'
  import Sidebar from './lib/Sidebar.svelte'
  import MediaCard from './lib/MediaCard.svelte'
  import MediaModal from './lib/MediaModal.svelte'
  import DownloadsTab from './lib/DownloadsTab.svelte'
  import SettingsTab from './lib/SettingsTab.svelte'
  import {
    Search,
    Loader2,
    Sparkles,
    AlertCircle,
    Settings,
    RefreshCw,
    CheckCircle2,
    ArrowRight,
    X,
    Menu,
    Compass,
    Download,
    Film,
    Activity,
    Database,
  } from 'lucide-svelte'

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
  let toast: { message: string; title: string; type?: 'movie' | 'series' } | null = null
  let toastTimer: number | null = null
  let mobileDrawerOpen = false

  $: activeDownloads = downloads.filter(
    (d) => d.status === 'downloading' || d.status === 'queued' || d.status === 'extracting'
  )
  $: activeDownloadsCount = activeDownloads.length
  $: displayItems = searchQuery.trim() ? searchResults : catalogItems

  function syncRoute() {
    activeTab = getRouteFromPath()
    if (activeTab !== 'browse') {
      selectedMedia = null
    }
  }

  onMount(async () => {
    window.addEventListener('popstate', syncRoute)

    loadCatalog()

    try {
      downloads = await fetchDownloads()
    } catch (e) {
      console.error('Failed to load initial downloads', e)
    }

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

  function showToast(message: string, title: string, type?: 'movie' | 'series') {
    if (toastTimer) clearTimeout(toastTimer)
    toast = { message, title, type }
    toastTimer = window.setTimeout(() => {
      toast = null
    }, 5000)
  }

  function handleDownloadStarted(title: string, mediaType?: 'movie' | 'series') {
    showToast('Download berhasil diantrekan', title, mediaType)

    if (mediaType === 'movie') {
      selectedMedia = null
      setTimeout(() => {
        navigateTo('downloads')
      }, 300)
    }
  }
</script>

<div class="flex h-screen w-full bg-zinc-950 text-zinc-100 overflow-hidden selection:bg-indigo-500 selection:text-white font-sans antialiased">
  <!-- Desktop Left Sidebar -->
  <div class="hidden md:flex md:w-64 lg:w-72 shrink-0 h-full">
    <Sidebar {activeTab} {downloads} />
  </div>

  <!-- Mobile Drawer Backdrop -->
  {#if mobileDrawerOpen}
    <button
      type="button"
      class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm md:hidden transition-opacity cursor-default border-none outline-none"
      on:click={() => (mobileDrawerOpen = false)}
      aria-label="Close drawer"
    ></button>
    <!-- Mobile Drawer Content -->
    <div class="fixed inset-y-0 left-0 z-50 w-72 max-w-[80vw] bg-zinc-950 md:hidden shadow-2xl animate-in slide-in-from-left duration-200">
      <Sidebar {activeTab} {downloads} onNavigate={() => (mobileDrawerOpen = false)} />
    </div>
  {/if}

  <!-- Main View Area (Right Pane) -->
  <div class="flex-1 flex flex-col min-w-0 h-full overflow-hidden bg-zinc-950">
    <!-- Top Header Bar -->
    <header class="h-16 shrink-0 border-b border-zinc-800/80 bg-zinc-950/80 px-4 sm:px-6 flex items-center justify-between backdrop-blur-xl z-30">
      <!-- Left: Mobile Menu Button or Desktop Breadcrumb -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          class="flex md:hidden h-9 w-9 items-center justify-center rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-300 hover:text-white"
          on:click={() => (mobileDrawerOpen = true)}
          aria-label="Open menu"
        >
          <Menu class="h-5 w-5" />
        </button>

        <div class="flex flex-col">
          <div class="flex items-center gap-2">
            {#if activeTab === 'browse'}
              <Compass class="h-4 w-4 text-indigo-400" />
              <h1 class="text-sm sm:text-base font-bold text-white tracking-tight">Katalog & Pencarian</h1>
            {:else if activeTab === 'downloads'}
              <Download class="h-4 w-4 text-indigo-400" />
              <h1 class="text-sm sm:text-base font-bold text-white tracking-tight">Antrean & Riwayat Download</h1>
            {:else if activeTab === 'settings'}
              <Settings class="h-4 w-4 text-indigo-400" />
              <h1 class="text-sm sm:text-base font-bold text-white tracking-tight">Pengaturan Sistem</h1>
            {/if}
          </div>
          <span class="text-[10px] text-zinc-400 hidden sm:inline">
            {#if activeTab === 'browse'}
              Jelajahi film, series dan episode terbaru IDLIX
            {:else if activeTab === 'downloads'}
              Pantau progres unduhan multi-threaded dan log proses
            {:else if activeTab === 'settings'}
              Kelola direktori penyimpanan, thread paralel, dan konfigurasi API
            {/if}
          </span>
        </div>
      </div>

      <!-- Right: Status Indicators & Quick Actions -->
      <div class="flex items-center gap-2.5">
        {#if activeDownloadsCount > 0}
          <button
            class="flex items-center gap-2 rounded-xl bg-indigo-500/10 border border-indigo-500/30 px-3 py-1.5 text-xs font-semibold text-indigo-300 hover:bg-indigo-500/20 transition-all"
            on:click={() => navigateTo('downloads')}
          >
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-indigo-500"></span>
            </span>
            <span class="font-mono">{activeDownloadsCount} Aktif</span>
          </button>
        {/if}

        <div class="hidden sm:flex items-center gap-1.5 rounded-xl border border-zinc-800 bg-zinc-900/60 px-3 py-1.5 text-xs text-zinc-400">
          <Activity class="h-3.5 w-3.5 text-emerald-400" />
          <span class="text-[11px] font-medium text-zinc-300">Rust Core :8989</span>
        </div>
      </div>
    </header>

    <!-- Main Scrollable Content View -->
    <main class="flex-1 overflow-y-auto pb-24 md:pb-10">
      {#if activeTab === 'browse'}
        <div class="mx-auto max-w-7xl px-4 sm:px-8 py-5 sm:py-8">
          <!-- Hero Search Card -->
          <div class="relative mb-6 sm:mb-10 overflow-hidden rounded-3xl border border-zinc-800/80 bg-gradient-to-b from-indigo-950/40 via-zinc-900/50 to-zinc-900/20 p-5 sm:p-8 text-center backdrop-blur-md shadow-xl shadow-black/40">
            <div class="mx-auto max-w-2xl">
              <div class="inline-flex items-center gap-1.5 rounded-full bg-indigo-500/10 border border-indigo-500/20 px-3 py-1 text-[11px] font-semibold text-indigo-400 mb-3">
                <Sparkles class="h-3.5 w-3.5" />
                <span>Rust Multi-Threaded Engine</span>
              </div>
              <h2 class="text-xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-white">
                Download Movies & Series
              </h2>
              <p class="mt-1.5 text-xs sm:text-sm text-zinc-400">
                Pencarian cepat, unduhan multi-part otomatis, dan subtitle langsung tersimpan rapi.
              </p>

              <!-- Search Bar -->
              <div class="mt-5 sm:mt-6 flex items-center gap-2">
                <div class="relative flex-1">
                  <Search class="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
                  <input
                    type="text"
                    bind:value={searchQuery}
                    on:keydown={handleKeyDown}
                    placeholder="Cari judul (contoh: Moana, Avatar, Breaking Bad)..."
                    class="w-full rounded-2xl border border-zinc-700/80 bg-zinc-900/90 pl-10 pr-4 py-2.5 sm:py-3 text-xs sm:text-sm text-white placeholder-zinc-500 shadow-inner focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <button
                  class="flex items-center gap-2 rounded-2xl bg-indigo-600 px-4 sm:px-6 py-2.5 sm:py-3 text-xs sm:text-sm font-bold text-white shadow-lg shadow-indigo-600/25 hover:bg-indigo-500 active:scale-95 transition-all disabled:opacity-50 shrink-0"
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
            <div class="mb-6 flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-2xl bg-red-500/10 border border-red-500/30 p-4 text-xs text-red-300">
              <div class="flex items-center gap-3">
                <AlertCircle class="h-5 w-5 shrink-0 text-red-400" />
                <div>
                  <p class="font-semibold text-red-200">Gagal terhubung ke IDLIX</p>
                  <p class="text-[11px] text-red-400/90 mt-0.5">{errorMsg}. Cek koneksi internet atau perbarui Base URL di Settings.</p>
                </div>
              </div>
              <div class="flex items-center gap-2 self-end sm:self-auto">
                <button
                  class="flex items-center gap-1.5 rounded-xl bg-zinc-800 border border-zinc-700 px-3 py-1.5 text-xs font-semibold text-zinc-200 hover:bg-zinc-700 active:scale-95"
                  on:click={() => navigateTo('settings')}
                >
                  <Settings class="h-3.5 w-3.5 text-zinc-400" />
                  <span>Ganti Domain</span>
                </button>
                <button
                  class="flex items-center gap-1.5 rounded-xl bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-500 active:scale-95 shadow-sm"
                  on:click={loadCatalog}
                >
                  <RefreshCw class="h-3.5 w-3.5" />
                  <span>Coba Lagi</span>
                </button>
              </div>
            </div>
          {/if}

          <!-- Section Header -->
          <div class="mb-5 flex items-center justify-between px-1">
            <div>
              <h3 class="text-base sm:text-lg font-bold text-white">
                {searchQuery.trim() ? `Hasil Pencarian "${searchQuery}"` : 'Popular & Featured Media'}
              </h3>
              <p class="text-xs text-zinc-400">
                {displayItems.length} judul ditemukan
              </p>
            </div>

            {#if searchQuery.trim()}
              <button
                class="text-xs font-semibold text-indigo-400 hover:text-indigo-300 hover:underline p-1"
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
            <div class="grid grid-cols-2 gap-3 sm:gap-5 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
              {#each Array(12) as _}
                <div class="flex flex-col gap-2 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-2.5 animate-pulse">
                  <div class="aspect-[2/3] w-full rounded-xl bg-zinc-800"></div>
                  <div class="h-3 w-3/4 rounded bg-zinc-800"></div>
                </div>
              {/each}
            </div>
          {:else if displayItems.length === 0}
            <div class="flex flex-col items-center justify-center rounded-3xl border border-zinc-800/80 bg-zinc-900/40 py-20 text-center">
              <p class="text-sm font-semibold text-zinc-300">Tidak ada film atau series ditemukan</p>
              <p class="text-xs text-zinc-500 mt-1">Coba kata kunci pencarian yang lain.</p>
            </div>
          {:else}
            <div class="grid grid-cols-2 gap-3 sm:gap-5 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
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

    <!-- Mobile Bottom Navigation Bar (Hidden on Desktop) -->
    <nav class="fixed bottom-0 inset-x-0 z-40 border-t border-zinc-800/80 bg-zinc-950/95 backdrop-blur-xl px-4 py-2 md:hidden">
      <div class="grid grid-cols-3 gap-1">
        <button
          type="button"
          class="flex flex-col items-center justify-center gap-1 py-1 rounded-xl transition-all {activeTab === 'browse'
            ? 'text-indigo-400 font-bold bg-indigo-500/10'
            : 'text-zinc-400 font-medium'}"
          on:click={() => navigateTo('browse')}
        >
          <Compass class="h-4 w-4" />
          <span class="text-[10px]">Katalog</span>
        </button>

        <button
          type="button"
          class="relative flex flex-col items-center justify-center gap-1 py-1 rounded-xl transition-all {activeTab === 'downloads'
            ? 'text-indigo-400 font-bold bg-indigo-500/10'
            : 'text-zinc-400 font-medium'}"
          on:click={() => navigateTo('downloads')}
        >
          <div class="relative">
            <Download class="h-4 w-4" />
            {#if activeDownloadsCount > 0}
              <span class="absolute -top-1 -right-2 flex h-3.5 min-w-[14px] items-center justify-center rounded-full bg-indigo-500 px-1 text-[8px] font-extrabold text-white">
                {activeDownloadsCount}
              </span>
            {/if}
          </div>
          <span class="text-[10px]">Downloads</span>
        </button>

        <button
          type="button"
          class="flex flex-col items-center justify-center gap-1 py-1 rounded-xl transition-all {activeTab === 'settings'
            ? 'text-indigo-400 font-bold bg-indigo-500/10'
            : 'text-zinc-400 font-medium'}"
          on:click={() => navigateTo('settings')}
        >
          <Settings class="h-4 w-4" />
          <span class="text-[10px]">Pengaturan</span>
        </button>
      </div>
    </nav>
  </div>

  <!-- Media Details & Episode Selector Modal -->
  {#if selectedMedia && activeTab === 'browse'}
    <MediaModal
      item={selectedMedia}
      {downloads}
      onClose={() => (selectedMedia = null)}
      onDownloadStarted={handleDownloadStarted}
    />
  {/if}

  <!-- Floating Toast Notification -->
  {#if toast}
    <div class="fixed bottom-20 md:bottom-6 right-3 sm:right-6 z-70 max-w-sm w-full animate-in slide-in-from-bottom-5 fade-in duration-200">
      <div class="flex items-center justify-between gap-3 rounded-2xl border border-indigo-500/30 bg-zinc-900/95 p-3.5 shadow-2xl backdrop-blur-md">
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-indigo-600/20 text-indigo-400">
            <CheckCircle2 class="h-5 w-5" />
          </div>
          <div class="min-w-0">
            <h4 class="text-xs font-bold text-white truncate">{toast.title}</h4>
            <p class="text-[11px] text-zinc-400">{toast.message}</p>
          </div>
        </div>

        <div class="flex items-center gap-1.5 shrink-0">
          <button
            class="flex items-center gap-1 rounded-lg bg-indigo-600 px-2.5 py-1.5 text-xs font-bold text-white shadow-sm hover:bg-indigo-500 active:scale-95 transition-all"
            on:click={() => {
              toast = null
              selectedMedia = null
              navigateTo('downloads')
            }}
          >
            <span>Lihat</span>
            <ArrowRight class="h-3.5 w-3.5" />
          </button>
          <button
            class="rounded-lg p-1.5 text-zinc-400 hover:text-white hover:bg-zinc-800 transition-all"
            on:click={() => (toast = null)}
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
