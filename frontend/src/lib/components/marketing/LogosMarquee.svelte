<script lang="ts">
  import { logos } from '$lib/marketing/logos';
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-10">
    <div class="text-center text-indigo-300 text-xs">Trusted by forward‑thinking teams</div>
    <div class="mt-5 overflow-hidden">
      <div class="flex gap-10 animate-[scroll_40s_linear_infinite] whitespace-nowrap opacity-80">
        {#each logos as l}
          <img src={l.src} alt={l.alt + ' logo'} class="h-8 w-auto opacity-80 hover:opacity-100 transition" />
        {/each}
        {#each logos as l}
          <img src={l.src} alt={l.alt + ' logo'} class="h-8 w-auto opacity-80 hover:opacity-100 transition" />
        {/each}
      </div>
    </div>
  </div>
</section>

<style>
  @keyframes scroll {
    from { transform: translateX(0); }
    to { transform: translateX(-50%); }
  }
</style>
