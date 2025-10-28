<script lang="ts">
  import { onMount } from 'svelte';
  import { API_BASE } from '$lib/api';
  import Modal from '$lib/components/Modal.svelte';
  import Toast from '$lib/components/Toast.svelte';

  type DLQItem = {
    id: string;
    payload: any;
    reason?: string;
    deliveries?: number;
    at?: number; // unix seconds
  };

  let items: DLQItem[] = [];
  let loading = false;
  let error: string | null = null;

  let count = 50;
  let olderThan: string = ''; // ISO string from input[type=datetime-local]
  let beforeId: string | null = null; // for pagination
  let selected: Record<string, boolean> = {};
  let showDetails = false;
  let detailsJson = '';
  let fetchedCount = 0;
  let sentinel: HTMLDivElement | null = null;
  let toastOpen = false;
  let toastMessage = '';
  let toastVariant: 'success' | 'error' | 'info' = 'info';

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
    const url = new URL(`${API_BASE.replace(/\/$/, '')}/admin/queue/dlq`);
    url.searchParams.set('count', String(count));
    const ot = toUnixSeconds(olderThan);
    if (ot) url.searchParams.set('older_than', String(ot));
    if (!reset && beforeId) url.searchParams.set('before_id', beforeId);
    return url.toString();
  }

  async function loadDLQ(reset = true) {
    if (reset) { items = []; beforeId = null; fetchedCount = 0; }
    loading = true; error = null;
    try {
      const res = await fetch(buildURL(reset), { headers: authHeaders() });
      if (!res.ok) throw new Error(`failed: ${res.status}`);
      const data = await res.json();
      const parsed: DLQItem[] = (data.items || []).map((m: any) => ({
        id: m.id,
        payload: parsePayload(m.payload),
        reason: m.reason,
        deliveries: m.deliveries,
        at: m.at,
      }));
      if (reset) items = parsed; else items = [...items, ...parsed];
      fetchedCount += parsed.length;
      selected = {};
        // Update paging cursor using server-provided next_before_id (older cursor)
        beforeId = (data.next_before_id && String(data.next_before_id)) || null;

  function notifyDLQChange() {
    try { window.dispatchEvent(new CustomEvent('dlq:changed')); } catch {}
  }
    } catch (e: any) {
      error = e?.message || 'failed to load DLQ';
    } finally {
      loading = false;
    }
  }

  function parsePayload(p: any): any {
    if (typeof p === 'string') {
      try { return JSON.parse(p); } catch { return p; }
    }
    return p;
  }

  // server-side filtering, no client filter needed

  async function requeueSelected() {
    const ids = Object.keys(selected).filter(k => selected[k]);
    if (ids.length === 0) return;
    await requeue({ ids });
    if (!error) { toastVariant='success'; toastMessage=`Requeued ${ids.length} item(s)`; toastOpen=true; } else { toastVariant='error'; toastMessage=error; toastOpen=true; }
    await loadDLQ(true);
    notifyDLQChange();
  }

  async function requeueAll() {
    await requeue({ all: true, count });
    if (!error) { toastVariant='success'; toastMessage=`Requeued up to ${count} item(s)`; toastOpen=true; } else { toastVariant='error'; toastMessage=error; toastOpen=true; }
    await loadDLQ(true);
    notifyDLQChange();
  }

  async function requeue(body: { ids?: string[]; all?: boolean; count?: number }) {
    error = null;
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/queue/dlq/requeue`, {
        method: 'POST', headers: authHeaders(), body: JSON.stringify(body)
      });
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t || `requeue failed: ${res.status}`);
      }
    } catch (e: any) {
      error = e?.message || 'requeue failed';
    }
  }

  function toggleAll(ev: Event) {
    const checked = (ev.target as HTMLInputElement).checked;
    const map: Record<string, boolean> = {};
    for (const i of items) map[i.id] = checked;
    selected = map;
  }

  function formatDate(at?: number) {
    if (!at) return '—';
    try { return new Date(at * 1000).toLocaleString(); } catch { return String(at); }
  }

  function openDetails(i: DLQItem) {
    detailsJson = JSON.stringify(i.payload, null, 2);
    showDetails = true;
  }

  function downloadPayload(i: DLQItem) {
    const data = JSON.stringify(i.payload, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `dlq_${i.id.replace(/[:]/g,'_')}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  }

  async function deleteSelected() {
    const ids = Object.keys(selected).filter(k => selected[k]);
    if (ids.length === 0) return;
    await deleteDLQ({ ids });
    if (!error) { toastVariant='success'; toastMessage=`Deleted ${ids.length} item(s)`; toastOpen=true; } else { toastVariant='error'; toastMessage=error; toastOpen=true; }
    await loadDLQ(true);
    notifyDLQChange();
  }

  async function deleteRow(id: string) {
    await deleteDLQ({ ids: [id] });
    if (!error) { toastVariant='success'; toastMessage='Deleted 1 item'; toastOpen=true; } else { toastVariant='error'; toastMessage=error; toastOpen=true; }
    await loadDLQ(true);
    notifyDLQChange();
  }

  async function deleteDLQ(body: { ids?: string[]; all?: boolean; count?: number }) {
    error = null;
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/queue/dlq/delete`, {
        method: 'POST', headers: authHeaders(), body: JSON.stringify(body)
      });
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t || `delete failed: ${res.status}`);
      }
    } catch (e: any) {
      error = e?.message || 'delete failed';
    }
  }

  onMount(() => {
    loadDLQ();
    // Infinite scroll (best-effort)
    try {
      const io = new IntersectionObserver((entries) => {
        const e = entries[0];
        if (e && e.isIntersecting && beforeId && !loading) {
          loadDLQ(false);
        }
      }, { rootMargin: '200px' });
      if (sentinel) io.observe(sentinel);
    } catch {}
  });
</script>

<div class="space-y-6">
  <div class="flex items-end gap-4">
    <div>
      <label class="block text-sm text-gray-400" for="cg-count">Count</label>
      <input id="cg-count" type="number" min="1" max="500" bind:value={count} class="bg-[#151515] border border-white/10 rounded px-3 py-2 w-28" />
    </div>
    <div>
      <label class="block text-sm text-gray-400" for="cg-older">Older than</label>
      <input id="cg-older" type="datetime-local" bind:value={olderThan} class="bg-[#151515] border border-white/10 rounded px-3 py-2" />
    </div>
    <button disabled={loading} on:click={() => loadDLQ(true)} class="self-start px-4 py-2 bg-indigo-600 hover:bg-indigo-500 rounded disabled:opacity-50">Refresh</button>
    <div class="flex-1"></div>
    <button on:click={requeueSelected} class="self-start px-4 py-2 bg-emerald-600 hover:bg-emerald-500 rounded">Requeue selected</button>
    <button on:click={deleteSelected} class="self-start px-4 py-2 bg-red-700 hover:bg-red-600 rounded">Delete selected</button>
    <button on:click={requeueAll} class="self-start px-4 py-2 bg-amber-600 hover:bg-amber-500 rounded">Requeue all (within count)</button>
  </div>

  {#if loading}
    <div class="text-gray-400">Loading DLQ…</div>
  {:else if error}
    <div class="text-red-400">{error}</div>
  {:else if items.length === 0}
    <div class="text-gray-400">No DLQ items.</div>
  {:else}
  <div class="text-xs text-gray-400 pb-2">Fetched {fetchedCount} item(s) this session</div>
    <div class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead>
          <tr class="text-left text-gray-400">
            <th class="py-2 px-3"><input type="checkbox" on:change={toggleAll} aria-label="Select all" /></th>
            <th class="py-2 px-3">ID</th>
            <th class="py-2 px-3">Lang</th>
            <th class="py-2 px-3">Reason</th>
            <th class="py-2 px-3">Deliveries</th>
            <th class="py-2 px-3">At</th>
            <th class="py-2 px-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each items as i}
            <tr class="border-t border-white/10">
              <td class="py-2 px-3"><input type="checkbox" bind:checked={selected[i.id]} aria-label={`Select ${i.id}`} /></td>
              <td class="py-2 px-3 font-mono text-xs">{i.id}</td>
              <td class="py-2 px-3">{i.payload?.lang || '—'}</td>
              <td class="py-2 px-3">{i.reason || '—'}</td>
              <td class="py-2 px-3">{i.deliveries ?? '—'}</td>
              <td class="py-2 px-3">{formatDate(i.at)}</td>
              <td class="py-2 px-3">
                <div class="flex gap-2">
                  <button class="px-3 py-1 bg-slate-700 hover:bg-slate-600 rounded" on:click={() => openDetails(i)}>View</button>
                  <button class="px-3 py-1 bg-slate-700 hover:bg-slate-600 rounded" on:click={() => downloadPayload(i)}>Download</button>
                  <button class="px-3 py-1 bg-emerald-700 hover:bg-emerald-600 rounded" on:click={async () => { await requeue({ ids: [i.id] }); if (!error) { toastVariant='success'; toastMessage='Requeued 1 item'; toastOpen=true; } else { toastVariant='error'; toastMessage=error; toastOpen=true; } await loadDLQ(true); notifyDLQChange(); }}>Requeue</button>
                  <button class="px-3 py-1 bg-red-700 hover:bg-red-600 rounded" on:click={() => deleteRow(i.id)}>Delete</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <div class="pt-4">
      <button disabled={loading || !beforeId} class="px-4 py-2 bg-white/10 hover:bg-white/20 rounded disabled:opacity-50" on:click={() => loadDLQ(false)}>Load more (older)</button>
      <div bind:this={sentinel} class="h-1"></div>
    </div>
  {/if}

  <Modal bind:showModal={showDetails} title="DLQ Payload">
    <pre class="text-xs whitespace-pre-wrap break-words">{detailsJson}</pre>
  </Modal>
  <Toast bind:open={toastOpen} bind:message={toastMessage} bind:variant={toastVariant} />
</div>
