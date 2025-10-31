<script lang="ts">
  import { onMount } from 'svelte';
  import { API_BASE } from '$lib/api';
  import { toast } from '$lib/toast';

  function authHeaders() {
    const token = localStorage.getItem('aura_token') || '';
    return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } as Record<string, string>;
  }

  type Device = { id: string; org_id?: string; org_name?: string; fingerprint: string; provider?: string; posture_ok: boolean; last_attested_at?: string; posture?: any };
  type Cert = { serial: string; subject: string; device_id?: string; revoked: boolean; not_before: string; not_after: string };

  let devices: Device[] = [];
  let certs: Cert[] = [];
  let loading = false;

  async function loadAll() {
    loading = true;
    try {
      // Use the richer admin detail endpoint when available.
      let d: Device[] = [];
      try {
        const r = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/devices/detail`, { headers: authHeaders() });
        if (r.ok) {
          const j = await r.json();
          d = j.items || [];
        } else {
          // fallback to minimal list
          const r2 = await fetch(`${API_BASE.replace(/\/$/, '')}/admin/devices`, { headers: authHeaders() });
          if (r2.ok) d = await r2.json();
        }
      } catch {}
      devices = d || [];

      const cr = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/certs`, { headers: authHeaders() });
      if (cr.ok) {
        const j = await cr.json();
        certs = j.items || [];
      }
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  let issueDays = 30;
  let subjectCN = '';
  async function issueCert(deviceId: string) {
    if (!subjectCN.trim()) { toast('Enter Subject CN', 'warning'); return; }
    const body = JSON.stringify({ device_id: deviceId, subject_cn: subjectCN.trim(), days: issueDays });
    const r = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/certs/issue`, { method: 'POST', headers: authHeaders(), body });
    if (!r.ok) { toast('Issue failed', 'error'); return; }
    const j = await r.json();
    toast(`Issued serial ${j.serial}`, 'success');
    await loadAll();
  }

  async function revokeCert(serial: string) {
    const r = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/certs/${encodeURIComponent(serial)}/revoke`, { method: 'POST', headers: authHeaders() });
    if (r.status === 204) { toast('Revoked', 'success'); await loadAll(); return; }
    toast('Revoke failed', 'error');
  }

  onMount(loadAll);
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold a-text-gradient a-header">Devices</h1>
    <button on:click={loadAll} class="px-3 py-2 rounded a-gradient text-white hover:opacity-90">Refresh</button>
  </div>

  <div class="a-card a-ribbon flex items-center gap-3">
    <label for="subj" class="text-sm text-gray-400">Subject CN</label>
    <input id="subj" bind:value={subjectCN} class="px-3 py-2 rounded bg-black/30 border border-white/10" placeholder="agent-123@org" />
    <label for="days" class="text-sm text-gray-400">Days</label>
    <input id="days" type="number" min="1" max="365" bind:value={issueDays} class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[100px]" />
  </div>

  <div>
    <h2 class="text-sm font-semibold text-gray-300 mb-2 a-header">Known Devices</h2>
    {#if devices.length === 0}
      <div class="text-gray-400 text-sm">No devices yet. Call POST /v2/attest first.</div>
    {:else}
      <div class="space-y-3">
        {#each devices as d}
          <div class="a-card">
            <div class="text-sm"><span class="font-mono text-xs">{d.id}</span>{#if d.org_name}&nbsp;<span class="text-xs text-gray-400">• {d.org_name}</span>{/if}</div>
            <div class="text-xs text-gray-400 break-all">fp: {d.fingerprint}</div>
            <div class="text-xs">provider: {d.provider || '—'} • posture_ok: {d.posture_ok ? 'true' : 'false'} • last_attested_at: {d.last_attested_at || '—'}</div>
            {#if d.posture}
              <div class="mt-2 text-[11px] text-gray-300 whitespace-pre-wrap break-words bg-black/30 rounded p-2 border border-white/10">
                {JSON.stringify(d.posture).slice(0, 400)}{JSON.stringify(d.posture).length > 400 ? '…' : ''}
              </div>
            {/if}
            <div class="mt-2">
              <button class="px-2 py-1 rounded a-gradient text-white hover:opacity-90 disabled:opacity-60" on:click={() => issueCert(d.id)} disabled={!d.posture_ok}>Issue cert</button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <div>
    <h2 class="text-sm font-semibold text-gray-300 mb-2 a-header">Client Certificates</h2>
    {#if certs.length === 0}
      <div class="text-gray-400 text-sm">No client certs yet.</div>
    {:else}
      <div class="space-y-3">
        {#each certs as c}
          <div class="a-card">
            <div class="text-sm">serial: <span class="font-mono text-xs">{c.serial}</span> • subject: {c.subject}</div>
            <div class="text-xs text-gray-400">device: {c.device_id || '—'} • nb: {c.not_before} • na: {c.not_after}</div>
            <div class="mt-2">
              {#if !c.revoked}
                <button on:click={() => revokeCert(c.serial)} class="px-2 py-1 rounded bg-white/10 hover:bg-white/20">Revoke</button>
              {:else}
                <span class="text-xs text-red-400">revoked</span>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
