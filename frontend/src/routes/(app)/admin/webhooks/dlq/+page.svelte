<script lang="ts">
  import { onMount } from 'svelte';
  import { API_BASE } from '$lib/api';
  import Toast from '$lib/components/Toast.svelte';

  type Item = {
    id: string;
    org_id?: string;
    endpoint?: string;
    url?: string;
    event?: string;
    attempts?: number | string;
    last_code?: number | string;
    at?: number | string;
  };

  let items: Item[] = [];
  let loading = false;
  let error: string | null = null;
  let count = 50;
  let olderThan: string = '';
  let beforeId: string | null = null;
  let selected: Record<string, boolean> = {};
  let toastOpen = false;
  let toastMessage = '';
  let toastVariant: 'success' | 'error' | 'info' = 'info';
  let fetchedCount = 0;
  let sentinel: HTMLDivElement | null = null;

  function authHeaders() {
    const token = localStorage.getItem('aura_token') || '';
    return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
  }

  function toUnixSeconds(dt: string): number | null {
    if (!dt) return null;
    const ms = Date.parse(dt);
    if (Number.isNaN(ms)) return null;
    return Math.floor(ms / 1000);
  }

  function buildURL(reset = false): string {
    const url = new URL(`${API_BASE.replace(/\/$/, '')}/admin/webhooks/dlq`);
    url.searchParams.set('count', String(count));
    const ot = toUnixSeconds(olderThan);
    if (ot) url.searchParams.set('older_than', String(ot));
    if (!reset && beforeId) url.searchParams.set('before_id', beforeId);
    return url.toString();
  }

  async function loadList(reset = true) {
    if (reset) { items = []; beforeId = null; fetchedCount = 0; }
    loading = true; error = null;
    try {
      const res = await fetch(buildURL(reset), { headers: authHeaders() });
      if (!res.ok) throw new Error(`failed: ${res.status}`);
      const data = await res.json();
      const parsed: Item[] = (data.items || []) as Item[];
      if (reset) items = parsed; else items = [...items, ...parsed];
      fetchedCount += parsed.length;
      beforeId = (data.next_before_id && String(data.next_before_id)) || null;
      selected = {};
    } catch (e: any) {
      error = e?.message || 'failed to load webhooks DLQ';
    } finally {
      loading = false;
    }
  }

  async function requeue(body: { ids?: string[]; all?: boolean; count?: number }) {
    error = null;
    const res = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/webhooks/dlq/requeue`, {
      method: 'POST', headers: authHeaders(), body: JSON.stringify(body)
    });
    if (!res.ok) {
      const t = await res.text();
      throw new Error(t || `requeue failed: ${res.status}`);
    }
  }

  async function removeDLQ(body: { ids?: string[]; all?: boolean; count?: number }) {
    error = null;
    const res = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/webhooks/dlq/delete`, {
      method: 'POST', headers: authHeaders(), body: JSON.stringify(body)
    });
    if (!res.ok) {
      const t = await res.text();
      throw new Error(t || `delete failed: ${res.status}`);
    }
  }

  async function requeueSelected() {
    const ids = Object.keys(selected).filter(k => selected[k]);
    if (ids.length === 0) return;
    try {
      await requeue({ ids });
      toastVariant = 'success'; toastMessage = `Requeued ${ids.length} item(s)`; toastOpen = true;
    } catch (e: any) { error = e?.message || 'requeue failed'; }
    await loadList();
    try { window.dispatchEvent(new CustomEvent('dlq:changed')); } catch {}
  }

  async function deleteSelected() {
    const ids = Object.keys(selected).filter(k => selected[k]);
    if (ids.length === 0) return;
    try {
      await removeDLQ({ ids });
      toastVariant = 'success'; toastMessage = `Deleted ${ids.length} item(s)`; toastOpen = true;
    } catch (e: any) { error = e?.message || 'delete failed'; }
    await loadList();
    try { window.dispatchEvent(new CustomEvent('dlq:changed')); } catch {}
  }

  function toggleAll(ev: Event) {
    const checked = (ev.target as HTMLInputElement).checked;
    const map: Record<string, boolean> = {};
    for (const i of items) map[i.id] = checked;
    selected = map;
  }

  function formatDate(at: number | string | undefined) {
    if (!at) return '—';
    const n = typeof at === 'string' ? parseInt(at, 10) : at;
    if (!n || Number.isNaN(n)) return String(at);
    try { return new Date((n as number) * 1000).toLocaleString(); } catch { return String(at); }
  }

  onMount(() => {
    loadList();
    // Infinite scroll
    try {
      const io = new IntersectionObserver((entries) => {
        const e = entries[0];
        if (e && e.isIntersecting && beforeId && !loading) {
          loadList(false);
        }
      }, { rootMargin: '200px' });
      if (sentinel) io.observe(sentinel);
    } catch {}
  });
</script>

