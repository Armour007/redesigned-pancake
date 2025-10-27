<script lang="ts">
  import { posts } from '$lib/blog/posts';
  export let data: { post: (typeof posts)[number] | null };
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    {#if data.post}
      <div class="text-sm text-indigo-300">{data.post.date}</div>
      <h1 class="mt-1 text-3xl sm:text-4xl font-semibold text-white">{data.post.title}</h1>
      <article class="prose prose-invert prose-indigo mt-6 max-w-none">
        {@html data.post.html}
      </article>
    {:else}
      <h1 class="text-3xl sm:text-4xl font-semibold text-white">Post not found</h1>
    {/if}
  </div>
</section>
