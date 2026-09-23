<script lang="ts">
  import { tick } from 'svelte'
  import type { DownloadTask } from './types'
  import { openFolder, fetchTaskLog, retryDownload, deleteDownload } from './api'
  import {
    Download,
    Folder,
    FolderOpen,
    Film,
    Tv,
    Loader2,
    CheckCircle2,
    AlertCircle,
    Clock,
    Terminal,
    Copy,
    Check,
    Search,
    X,
    Gauge,
    Timer,
    FileText,
    Database,
    RotateCcw,
    Trash2,
  } from 'lucide-svelte'

  export let downloads: DownloadTask[] = []

  let selectedTaskForLog: DownloadTask | null = null
  let logSearchQuery = ''
  let autoScroll = true
  let logContainer: HTMLDivElement | null = null
  let copied = false
  let dbLogLines: string[] = []
  let loadingDbLog = false
  let actionLoading: Record<string, boolean> = {}

  async function handleOpenFolder(outputDir: string) {
    try {
      await openFolder(undefined, outputDir)
    } catch (e) {
      console.error('Failed to open folder:', e)
    }
  }

  async function handleRetry(taskId: string) {
    actionLoading[taskId] = true
    try {
      await retryDownload(taskId)
    } catch (e) {
      console.error('Failed to retry task:', e)
    } finally {
      actionLoading[taskId] = false
    }
  }

  async function handleDelete(taskId: string) {
    if (!confirm('Hapus task download ini beserta catatan lognya?')) return
    actionLoading[taskId] = true
    try {
      await deleteDownload(taskId)
      downloads = downloads.filter((d) => d.id !== taskId)
      if (selectedTaskForLog?.id === taskId) {
        selectedTaskForLog = null
      }
    } catch (e) {
      console.error('Failed to delete task:', e)
    } finally {
      actionLoading[taskId] = false
    }
  }

  $: currentModalTask = selectedTaskForLog
    ? downloads.find((d) => d.id === selectedTaskForLog?.id) || selectedTaskForLog
    : null

  $: taskLogs = (currentModalTask?.logs && currentModalTask.logs.length > 0)
    ? currentModalTask.logs
    : dbLogLines

  $: filteredLogs = logSearchQuery.trim()
    ? taskLogs.filter((line) => line.toLowerCase().includes(logSearchQuery.toLowerCase()))
    : taskLogs

  // Auto-scroll effect when new logs arrive using requestAnimationFrame to prevent layout thrashing
  let scrollPending = false
  function scrollToBottom() {
    if (!autoScroll || !logContainer || scrollPending) return
    scrollPending = true
    requestAnimationFrame(() => {
      if (logContainer && autoScroll) {
        logContainer.scrollTop = logContainer.scrollHeight
      }
      scrollPending = false
    })
  }

  $: if (taskLogs.length && autoScroll) {
    scrollToBottom()
  }

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

  async function openLogModal(task: DownloadTask) {
    selectedTaskForLog = task
    logSearchQuery = ''
    autoScroll = true
    dbLogLines = []

    // If task has no logs in memory, fetch directly from SQLite database
    if (!task.logs || task.logs.length === 0) {
      loadingDbLog = true
      try {
        const data = await fetchTaskLog(task.id)
        if (data && data.log) {
          dbLogLines = data.log.split('\n').filter((l: string) => l.trim().length > 0)
        }
      } catch (e) {
        console.error('Failed to load database logs', e)
      } finally {
        loadingDbLog = false
      }
    }

    tick().then(() => {
      if (logContainer) {
        logContainer.scrollTop = logContainer.scrollHeight
      }
    })
  }

  function closeLogModal() {
    selectedTaskForLog = null
    dbLogLines = []
  }

  async function copyLogsToClipboard() {
    if (!currentModalTask) return
    const text = taskLogs.join('\n')
    try {
      await navigator.clipboard.writeText(text)
      copied = true
      setTimeout(() => {
        copied = false
      }, 2000)
    } catch (e) {
      console.error('Failed to copy logs', e)
    }
  }

  function parseLogLine(line: string) {
    // Fast O(1) parse for format: [HH:MM:SS] Message
    if (line.length >= 10 && line.charCodeAt(0) === 91 && line.charCodeAt(9) === 93) {
      return {
        time: line.slice(1, 9),
        rest: line.slice(10).trimStart(),
      }
    }
    return { time: '', rest: line }
  }
</script>

