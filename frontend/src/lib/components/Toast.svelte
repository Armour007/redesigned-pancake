<script lang="ts">
  export let open = false;
  export let message: string = '';
  export let variant: 'success' | 'error' | 'info' = 'info';
  export let timeout = 2500;
  let hideTimer: ReturnType<typeof setTimeout> | null = null;

  $: if (open) {
    if (hideTimer) clearTimeout(hideTimer);
    hideTimer = setTimeout(() => { open = false; }, timeout);
  }
</script>

{#if open}
  <div class="fixed bottom-4 right-4 z-50">
    <div class={`px-4 py-3 rounded shadow-lg border ${variant === 'success' ? 'bg-emerald-600/90 border-emerald-400' : variant === 'error' ? 'bg-red-700/90 border-red-500' : 'bg-slate-700/90 border-white/10'}`}>
      <div class="text-sm">{message}</div>
    </div>
  </div>
{/if}
