<script lang="ts">
  import {
    Film,
    Download,
    Settings,
    Folder,
    FolderOpen,
    Tv,
    HardDrive,
    Database,
    Activity,
    Compass,
    ExternalLink,
    Sparkles,
    CheckCircle2,
    Clock,
  } from 'lucide-svelte'
  import { navigateTo, type Route } from './router'
  import { openFolder } from './api'
  import type { DownloadTask } from './types'

  export let activeTab: Route = 'browse'
  export let downloads: DownloadTask[] = []
  export let onNavigate: (() => void) | undefined = undefined

  $: activeDownloads = downloads.filter(
    (d) => d.status === 'downloading' || d.status === 'queued' || d.status === 'extracting'
  )
  $: activeCount = activeDownloads.length
  $: completedCount = downloads.filter((d) => d.status === 'completed').length

  let openingFolder: 'movies' | 'series' | null = null

  async function handleOpenFolder(target: 'movies' | 'series') {
    openingFolder = target
    try {
      await openFolder(target)
    } catch (e) {
      console.error('Gagal membuka folder:', e)
    } finally {
      setTimeout(() => {
        openingFolder = null
      }, 600)
    }
  }

  function handleNav(route: Route) {
    navigateTo(route)
    if (onNavigate) onNavigate()
  }
</script>

<aside class="flex h-full w-full flex-col justify-between bg-zinc-950/95 border-r border-zinc-800/80 p-4 select-none backdrop-blur-xl">
  <!-- Top Brand & Navigation -->
  <div class="flex flex-col gap-6">
    <!-- Brand Logo -->
    <a
      href="/"
      class="group flex items-center gap-3 px-2 py-1.5 transition-all"
      on:click|preventDefault={() => handleNav('browse')}
    >
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-tr from-indigo-600 via-indigo-500 to-violet-500 text-white shadow-lg shadow-indigo-600/25 group-hover:scale-105 transition-transform">
        <Film class="h-5 w-5" />
      </div>
      <div class="flex flex-col min-w-0">
        <div class="flex items-center gap-1.5">
          <span class="text-base font-extrabold tracking-tight text-white">IDLIX</span>
          <span class="rounded-md bg-indigo-500/20 px-1.5 py-0.5 text-[9px] font-bold text-indigo-400 border border-indigo-500/30">
            PRO
          </span>
        </div>
        <span class="text-[11px] font-medium text-zinc-400 truncate">Stream Downloader</span>
      </div>
    </a>

    <!-- Navigation Section -->
    <nav class="flex flex-col gap-5">
      <!-- Section: Main Menu -->
      <div>
        <div class="px-2.5 mb-2 text-[10px] font-bold uppercase tracking-wider text-zinc-400">
          Menu Utama
        </div>
        <div class="flex flex-col gap-1">
          <!-- Browse Catalog -->
          <a
            href="/"
            class="flex items-center justify-between rounded-xl px-3 py-2.5 text-xs font-semibold transition-all active:scale-[0.98] {activeTab === 'browse'
              ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
              : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'}"
            on:click|preventDefault={() => handleNav('browse')}
          >
            <div class="flex items-center gap-2.5">
              <Compass class="h-4 w-4 shrink-0" />
              <span>Katalog & Cari</span>
            </div>
            {#if activeTab === 'browse'}
              <span class="h-1.5 w-1.5 rounded-full bg-white animate-pulse"></span>
            {/if}
          </a>

          <!-- Downloads Manager -->
          <a
            href="/downloads"
            class="flex items-center justify-between rounded-xl px-3 py-2.5 text-xs font-semibold transition-all active:scale-[0.98] {activeTab === 'downloads'
              ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
              : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'}"
            on:click|preventDefault={() => handleNav('downloads')}
          >
            <div class="flex items-center gap-2.5">
              <Download class="h-4 w-4 shrink-0" />
              <span>Antrean Download</span>
            </div>
            {#if activeCount > 0}
              <span class="flex items-center gap-1 rounded-full bg-indigo-500/30 border border-indigo-400/40 px-2 py-0.5 text-[10px] font-bold {activeTab === 'downloads' ? 'text-white' : 'text-indigo-400'}">
                <span class="h-1.5 w-1.5 rounded-full bg-indigo-400 animate-ping"></span>
                <span>{activeCount}</span>
              </span>
            {:else if downloads.length > 0}
              <span class="rounded-full bg-zinc-800 px-2 py-0.5 text-[10px] font-bold text-zinc-400">
                {downloads.length}
              </span>
            {/if}
          </a>

          <!-- Settings -->
          <a
            href="/settings"
            class="flex items-center justify-between rounded-xl px-3 py-2.5 text-xs font-semibold transition-all active:scale-[0.98] {activeTab === 'settings'
              ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
              : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'}"
            on:click|preventDefault={() => handleNav('settings')}
          >
            <div class="flex items-center gap-2.5">
              <Settings class="h-4 w-4 shrink-0" />
              <span>Pengaturan & Path</span>
            </div>
          </a>
        </div>
      </div>

      <!-- Section: Storage Folders -->
      <div>
        <div class="px-2.5 mb-2 text-[10px] font-bold uppercase tracking-wider text-zinc-400">
          Akses Penyimpanan
        </div>
        <div class="flex flex-col gap-1">
          <button
            type="button"
            class="flex items-center justify-between rounded-xl px-3 py-2 text-xs font-medium text-zinc-400 hover:bg-zinc-900/80 hover:text-zinc-200 transition-all text-left"
            on:click={() => handleOpenFolder('movies')}
            disabled={openingFolder === 'movies'}
          >
            <div class="flex items-center gap-2.5 truncate">
              <Film class="h-4 w-4 text-indigo-400 shrink-0" />
              <span class="truncate">Folder Movies</span>
            </div>
            <FolderOpen class="h-3.5 w-3.5 text-zinc-500 shrink-0" />
          </button>

          <button
            type="button"
            class="flex items-center justify-between rounded-xl px-3 py-2 text-xs font-medium text-zinc-400 hover:bg-zinc-900/80 hover:text-zinc-200 transition-all text-left"
            on:click={() => handleOpenFolder('series')}
            disabled={openingFolder === 'series'}
          >
            <div class="flex items-center gap-2.5 truncate">
              <Tv class="h-4 w-4 text-violet-400 shrink-0" />
              <span class="truncate">Folder Series</span>
            </div>
            <FolderOpen class="h-3.5 w-3.5 text-zinc-500 shrink-0" />
          </button>
        </div>
      </div>
    </nav>
  </div>

  <!-- Bottom System Status Card -->
  <div class="flex flex-col gap-2.5 pt-4 border-t border-zinc-900">
    <div class="rounded-2xl border border-zinc-800/80 bg-zinc-900/50 p-3 flex flex-col gap-2">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="relative flex h-2 w-2">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          <span class="text-[11px] font-bold text-zinc-200">Rust Core Server</span>
        </div>
        <span class="text-[10px] font-mono text-zinc-400">:8989</span>
      </div>

      <div class="flex items-center justify-between text-[10px] text-zinc-400 pt-1.5 border-t border-zinc-800/50">
        <div class="flex items-center gap-1">
          <Database class="h-3 w-3 text-indigo-400" />
          <span>SQLite Database</span>
        </div>
        <span class="text-emerald-400 font-semibold">Active</span>
      </div>
    </div>

    <div class="px-1 text-[10px] text-zinc-400 text-center">
      v0.1.3-alpha • IDLIX Engine
    </div>
  </div>
</aside>