<div class="mx-auto max-w-5xl px-4 py-8">
  <div class="flex items-center justify-between mb-6 flex-wrap gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Download Manager</h2>
      <p class="text-xs text-zinc-400">Background tasks, active streams & execution diagnostics</p>
    </div>
    <div class="flex items-center gap-2">
      <div class="rounded-xl bg-zinc-900 border border-zinc-800 px-3 py-1.5 text-xs font-medium text-zinc-300">
        Total Tasks: <span class="font-bold text-indigo-400">{downloads.length}</span>
      </div>
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
        <div class="flex flex-col gap-3 rounded-2xl border border-zinc-800/80 bg-zinc-900/80 p-4 shadow-sm backdrop-blur-sm transition-all hover:border-zinc-700/60">
          <div class="flex items-start justify-between gap-4">
            <div class="flex items-start gap-3 min-w-0">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 mt-0.5">
                {#if task.media_type === 'series'}
                  <Tv class="h-5 w-5" />
                {:else}
                  <Film class="h-5 w-5" />
                {/if}
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="text-sm font-bold text-white truncate">{task.title}</h3>
                  {#if task.season_num && task.episode_num}
                    <span class="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] font-semibold text-indigo-300 border border-zinc-700">
                      S{String(task.season_num).padStart(2, '0')}E{String(task.episode_num).padStart(2, '0')}
                    </span>
                  {/if}
                  {#if task.year}
                    <span class="text-[11px] text-zinc-400">({task.year})</span>
                  {/if}
                </div>
                <div class="flex items-center gap-1.5 text-[11px] text-zinc-400 mt-0.5 truncate">
                  <span class="font-mono text-zinc-300 truncate">{task.file_name}</span>
                </div>
                <div class="flex items-center gap-1 text-[10px] text-zinc-500 mt-0.5 truncate">
                  <Folder class="h-3 w-3 shrink-0" />
                  <span class="truncate">{task.output_dir}</span>
                </div>
              </div>
            </div>

            <!-- Status & Action Buttons -->
            <div class="shrink-0 flex items-center gap-1.5 flex-wrap justify-end">
              <!-- Open Target Folder -->
              <button
                type="button"
                on:click={() => handleOpenFolder(task.output_dir)}
                class="flex items-center gap-1 rounded-lg border border-zinc-700/70 bg-zinc-800/80 hover:bg-zinc-700 hover:border-zinc-600 px-2 py-1.5 text-xs font-medium text-zinc-300 transition shadow-sm cursor-pointer"
                title="Buka folder tujuan di File Explorer"
              >
                <FolderOpen class="h-3.5 w-3.5 text-sky-400" />
                <span class="hidden sm:inline">Folder</span>
              </button>

              <!-- Diagnostic Logs -->
              <button
                type="button"
                on:click={() => openLogModal(task)}
                class="flex items-center gap-1.5 rounded-lg border border-zinc-700/70 bg-zinc-800/80 hover:bg-zinc-700 hover:border-zinc-600 px-2.5 py-1.5 text-xs font-semibold text-zinc-200 transition shadow-sm cursor-pointer"
                title="Buka Diagnostic Log Terminal"
              >
                <Terminal class="h-3.5 w-3.5 text-indigo-400" />
                <span>Logs</span>
                {#if task.logs && task.logs.length > 0}
                  <span class="rounded-full bg-zinc-900 px-1.5 py-0.2 text-[10px] font-mono text-zinc-400">
                    {task.logs.length}
                  </span>
                {/if}
              </button>

              <!-- Retry Task (if failed or completed) -->
              {#if task.status === 'failed' || task.status === 'completed'}
                <button
                  type="button"
                  on:click={() => handleRetry(task.id)}
                  disabled={actionLoading[task.id]}
                  class="flex items-center gap-1 rounded-lg border border-amber-500/30 bg-amber-500/10 hover:bg-amber-500/20 px-2 py-1.5 text-xs font-medium text-amber-300 transition shadow-sm cursor-pointer disabled:opacity-50"
                  title="Ulangi download task ini dari awal (membersihkan file corrupt/temp otomatis)"
                >
                  <RotateCcw class="h-3.5 w-3.5 {actionLoading[task.id] ? 'animate-spin' : ''}" />
                  <span class="hidden sm:inline">Retry</span>
                </button>
              {/if}

              <!-- Delete Task -->
              <button
                type="button"
                on:click={() => handleDelete(task.id)}
                disabled={actionLoading[task.id]}
                class="flex items-center gap-1 rounded-lg border border-zinc-700/60 bg-zinc-800/60 hover:bg-rose-500/20 hover:border-rose-500/30 hover:text-rose-300 px-2 py-1.5 text-xs font-medium text-zinc-400 transition shadow-sm cursor-pointer disabled:opacity-50"
                title="Hapus task dari daftar dan database"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>

              <!-- Status Badge -->
              <div class="flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-semibold {badge.bg}">
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
          </div>

          <!-- Progress Bar & Metrics -->
          {#if task.status === 'downloading' || task.status === 'extracting' || task.status === 'completed'}
            <div class="flex flex-col gap-1.5 pt-1">
              <div class="flex items-center justify-between text-[11px]">
                <div class="flex items-center gap-3 font-mono">
                  {#if task.status === 'downloading'}
                    <span class="flex items-center gap-1 text-emerald-400 font-semibold">
                      <Gauge class="h-3 w-3" />
                      {task.speed || '0 B/s'}
                    </span>
                    <span class="flex items-center gap-1 text-zinc-400">
                      <Timer class="h-3 w-3" />
                      ETA: {task.eta || '--:--:--'}
                    </span>
                  {:else if task.status === 'extracting'}
                    <span class="text-amber-400">Resolving stream URL & subtitles...</span>
                  {:else if task.status === 'completed'}
                    <span class="text-emerald-400">Download and merge completed</span>
                  {/if}
                </div>
                <span class="font-bold text-zinc-200 font-mono">{task.progress.toFixed(1)}%</span>
              </div>

              <div class="h-2 w-full overflow-hidden rounded-full bg-zinc-800">
                <div
                  class="h-full rounded-full transition-all duration-150 {task.status === 'completed'
                    ? 'bg-emerald-500'
                    : task.status === 'extracting'
                    ? 'bg-amber-500 animate-pulse'
                    : 'bg-emerald-500'}"
                  style="width: {Math.min(100, Math.max(0, task.progress))}%"
                ></div>
              </div>
            </div>
          {/if}

          <!-- Error message if any -->
          {#if task.error_msg}
            <div class="rounded-xl bg-rose-500/10 border border-rose-500/20 p-3 text-xs text-rose-300 flex items-start gap-2.5">
              <AlertCircle class="h-4 w-4 shrink-0 text-rose-400 mt-0.5" />
              <div class="min-w-0 flex-1">
                <div class="font-semibold text-rose-400">Download Error</div>
                <div class="mt-0.5 text-rose-200/90 leading-relaxed break-words">{task.error_msg}</div>
                <div class="mt-2 flex items-center gap-2">
                  <button
                    type="button"
                    on:click={() => openLogModal(task)}
                    class="text-[11px] underline text-rose-300 hover:text-rose-100 transition"
                  >
                    Buka Terminal Logs untuk diagnosa detail &rarr;
                  </button>
                </div>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Terminal Log Modal -->
{#if currentModalTask}
  {@const modalBadge = getStatusBadge(currentModalTask.status)}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4 backdrop-blur-md animate-in fade-in duration-200"
    on:click|self={closeLogModal}
    on:keydown={(e) => e.key === 'Escape' && closeLogModal()}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div
      class="flex h-[85vh] w-full max-w-4xl flex-col rounded-2xl border border-zinc-800 bg-zinc-950 shadow-2xl overflow-hidden"
      role="document"
    >
      <!-- Modal Header -->
      <div class="flex items-center justify-between border-b border-zinc-800/80 bg-zinc-900/90 px-5 py-3.5">
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-zinc-800 text-indigo-400">
            <Terminal class="h-5 w-5" />
          </div>
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <h3 class="text-sm font-bold text-white truncate">{currentModalTask.title}</h3>
              <span class="rounded border px-2 py-0.5 text-[10px] font-semibold {modalBadge.bg}">
                {modalBadge.label}
              </span>
            </div>
            <p class="text-xs font-mono text-zinc-400 truncate">Task ID: {currentModalTask.id}</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <!-- Copy All -->
          <button
            type="button"
            on:click={copyLogsToClipboard}
            class="flex items-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-800 hover:bg-zinc-700 px-2.5 py-1.5 text-xs font-medium text-zinc-200 transition"
            title="Salin seluruh log ke clipboard"
          >
            {#if copied}
              <Check class="h-3.5 w-3.5 text-emerald-400" />
              <span class="text-emerald-400">Tersalin!</span>
            {:else}
              <Copy class="h-3.5 w-3.5" />
              <span class="hidden sm:inline">Salin Log</span>
            {/if}
          </button>

          <!-- Close Modal -->
          <button
            type="button"
            on:click={closeLogModal}
            class="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-white transition"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </div>

      <!-- Modal Toolbar -->
      <div class="flex items-center justify-between border-b border-zinc-800/80 bg-zinc-900/40 px-5 py-2.5 text-xs gap-3 flex-wrap">
        <div class="relative flex-1 min-w-[200px]">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-zinc-500" />
          <input
            type="text"
            bind:value={logSearchQuery}
            placeholder="Cari kata kunci dalam logs (misal: 403, error, m3u8, ffmpeg)..."
            class="w-full rounded-lg border border-zinc-800 bg-zinc-950/80 py-1.5 pl-8 pr-3 text-xs text-white placeholder-zinc-500 focus:border-indigo-500 focus:outline-none"
          />
        </div>

        <div class="flex items-center gap-4 text-zinc-400">
          <label class="flex items-center gap-2 cursor-pointer select-none text-xs">
            <input
              type="checkbox"
              bind:checked={autoScroll}
              class="h-3.5 w-3.5 rounded border-zinc-700 bg-zinc-800 text-indigo-600 focus:ring-0"
            />
            <span>Auto-scroll ke bawah</span>
          </label>

          <span class="font-mono text-zinc-500 text-[11px]">
            {filteredLogs.length} / {taskLogs.length} lines
          </span>
        </div>
      </div>

      <!-- Modal Log Terminal Body -->
      <div
        bind:this={logContainer}
        class="flex-1 overflow-y-auto p-4 font-mono text-xs leading-relaxed select-text bg-zinc-950 text-zinc-300"
      >
        {#if loadingDbLog}
          <div class="flex flex-col items-center justify-center h-full text-zinc-400 text-center py-12 gap-2">
            <Loader2 class="h-6 w-6 animate-spin text-indigo-400" />
            <p class="text-xs">Memuat riwayat log dari database SQLite...</p>
          </div>
        {:else if filteredLogs.length === 0}
          <div class="flex flex-col items-center justify-center h-full text-zinc-600 text-center py-12">
            <FileText class="h-8 w-8 mb-2 opacity-50" />
            {#if logSearchQuery}
              <p>Tidak ada baris log yang cocok dengan "{logSearchQuery}"</p>
            {:else}
              <p>Belum ada rekaman log untuk task ini.</p>
            {/if}
          </div>
        {:else}
          <div class="flex flex-col gap-0.5">
            {#each filteredLogs as line, index}
              {@const parsed = parseLogLine(line)}
              <div class="flex items-start gap-2 hover:bg-zinc-900/60 px-1.5 py-0.5 rounded transition-colors group">
                <span class="w-8 shrink-0 text-right text-[10px] text-zinc-600 select-none group-hover:text-zinc-400">
                  {index + 1}
                </span>

                {#if parsed.time}
                  <span class="shrink-0 text-zinc-500 select-none">[{parsed.time}]</span>
                {/if}

                <div class="min-w-0 flex-1 break-all whitespace-pre-wrap">
                  {#if parsed.rest.includes('[ERROR]') || parsed.rest.includes('[STDERR]') || parsed.rest.includes('403') || parsed.rest.includes('404') || parsed.rest.toLowerCase().includes('failed')}
                    <span class="text-rose-400 font-semibold">{parsed.rest}</span>
                  {:else if parsed.rest.includes('[INFO]') || parsed.rest.includes('[Init]') || parsed.rest.includes('[STATUS]')}
                    <span class="text-sky-300">{parsed.rest}</span>
                  {:else if parsed.rest.includes('[EXTRACT]') || parsed.rest.includes('[STAGE]')}
                    <span class="text-amber-300">{parsed.rest}</span>
                  {:else if parsed.rest.includes('[EXEC]')}
                    <span class="text-purple-300 font-medium">{parsed.rest}</span>
                  {:else if parsed.rest.includes('[SUCCESS]') || parsed.rest.includes('Completed')}
                    <span class="text-emerald-400 font-semibold">{parsed.rest}</span>
                  {:else}
                    <span class="text-zinc-300">{parsed.rest}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Modal Footer -->
      <div class="flex items-center justify-between border-t border-zinc-800/80 bg-zinc-900/90 px-5 py-2.5 text-xs text-zinc-500">
        <div class="flex items-center gap-1.5 truncate">
          <Database class="h-3.5 w-3.5 text-indigo-400" />
          <span>Tersimpan di database SQLite (<code class="text-zinc-400 font-mono">idlix.db</code>)</span>
        </div>
        <button
          type="button"
          on:click={closeLogModal}
          class="rounded-lg bg-zinc-800 hover:bg-zinc-700 px-3 py-1 text-xs font-semibold text-zinc-200 transition"
        >
          Tutup
        </button>
      </div>
    </div>
  </div>
{/if}