<div class="space-y-6">
  <h1 class="text-3xl font-bold a-text-gradient a-header">Webhooks DLQ</h1>
  <div class="flex items-end gap-4 a-card a-ribbon">
    <div>
      <label class="block text-sm text-gray-400" for="wh-count">Count</label>
      <input id="wh-count" type="number" min="1" max="500" bind:value={count} class="bg-[#151515] border border-white/10 rounded px-3 py-2 w-28" />
    </div>
    <div>
      <label class="block text-sm text-gray-400" for="wh-older">Older than</label>
      <input id="wh-older" type="datetime-local" bind:value={olderThan} class="bg-[#151515] border border-white/10 rounded px-3 py-2" />
    </div>
  <button disabled={loading} on:click={() => loadList(true)} class="self-start px-4 py-2 bg-indigo-600 hover:bg-indigo-500 rounded disabled:opacity-50">Refresh</button>
  <div class="flex-1"></div>
    <button on:click={requeueSelected} class="self-start px-4 py-2 bg-emerald-600 hover:bg-emerald-500 rounded">Requeue selected</button>
    <button on:click={deleteSelected} class="self-start px-4 py-2 bg-red-700 hover:bg-red-600 rounded">Delete selected</button>
  </div>

  {#if loading}
    <div class="text-gray-400">Loading Webhooks DLQ…</div>
  {:else if error}
    <div class="text-red-400">{error}</div>
  {:else if items.length === 0}
    <div class="text-gray-400">No webhook DLQ items.</div>
  {:else}
    <div class="text-xs text-gray-400 pb-2">Fetched {fetchedCount} item(s) this session</div>
    <div class="overflow-x-auto a-card a-ribbon">
      <table class="min-w-full text-sm">
        <thead>
          <tr class="text-left text-gray-400">
            <th class="py-2 px-3"><input type="checkbox" on:change={toggleAll} aria-label="Select all" /></th>
            <th class="py-2 px-3">ID</th>
            <th class="py-2 px-3">Endpoint</th>
            <th class="py-2 px-3">Event</th>
            <th class="py-2 px-3">Attempts</th>
            <th class="py-2 px-3">Last code</th>
            <th class="py-2 px-3">At</th>
            <th class="py-2 px-3">URL</th>
            <th class="py-2 px-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each items as i}
            <tr class="border-t border-white/10">
              <td class="py-2 px-3"><input type="checkbox" bind:checked={selected[i.id]} aria-label={`Select ${i.id}`} /></td>
              <td class="py-2 px-3 font-mono text-xs">{i.id}</td>
              <td class="py-2 px-3">
                <div class="flex items-center gap-2">
                  <span>{i.endpoint || '—'}</span>
                  {#if i.org_id && i.endpoint}
                    <a class="text-indigo-400 hover:text-indigo-300 underline text-xs" href={`/organizations/${i.org_id}/webhooks?endpointId=${i.endpoint}`} title="Open endpoint page">Open</a>
                  {/if}
                </div>
              </td>
              <td class="py-2 px-3">{i.event || '—'}</td>
              <td class="py-2 px-3">{i.attempts ?? '—'}</td>
              <td class="py-2 px-3">{i.last_code ?? '—'}</td>
              <td class="py-2 px-3">{formatDate(i.at)}</td>
              <td class="py-2 px-3 max-w-[320px] truncate" title={i.url || ''}>{i.url || '—'}</td>
              <td class="py-2 px-3">
                <div class="flex gap-2">
                  <button class="px-3 py-1 bg-emerald-700 hover:bg-emerald-600 rounded" on:click={async () => { try { await requeue({ ids: [i.id] }); toastVariant='success'; toastMessage='Requeued 1 item'; toastOpen=true; } catch (e:any) { error = e?.message || 'requeue failed'; toastVariant='error'; toastMessage='Requeue failed'; toastOpen=true; } await loadList(); try { window.dispatchEvent(new CustomEvent('dlq:changed')); } catch {} }}>Requeue</button>
                  <button class="px-3 py-1 bg-red-700 hover:bg-red-600 rounded" on:click={async () => { try { await removeDLQ({ ids: [i.id] }); toastVariant='success'; toastMessage='Deleted 1 item'; toastOpen=true; } catch (e:any) { error = e?.message || 'delete failed'; toastVariant='error'; toastMessage='Delete failed'; toastOpen=true; } await loadList(); try { window.dispatchEvent(new CustomEvent('dlq:changed')); } catch {} }}>Delete</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <div class="pt-4">
      <button disabled={loading || !beforeId} class="px-4 py-2 bg-white/10 hover:bg-white/20 rounded disabled:opacity-50" on:click={() => loadList(false)}>Load more (older)</button>
      <div bind:this={sentinel} class="h-1"></div>
    </div>
  {/if}

  <Toast bind:open={toastOpen} bind:message={toastMessage} bind:variant={toastVariant} />
</div>
