<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { API_BASE } from '$lib/api';

  type Endpoint = { id: string; url: string; description?: string };
  let endpoints: Endpoint[] = [];
  let loading = false;
  let error: string | null = null;

  let focusId: string | null = null;

  function authHeaders() {
    const token = localStorage.getItem('aura_token') || '';
    return { Authorization: `Bearer ${token}` };
  }

  async function load() {
    loading = true; error = null;
    try {
      const orgId = $page.params.orgId;
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/webhooks`, { headers: authHeaders() });
      if (!res.ok) throw new Error(`failed: ${res.status}`);
      const data = await res.json();
      endpoints = (data.items || data || []) as Endpoint[];
      // Focus after DOM paint
      setTimeout(() => {
        const url = new URL(location.href);
        focusId = url.searchParams.get('endpointId');
        if (focusId) {
          const el = document.getElementById(`endpoint-${focusId}`);
          if (el) { el.scrollIntoView({ behavior: 'smooth', block: 'center' }); el.classList.add('ring-2','ring-indigo-500'); setTimeout(() => el.classList.remove('ring-2','ring-indigo-500'), 2500); }
        }
      }, 0);
    } catch (e: any) {
      error = e?.message || 'failed to load endpoints';
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-lg font-semibold">Webhook Endpoints</h1>
    <button disabled={loading} on:click={load} class="px-3 py-2 rounded bg-white/10 hover:bg-white/20 disabled:opacity-50">Refresh</button>
  </div>

  {#if loading}
    <div class="text-gray-400">Loading…</div>
  {:else if error}
    <div class="text-red-400">{error}</div>
  {:else if endpoints.length === 0}
    <div class="text-gray-400">No endpoints found.</div>
  {:else}
    <div class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead>
          <tr class="text-left text-gray-400">
            <th class="py-2 px-3">ID</th>
            <th class="py-2 px-3">URL</th>
            <th class="py-2 px-3">Description</th>
          </tr>
        </thead>
        <tbody>
          {#each endpoints as ep}
            <tr id={`endpoint-${ep.id}`} class="border-t border-white/10">
              <td class="py-2 px-3 font-mono text-xs">{ep.id}</td>
              <td class="py-2 px-3 max-w-[420px] truncate" title={ep.url}>{ep.url}</td>
              <td class="py-2 px-3">{ep.description || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
