<script lang="ts">
  import { toasts, type ToastItem } from '$lib/toast';
  let list: ToastItem[] = [];
  const unsub = toasts.subscribe((v) => (list = v));
  import { onDestroy } from 'svelte';
  onDestroy(() => unsub());

  function color(variant: ToastItem['variant']) {
    switch (variant) {
      case 'success': return 'bg-emerald-600/90';
      case 'error': return 'bg-red-600/90';
      case 'warning': return 'bg-amber-600/90';
      default: return 'bg-slate-800/90';
    }
  }
</script>

<div class="fixed bottom-4 right-4 z-[1000] space-y-2 w-80">
  {#each list as t (t.id)}
    <div class={`text-sm text-white px-3 py-2 rounded shadow-lg border border-white/10 ${color(t.variant)}`}
      role="status" aria-live="polite">
      {t.message}
    </div>
  {/each}
  <style>
    /* scope reserved for potential keyframe/animation later */
  </style>
</div>
