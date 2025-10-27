<script lang="ts">
  import { onMount } from 'svelte';
  let loading = false;
  let input = JSON.stringify({
    agent_id: 'aura_agent_123',
    request_context: { user_id: 'user_42', action: 'checkout.create' }
  }, null, 2);
  let result: any = null;
  let error: string | null = null;

  async function run() {
    error = null; result = null; loading = true;
    try {
      const res = await fetch('/api/mock/verify', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: input
      });
      if (!res.ok) throw new Error('Request failed');
      result = await res.json();
    } catch (e: any) {
      error = e?.message || 'Unexpected error';
    } finally { loading = false; }
  }
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">Try it — live</h2>
      <p class="mt-3 text-indigo-200">Send a mock verify() request. No auth needed.</p>
    </div>

    <div class="mt-6 grid md:grid-cols-2 gap-6">
      <div class="rounded-xl bg-black/60 ring-1 ring-white/10 p-4">
        <label class="text-xs text-indigo-300">Request (JSON)</label>
        <textarea bind:value={input} class="mt-2 w-full h-56 bg-black/50 text-indigo-100 text-xs p-3 rounded-lg outline-none ring-1 ring-white/10 focus:ring-indigo-500/50"></textarea>
        <button on:click={run} class="mt-3 inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-4 py-2 text-sm" disabled={loading}>{loading ? 'Running…' : 'Run'}</button>
      </div>
      <div class="rounded-xl bg-black/60 ring-1 ring-white/10 p-4">
        <label class="text-xs text-indigo-300">Response</label>
        {#if error}
          <pre class="mt-2 text-xs text-rose-300">{error}</pre>
        {:else if result}
          <pre class="mt-2 text-xs text-emerald-100">{JSON.stringify(result, null, 2)}</pre>
        {:else}
          <div class="mt-2 text-xs text-indigo-300">No response yet</div>
        {/if}
      </div>
    </div>
  </div>
</section>
