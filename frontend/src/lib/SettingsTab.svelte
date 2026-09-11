<script lang="ts">
  import { onMount } from 'svelte'
  import type { AppConfig } from './types'
  import { fetchSettings, saveSettings } from './api'
  import { Settings, Save, CheckCircle2, AlertCircle, Loader2, Film, Tv, Globe, Layers, Subtitles } from 'lucide-svelte'

  let config: AppConfig = {
    download_dir: './downloads',
    movies_dir: './downloads/Movies',
    series_dir: './downloads/TV Series',
    base_url: 'https://z2.idlixku.com',
    max_concurrent_downloads: 2,
    default_sub_lang: 'Indonesian',
  }

  let loading = false
  let saving = false
  let successMsg = ''
  let errorMsg = ''

  onMount(async () => {
    loading = true
    try {
      config = await fetchSettings()
    } catch (err: any) {
      errorMsg = err.message || 'Failed to load settings'
    } finally {
      loading = false
    }
  })

  async function handleSave() {
    saving = true
    successMsg = ''
    errorMsg = ''
    try {
      config = await saveSettings(config)
      successMsg = 'Settings saved successfully and saved to config.json'
      setTimeout(() => (successMsg = ''), 4000)
    } catch (err: any) {
      errorMsg = err.message || 'Failed to save settings'
    } finally {
      saving = false
    }
  }
</script>

<div class="mx-auto max-w-2xl px-3 sm:px-4 py-4 sm:py-8">
  <div class="mb-4 sm:mb-6 flex items-center justify-between">
    <div>
      <h2 class="text-lg sm:text-xl font-bold text-white flex items-center gap-2">
        <Settings class="h-4 w-4 sm:h-5 sm:w-5 text-indigo-400" />
        Application Settings
      </h2>
      <p class="text-[11px] sm:text-xs text-zinc-400">Configure separate folders for Movies and TV Series, and IDLIX mirror source</p>
    </div>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-indigo-500" />
    </div>
  {:else}
    <div class="flex flex-col gap-4 sm:gap-6 rounded-2xl sm:rounded-3xl border border-zinc-800/80 bg-zinc-900/80 p-4 sm:p-6 shadow-sm backdrop-blur-sm">
      {#if successMsg}
        <div class="flex items-center gap-2 rounded-xl bg-emerald-500/10 border border-emerald-500/30 p-3 text-xs text-emerald-400">
          <CheckCircle2 class="h-4 w-4 shrink-0" />
          <span>{successMsg}</span>
        </div>
      {/if}

      {#if errorMsg}
        <div class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/30 p-3 text-xs text-red-400">
          <AlertCircle class="h-4 w-4 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      {/if}

      <!-- Movies Directory -->
      <div class="flex flex-col gap-1.5">
        <label class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
          <Film class="h-3.5 w-3.5 text-indigo-400" />
          Movies Directory (Jellyfin / Plex)
        </label>
        <input
          type="text"
          bind:value={config.movies_dir}
          class="w-full rounded-xl border border-zinc-700 bg-zinc-950 px-3.5 py-2.5 text-xs text-white placeholder-zinc-500 focus:border-indigo-500 focus:outline-none"
          placeholder="Z:\Film\Movies"
        />
        <p class="text-[10px] text-zinc-500">
          Format: <code class="text-zinc-400">{config.movies_dir || 'Movies'}\[Title (Year)]\[Title (Year)].mp4</code>
        </p>
      </div>

      <!-- TV Series Directory -->
      <div class="flex flex-col gap-1.5">
        <label class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
          <Tv class="h-3.5 w-3.5 text-violet-400" />
          TV Series Directory (Jellyfin / Plex)
        </label>
        <input
          type="text"
          bind:value={config.series_dir}
          class="w-full rounded-xl border border-zinc-700 bg-zinc-950 px-3.5 py-2.5 text-xs text-white placeholder-zinc-500 focus:border-indigo-500 focus:outline-none"
          placeholder="Z:\Film\Series"
        />
        <p class="text-[10px] text-zinc-500">
          Format: <code class="text-zinc-400">{config.series_dir || 'TV Series'}\[Series (Year)]\Season 01\[Series - S01E01].mp4</code>
        </p>
      </div>

      <!-- IDLIX Base Mirror URL -->
      <div class="flex flex-col gap-1.5">
        <label class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
          <Globe class="h-3.5 w-3.5 text-sky-400" />
          IDLIX Base Mirror URL
        </label>
        <input
          type="text"
          bind:value={config.base_url}
          class="w-full rounded-xl border border-zinc-700 bg-zinc-950 px-3.5 py-2.5 text-xs text-white placeholder-zinc-500 focus:border-indigo-500 focus:outline-none"
          placeholder="https://z2.idlixku.com"
        />
        <p class="text-[10px] text-zinc-500">
          Ubah jika domain mirror IDLIX berganti.
        </p>
      </div>

      <!-- Default Subtitle Language -->
      <div class="flex flex-col gap-1.5">
        <label class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
          <Subtitles class="h-3.5 w-3.5 text-emerald-400" />
          Default Subtitle Language
        </label>
        <select
          bind:value={config.default_sub_lang}
          class="w-full rounded-xl border border-zinc-700 bg-zinc-950 px-3.5 py-2.5 text-xs text-white focus:border-indigo-500 focus:outline-none"
        >
          <option value="Indonesian">Indonesian (Bahasa)</option>
          <option value="English">English</option>
          <option value="None">None</option>
        </select>
      </div>

      <!-- Save Button -->
      <div class="mt-2 sm:mt-4 flex justify-end border-t border-zinc-800 pt-4">
        <button
          class="flex items-center justify-center gap-2 rounded-xl bg-indigo-600 px-5 py-3 sm:py-2.5 text-xs font-semibold text-white shadow-md shadow-indigo-600/20 hover:bg-indigo-500 active:scale-95 transition-all disabled:opacity-50 w-full sm:w-auto"
          on:click={handleSave}
          disabled={saving}
        >
          {#if saving}
            <Loader2 class="h-4 w-4 animate-spin" />
            <span>Saving...</span>
          {:else}
            <Save class="h-4 w-4" />
            <span>Save Changes</span>
          {/if}
        </button>
      </div>
    </div>
  {/if}
</div>
