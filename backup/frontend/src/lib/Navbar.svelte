<script lang="ts">
  import { Film, Download, Settings, Search } from 'lucide-svelte'
  import { navigateTo, type Route } from './router'

  export let activeTab: Route = 'browse'
  export let activeDownloadsCount: number = 0
</script>

<!-- Top Header Bar -->
<header class="sticky top-0 z-40 w-full border-b border-zinc-800/80 bg-zinc-950/85 backdrop-blur-xl">
  <div class="mx-auto flex h-14 sm:h-16 max-w-7xl items-center justify-between px-3 sm:px-6">
    <!-- Brand Logo -->
    <a
      href="/"
      class="flex items-center gap-2.5 transition-opacity active:opacity-80"
      on:click|preventDefault={() => navigateTo('browse')}
    >
      <div class="flex h-8 w-8 sm:h-10 sm:w-10 items-center justify-center rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 text-white shadow-md shadow-indigo-500/20">
        <Film class="h-4 w-4 sm:h-5 sm:w-5" />
      </div>
      <div>
        <div class="flex items-center gap-1.5">
          <span class="text-sm sm:text-base font-bold tracking-tight text-white">IDLIX</span>
          <span class="rounded bg-indigo-500/20 px-1 py-0.5 text-[9px] sm:text-[10px] font-bold text-indigo-400">RUST</span>
        </div>
        <p class="hidden sm:block text-[10px] text-zinc-400">Stream Downloader</p>
      </div>
    </a>

    <!-- Desktop Navigation Tabs (Hidden on mobile) -->
    <nav class="hidden sm:flex items-center gap-1 rounded-xl bg-zinc-900/90 p-1 border border-zinc-800/60">
      <a
        href="/"
        class="flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-xs font-semibold transition-all {activeTab === 'browse'
          ? 'bg-zinc-800 text-white shadow-sm'
          : 'text-zinc-400 hover:text-zinc-200'}"
        on:click|preventDefault={() => navigateTo('browse')}
      >
        <Search class="h-3.5 w-3.5" />
        <span>Browse</span>
      </a>

      <a
        href="/downloads"
        class="relative flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-xs font-semibold transition-all {activeTab === 'downloads'
          ? 'bg-zinc-800 text-white shadow-sm'
          : 'text-zinc-400 hover:text-zinc-200'}"
        on:click|preventDefault={() => navigateTo('downloads')}
      >
        <Download class="h-3.5 w-3.5" />
        <span>Downloads</span>
        {#if activeDownloadsCount > 0}
          <span class="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-indigo-500 px-1 text-[10px] font-bold text-white animate-pulse">
            {activeDownloadsCount}
          </span>
        {/if}
      </a>

      <a
        href="/settings"
        class="flex items-center gap-2 rounded-lg px-3.5 py-1.5 text-xs font-semibold transition-all {activeTab === 'settings'
          ? 'bg-zinc-800 text-white shadow-sm'
          : 'text-zinc-400 hover:text-zinc-200'}"
        on:click|preventDefault={() => navigateTo('settings')}
      >
        <Settings class="h-3.5 w-3.5" />
        <span>Settings</span>
      </a>
    </nav>

    <!-- Mobile Quick Status Icon -->
    {#if activeDownloadsCount > 0}
      <a
        href="/downloads"
        class="sm:hidden flex items-center gap-1.5 rounded-full bg-indigo-500/20 border border-indigo-500/30 px-2.5 py-1 text-xs font-semibold text-indigo-300"
        on:click|preventDefault={() => navigateTo('downloads')}
      >
        <Download class="h-3.5 w-3.5 animate-bounce" />
        <span>{activeDownloadsCount} Active</span>
      </a>
    {/if}
  </div>
</header>

<!-- Mobile Bottom Navigation Bar (Fixed at bottom for thumb-reach UX) -->
<nav class="sm:hidden fixed bottom-0 left-0 right-0 z-40 border-t border-zinc-800/90 bg-zinc-950/95 backdrop-blur-2xl px-2 py-1.5 shadow-2xl">
  <div class="grid grid-cols-3 gap-1">
    <!-- Browse Tab -->
    <a
      href="/"
      class="flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl transition-all active:scale-95 {activeTab === 'browse'
        ? 'text-indigo-400 bg-indigo-500/10 font-bold'
        : 'text-zinc-400 font-medium hover:text-zinc-200'}"
      on:click|preventDefault={() => navigateTo('browse')}
    >
      <Search class="h-5 w-5" />
      <span class="text-[10px]">Browse</span>
    </a>

    <!-- Downloads Tab -->
    <a
      href="/downloads"
      class="relative flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl transition-all active:scale-95 {activeTab === 'downloads'
        ? 'text-indigo-400 bg-indigo-500/10 font-bold'
        : 'text-zinc-400 font-medium hover:text-zinc-200'}"
      on:click|preventDefault={() => navigateTo('downloads')}
    >
      <div class="relative">
        <Download class="h-5 w-5" />
        {#if activeDownloadsCount > 0}
          <span class="absolute -top-1.5 -right-2.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-indigo-500 px-1 text-[9px] font-extrabold text-white ring-2 ring-zinc-950">
            {activeDownloadsCount}
          </span>
        {/if}
      </div>
      <span class="text-[10px]">Downloads</span>
    </a>

    <!-- Settings Tab -->
    <a
      href="/settings"
      class="flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl transition-all active:scale-95 {activeTab === 'settings'
        ? 'text-indigo-400 bg-indigo-500/10 font-bold'
        : 'text-zinc-400 font-medium hover:text-zinc-200'}"
      on:click|preventDefault={() => navigateTo('settings')}
    >
      <Settings class="h-5 w-5" />
      <span class="text-[10px]">Settings</span>
    </a>
  </div>
</nav>
