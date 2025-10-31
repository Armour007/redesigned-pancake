<script lang="ts">
  import { onMount } from 'svelte';
  let posts = [];
  let loading = true;
  let error = '';
  let page = 1;
  let limit = 4;
  let total = 0;

  async function fetchPosts() {
    loading = true;
    error = '';
    try {
      const res = await fetch(`https://api.mocki.io/v2/549a5d8b/BlogPosts?page=${page}&limit=${limit}`);
      if (!res.ok) throw new Error('Failed to fetch blog posts');
      const data = await res.json();
      posts = data.posts || [];
      total = data.total || posts.length;
    } catch (e) {
      error = e.message || 'Error fetching blog posts';
    } finally {
      loading = false;
    }
  }

  onMount(fetchPosts);
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <h1 class="text-3xl sm:text-4xl font-semibold text-white">Blog</h1>
    <p class="mt-2 text-indigo-200">Stories and updates from the AURA team</p>

    {#if loading}
      <div class="mt-8 grid gap-6">
        {#each Array(limit) as _}
          <div class="block rounded-xl ring-1 ring-white/10 bg-white/5 p-6 animate-pulse">
            <div class="h-4 w-24 bg-indigo-900/40 rounded mb-2"></div>
            <div class="h-6 w-48 bg-indigo-900/30 rounded mb-2"></div>
            <div class="h-4 w-64 bg-indigo-900/20 rounded"></div>
          </div>
        {/each}
      </div>
    {:else if error}
      <div class="mt-8 text-red-400">{error}</div>
    {:else}
      <div class="mt-8 grid gap-6">
        {#each posts as p}
          <a class="block rounded-xl ring-1 ring-white/10 bg-white/5 p-6 hover:bg-white/10 transition" href={'/blog/' + p.slug}>
            <div class="text-sm text-indigo-300">{p.date}</div>
            <h2 class="mt-1 text-white font-medium">{p.title}</h2>
            <p class="mt-2 text-indigo-200 text-sm">{p.summary}</p>
          </a>
        {/each}
      </div>
      <!-- Pagination Bar -->
      {#if total > limit}
        <div class="mt-10 flex justify-center gap-2">
          <button class="px-3 py-1 rounded bg-white/10 text-indigo-200 disabled:opacity-40" on:click={() => { page = Math.max(1, page - 1); fetchPosts(); }} disabled={page === 1}>Previous</button>
          <span class="px-3 py-1 text-indigo-300">Page {page} of {Math.ceil(total/limit)}</span>
          <button class="px-3 py-1 rounded bg-white/10 text-indigo-200 disabled:opacity-40" on:click={() => { page = Math.min(Math.ceil(total/limit), page + 1); fetchPosts(); }} disabled={page === Math.ceil(total/limit)}>Next</button>
        </div>
      {/if}
    {/if}
  </div>
</section>
