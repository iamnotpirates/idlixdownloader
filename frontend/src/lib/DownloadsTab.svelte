<script lang="ts">
  import type { DownloadTask } from './types'
  import { Download, Folder, Film, Loader2, CheckCircle2, AlertCircle, Clock } from 'lucide-svelte'

  export let downloads: DownloadTask[] = []

  function getStatusBadge(status: DownloadTask['status']) {
    switch (status) {
      case 'extracting':
        return { label: 'Extracting Stream', bg: 'bg-amber-500/15 text-amber-400 border-amber-500/30' }
      case 'downloading':
        return { label: 'Downloading', bg: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30' }
      case 'completed':
        return { label: 'Completed', bg: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30' }
      case 'queued':
        return { label: 'Queued', bg: 'bg-sky-500/15 text-sky-400 border-sky-500/30' }
      case 'failed':
        return { label: 'Failed', bg: 'bg-rose-500/15 text-rose-400 border-rose-500/30' }
      default:
        return { label: status, bg: 'bg-zinc-800 text-zinc-400 border-zinc-700' }
    }
  }
</script>

<div class="mx-auto max-w-5xl px-4 py-8">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h2 class="text-xl font-bold text-white">Download Manager</h2>
      <p class="text-xs text-zinc-400">Background tasks & downloads</p>
    </div>
    <div class="rounded-xl bg-zinc-900 border border-zinc-800 px-3 py-1.5 text-xs font-medium text-zinc-300">
      Total Tasks: <span class="font-bold text-indigo-400">{downloads.length}</span>
    </div>
  </div>

  {#if downloads.length === 0}
    <div class="flex flex-col items-center justify-center rounded-3xl border border-zinc-800/80 bg-zinc-900/40 py-16 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-zinc-800 text-zinc-500 mb-3">
        <Download class="h-6 w-6" />
      </div>
      <p class="text-sm font-semibold text-zinc-300">No active downloads</p>
      <p class="text-xs text-zinc-500 max-w-sm mt-1">
        Browse movies and TV series to start downloading streams.
      </p>
    </div>
  {:else}
    <div class="flex flex-col gap-3">
      {#each downloads as task (task.id)}
        {@const badge = getStatusBadge(task.status)}
        <div class="flex flex-col gap-2 rounded-2xl border border-zinc-800/80 bg-zinc-900/80 p-4 shadow-sm backdrop-blur-sm transition-all">
          <div class="flex items-center justify-between gap-4">
            <div class="flex items-center gap-3 min-w-0">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400">
                <Film class="h-5 w-5" />
              </div>
              <div class="min-w-0">
                <h3 class="text-sm font-bold text-white truncate">{task.file_name}</h3>
                <div class="flex items-center gap-1.5 text-[11px] text-zinc-400 mt-0.5 truncate">
                  <Folder class="h-3 w-3 shrink-0" />
                  <span class="truncate">{task.output_dir}</span>
                </div>
              </div>
            </div>

            <!-- Status Badge -->
            <div class="shrink-0 flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-semibold {badge.bg}">
              {#if task.status === 'downloading' || task.status === 'extracting'}
                <Loader2 class="h-3.5 w-3.5 animate-spin" />
              {:else if task.status === 'completed'}
                <CheckCircle2 class="h-3.5 w-3.5" />
              {:else if task.status === 'failed'}
                <AlertCircle class="h-3.5 w-3.5" />
              {:else if task.status === 'queued'}
                <Clock class="h-3.5 w-3.5" />
              {/if}
              <span>{badge.label}</span>
            </div>
          </div>

          <!-- Error message if any -->
          {#if task.error_msg}
            <div class="mt-1 rounded-lg bg-rose-500/10 border border-rose-500/20 p-2.5 text-xs text-rose-400">
              {task.error_msg}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
