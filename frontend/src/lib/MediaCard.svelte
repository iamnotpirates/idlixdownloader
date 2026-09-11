<script lang="ts">
  import type { MediaItem } from './types'
  import { Star, Tv, Film } from 'lucide-svelte'

  export let item: MediaItem
  export let onSelect: (item: MediaItem) => void

  function fallbackImage(e: Event) {
    const target = e.target as HTMLImageElement
    target.src = 'https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?auto=format&fit=crop&w=400&q=80'
  }
</script>

<div
  class="group relative flex cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-800/80 bg-zinc-900/60 p-2 transition-all duration-300 hover:-translate-y-1 hover:border-indigo-500/50 hover:shadow-xl hover:shadow-indigo-500/10"
  on:click={() => onSelect(item)}
  on:keydown={(e) => e.key === 'Enter' && onSelect(item)}
  tabindex="0"
  role="button"
>
  <!-- Poster Image -->
  <div class="relative aspect-[2/3] w-full overflow-hidden rounded-xl bg-zinc-950">
    <img
      src={item.poster || 'https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?auto=format&fit=crop&w=400&q=80'}
      alt={item.title}
      loading="lazy"
      on:error={fallbackImage}
      class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
    />

    <!-- Gradient Overlay -->
    <div class="absolute inset-0 bg-gradient-to-t from-zinc-950 via-transparent to-transparent opacity-80" />

    <!-- Badges -->
    <div class="absolute top-2 left-2 flex items-center gap-1.5">
      <span class="inline-flex items-center gap-1 rounded-md bg-zinc-900/85 px-2 py-0.5 text-[10px] font-semibold text-zinc-300 backdrop-blur-md">
        {#if item.type === 'TV Series'}
          <Tv class="h-3 w-3 text-indigo-400" />
        {:else}
          <Film class="h-3 w-3 text-emerald-400" />
        {/if}
        {item.type}
      </span>
    </div>

    <!-- Rating Badge -->
    {#if item.rating && item.rating !== 'N/A'}
      <div class="absolute top-2 right-2 flex items-center gap-1 rounded-md bg-amber-500/90 px-1.5 py-0.5 text-[10px] font-bold text-zinc-950 backdrop-blur-md shadow-sm">
        <Star class="h-3 w-3 fill-zinc-950" />
        <span>{item.rating}</span>
      </div>
    {/if}

    <!-- Year badge on bottom of poster -->
    {#if item.year}
      <div class="absolute bottom-2 left-2 text-[11px] font-medium text-zinc-400">
        {item.year}
      </div>
    {/if}
  </div>

  <!-- Content Info -->
  <div class="mt-2 flex flex-col p-1">
    <h3 class="line-clamp-1 text-xs font-semibold text-zinc-100 group-hover:text-indigo-400 transition-colors" title={item.title}>
      {item.title}
    </h3>
  </div>
</div>
