<script lang="ts">
  export let data: { caseStudies: Array<{ slug: string; name: string; logo: string; headline: string; summary: string }> };
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <h1 class="text-3xl sm:text-4xl font-semibold text-white">Customers</h1>
    <p class="mt-2 text-indigo-200">How teams use AURA to ship trust decisions faster.</p>

    <div class="mt-8 grid sm:grid-cols-2 gap-6">
      {#each data.caseStudies as c}
        <a class="block rounded-xl ring-1 ring-white/10 bg-white/5 p-6 hover:bg-white/10 transition" href={'/customers/' + c.slug}>
          <div class="flex items-center gap-3">
            <img src={c.logo} alt={c.name + ' logo'} class="h-6 w-auto" />
            <div class="text-white font-medium">{c.name}</div>
          </div>
          <div class="mt-2 text-indigo-200 text-sm">{c.headline}</div>
        </a>
      {/each}
    </div>
  </div>
</section>
