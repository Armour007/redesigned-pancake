<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { API_BASE, authHeaders } from '$lib/api';
  import { browser } from '$app/environment';

  let cron = '0 2 * * *';
  let destType: 'webhook' | 'file' = 'file';
  let dest = '';
  let format: 'json' | 'csv' = 'json';
  let lookback = '720h';
  let loading = false;
  let status: string = '';

  let token: string | null = null;

  let orgIdFromStore: string | undefined;
  let unsub: (() => void) | undefined;

  onMount(async () => {
    try { unsub = page.subscribe((v: any) => { orgIdFromStore = v?.params?.orgId; }); } catch {}
    if (!browser) return;
    token = localStorage.getItem('aura_token');
    if (!token) return;
    const orgId = orgIdFromStore ?? (localStorage.getItem('aura_org_id') || undefined);
    if (!orgId) return;
    const res = await fetch(`${API_BASE}/organizations/${orgId}/regulator/audit-export/schedule`, {
      headers: authHeaders(token)
    });
    if (res.ok) {
      const data = await res.json();
      if (data?.schedule) {
        cron = data.schedule.cron || cron;
        destType = data.schedule.destType || destType;
        dest = data.schedule.dest || dest;
        format = data.schedule.format || format;
        lookback = data.schedule.lookback || lookback;
      }
    }
  });

  async function save() {
    if (!browser || !token) return;
    loading = true;
    status = '';
    const orgId = orgIdFromStore ?? (localStorage.getItem('aura_org_id') || undefined);
    if (!orgId) { loading = false; status = 'Failed to save'; return; }
    const body = { cron, dest_type: destType, dest, format, lookback };
    const res = await fetch(`${API_BASE}/organizations/${orgId}/regulator/audit-export/schedule`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders(token!) },
      body: JSON.stringify(body)
    });
    loading = false;
    status = res.ok ? 'Saved' : 'Failed to save';
  }

  onDestroy(() => { try { unsub?.(); } catch {} });
</script>

<div class="p-6 space-y-4">
  <h1 class="text-xl font-semibold">Scheduled Audit Export</h1>
  <div class="grid gap-2 max-w-xl">
    <label class="text-sm" for="cron">Cron</label>
    <input id="cron" class="border rounded p-2" bind:value={cron} placeholder="0 2 * * *" />

    <label class="text-sm" for="destType">Destination Type</label>
    <select id="destType" class="border rounded p-2" bind:value={destType}>
      <option value="file">File</option>
      <option value="webhook">Webhook</option>
    </select>

  <label class="text-sm" for="dest">Destination</label>
  <input id="dest" class="border rounded p-2" bind:value={dest} placeholder="/tmp/exports or https://example.com/webhook" />

  <label class="text-sm" for="format">Format</label>
  <select id="format" class="border rounded p-2" bind:value={format}>
      <option value="json">JSON (ZIP)</option>
      <option value="csv" disabled>CSV (ZIP)</option>
    </select>

  <label class="text-sm" for="lookback">Lookback (duration)</label>
  <input id="lookback" class="border rounded p-2" bind:value={lookback} placeholder="720h" />

    <button class="bg-blue-600 text-white px-4 py-2 rounded" on:click|preventDefault={save} disabled={loading}>
      {loading ? 'Saving…' : 'Save'}
    </button>
    {#if status}
      <div class="text-sm text-gray-600">{status}</div>
    {/if}
  </div>
</div>
