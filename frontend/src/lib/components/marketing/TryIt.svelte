<script lang="ts">
  import { onMount } from 'svelte';
  let loading = false;
  let inputObj: any = {
    agent_id: 'aura_agent_123',
    request_context: { user_id: 'user_42', action: 'checkout.create', risk_signals: {} }
  };
  let result: any = null;
  let error: string | null = null;
  let history: Array<{ t: number; allow: boolean; latency: number }> = [];

  let rsKey = '';
  let rsVal = '';

  $: avgLatency = history.length ? Math.round(history.reduce((s,h)=> s + h.latency, 0) / history.length) : 0;
  $: allowRate = history.length ? Math.round((history.filter(h=>h.allow).length / history.length) * 100) : 0;

  function toPretty(v: any) { return JSON.stringify(v, null, 2); }
  function parseMaybeNumber(v: string): any { const n = Number(v); return isFinite(n) ? n : v; }

  function addSignal() {
    if (!rsKey) return;
    inputObj.request_context.risk_signals[rsKey] = parseMaybeNumber(rsVal);
    rsKey = ''; rsVal = '';
  }

  async function run() {
    error = null; result = null; loading = true;
    const start = Date.now();
    try {
      const res = await fetch('/api/mock/verify', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(inputObj)
      });
      if (!res.ok) throw new Error('Request failed');
      result = await res.json();
      history = [
        ...history.slice(-49),
        { t: Date.now(), allow: result.decision === 'allow', latency: result.latency_ms ?? (Date.now()-start) }
      ];
    } catch (e: any) {
      error = e?.message || 'Unexpected error';
    } finally { loading = false; }
  }
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">Try it — live</h2>
      <p class="mt-3 text-indigo-200">Send a mock verify() request. Add custom risk signals below.</p>
    </div>

    <div class="mt-6 grid md:grid-cols-2 gap-6">
      <div class="rounded-xl bg-black/60 ring-1 ring-white/10 p-4">
        <label class="text-xs text-indigo-300" for="req-json">Request (JSON)</label>
        <textarea id="req-json" class="mt-2 w-full h-56 bg-black/50 text-indigo-100 text-xs p-3 rounded-lg outline-none ring-1 ring-white/10 focus:ring-indigo-500/50" readonly>{toPretty(inputObj)}</textarea>
        <div class="mt-3 grid grid-cols-3 gap-2 items-end">
          <div class="col-span-1">
            <label class="text-[11px] text-indigo-300" for="rs-key">Signal key</label>
            <input id="rs-key" class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" bind:value={rsKey} placeholder="velocity" />
          </div>
          <div class="col-span-1">
            <label class="text-[11px] text-indigo-300" for="rs-val">Signal value</label>
            <input id="rs-val" class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" bind:value={rsVal} placeholder="0.82" />
          </div>
          <div class="col-span-1">
            <button on:click={addSignal} class="w-full inline-flex items-center justify-center rounded-lg bg-white/10 hover:bg-white/20 ring-1 ring-white/20 text-white px-4 py-2 text-sm">Add</button>
          </div>
        </div>
        <button on:click={run} class="mt-3 inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-4 py-2 text-sm" disabled={loading}>{loading ? 'Running…' : 'Run'}</button>
      </div>
      <div class="rounded-xl bg-black/60 ring-1 ring-white/10 p-4">
        <div class="text-xs text-indigo-300">Response</div>
        {#if error}
          <pre class="mt-2 text-xs text-rose-300">{error}</pre>
        {:else if result}
          <pre class="mt-2 text-xs text-emerald-100">{toPretty(result)}</pre>
        {:else}
          <div class="mt-2 text-xs text-indigo-300">No response yet</div>
        {/if}

        <div class="mt-4 grid grid-cols-2 gap-3">
          <div class="rounded-lg bg-white/5 ring-1 ring-white/10 p-3">
            <div class="text-xs text-indigo-300">Allow rate</div>
            <div class="text-white text-xl font-semibold">{allowRate}%</div>
          </div>
          <div class="rounded-lg bg-white/5 ring-1 ring-white/10 p-3">
            <div class="text-xs text-indigo-300">Avg latency</div>
            <div class="text-white text-xl font-semibold">{avgLatency}ms</div>
          </div>
        </div>

        {#if history.length}
          <div class="mt-4">
            <div class="text-[11px] text-indigo-300">Last {history.length} decisions</div>
            <svg class="mt-2 w-full h-20" viewBox="0 0 200 80" preserveAspectRatio="none">
              <defs>
                <linearGradient id="g" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="#6366f1" stop-opacity="0.6" />
                  <stop offset="100%" stop-color="#6366f1" stop-opacity="0" />
                </linearGradient>
              </defs>
              {#if history.length > 1}
                <polyline fill="none" stroke="#818cf8" stroke-width="2" points={history.map((h,i)=> `${(i/(Math.max(1,history.length-1)))*200},${80 - Math.min(78, h.latency)}` ).join(' ')} />
                <polygon fill="url(#g)" points={`${history.map((h,i)=> `${(i/(Math.max(1,history.length-1)))*200},${80 - Math.min(78, h.latency)}` ).join(' ')} 200,80 0,80`} />
              {/if}
              {#each history as h, i}
                <circle cx={(i/(Math.max(1,history.length-1)))*200} cy={80 - Math.min(78, h.latency)} r="2" fill={h.allow ? '#10b981' : '#ef4444'} />
              {/each}
            </svg>
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>
