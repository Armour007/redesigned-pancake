<script lang="ts">
  import { navigating } from '$app/stores';
  $: isNav = $navigating != null;
  // force reflow for progressive feel
  $: key = isNav ? Date.now() : '';
</script>

{#if isNav}
  {#key key}
    <div class="fixed left-0 right-0 top-0 z-[60]" aria-live="polite" aria-atomic="true">
      <div class="h-[2px] w-full bg-transparent">
        <div class="h-full w-0 animate-aura-progress bg-[linear-gradient(90deg,#33C3FF_0%,#8A4DFF_100%)] shadow-[0_0_12px_rgba(138,77,255,0.6)]"></div>
      </div>
    </div>
  {/key}
{/if}

<style>
  @keyframes aura-progress {
    0% { width: 0%; opacity: 1; }
    60% { width: 80%; opacity: 1; }
    100% { width: 100%; opacity: 1; }
  }
  .animate-aura-progress { animation: aura-progress 1.2s cubic-bezier(.4,0,.2,1) infinite; }
</style>
