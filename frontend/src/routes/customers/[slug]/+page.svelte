<script lang="ts">
  export let data: { cs: ({ slug: string; name: string; logo: string; headline: string; summary: string; metrics?: { label: string; value: string }[]; bodyHtml?: string; component?: any } | null) };
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    {#if data.cs}
      <div class="flex items-center gap-3">
        <img src={data.cs.logo} alt={data.cs.name + ' logo'} class="h-7 w-auto" />
        <h1 class="text-3xl sm:text-4xl font-semibold text-white">{data.cs.name}</h1>
      </div>
      <p class="mt-2 text-indigo-200">{data.cs.headline}</p>
      <div class="mt-6 grid sm:grid-cols-3 gap-4">
        {#each data.cs.metrics as m}
          <div class="rounded-xl ring-1 ring-white/10 bg-white/5 p-4">
            <div class="text-white text-xl font-semibold">{m.value}</div>
            <div class="text-indigo-200 text-sm">{m.label}</div>
          </div>
        {/each}
      </div>
      {#if data.cs.component}
        <article class="prose prose-invert prose-indigo mt-8 max-w-none">
          <svelte:component this={data.cs.component} />
        </article>
      {:else}
        <article class="prose prose-invert prose-indigo mt-8 max-w-none">
          {@html data.cs.bodyHtml}
        </article>
      {/if}
    {:else}
      <h1 class="text-3xl sm:text-4xl font-semibold text-white">Case study not found</h1>
    {/if}
  </div>
</section>
