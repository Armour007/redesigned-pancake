<script lang="ts">
  export let data: { posts: Array<{ slug: string; title: string; date: string; summary?: string; description?: string }> };
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <h1 class="text-3xl sm:text-4xl font-semibold text-white">Blog</h1>
    <p class="mt-2 text-indigo-200">Stories and updates from the AURA team</p>

    <div class="mt-8 grid gap-6">
      {#each data.posts as p}
        <a class="block rounded-xl ring-1 ring-white/10 bg-white/5 p-6 hover:bg-white/10 transition" href={'/blog/' + p.slug}>
          <div class="text-sm text-indigo-300">{p.date}</div>
          <h2 class="mt-1 text-white font-medium">{p.title}</h2>
          <p class="mt-2 text-indigo-200 text-sm">{p.summary || p.description}</p>
        </a>
      {/each}
    </div>
  </div>
</section>
