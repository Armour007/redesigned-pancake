<script lang="ts">
  export let data: { keys: any[]; orgId?: string; error?: string };

  import { API_BASE } from '$lib/api';
  import Alert from '$lib/components/Alert.svelte';

  async function revoke(keyId: string) {
    const token = localStorage.getItem('aura_token');
    if (!token || !data.orgId) return;
    const res = await fetch(`${API_BASE}/organizations/${data.orgId}/apikeys/${keyId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` }
    });
    if (res.ok) location.reload();
  }
</script>

<h1 class="text-3xl font-bold mb-6">API Keys</h1>
{#if data?.error}
  <Alert variant="error">{data.error}</Alert>
{/if}

<div class="overflow-x-auto">
  <table class="min-w-full text-left text-sm">
    <thead class="bg-[#1A1A1A] text-gray-300">
      <tr>
        <th class="px-4 py-3">Name</th>
        <th class="px-4 py-3">Prefix</th>
        <th class="px-4 py-3">Created</th>
        <th class="px-4 py-3">Last Used</th>
        <th class="px-4 py-3">Expires</th>
        <th class="px-4 py-3">Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each data.keys as k}
        <tr class="border-b border-white/10">
          <td class="px-4 py-3">{k.name}</td>
          <td class="px-4 py-3">{k.key_prefix}</td>
          <td class="px-4 py-3">{k.created_at}</td>
          <td class="px-4 py-3">{k.last_used_at || '-'}</td>
          <td class="px-4 py-3">{k.expires_at || '-'}</td>
          <td class="px-4 py-3">
            <button class="text-sm text-red-400 hover:text-red-300" on:click={() => revoke(k.id)}>Revoke</button>
          </td>
        </tr>
      {/each}
      {#if !data.keys || data.keys.length === 0}
        <tr><td colspan="6" class="px-4 py-6 text-gray-400">No API keys yet.</td></tr>
      {/if}
    </tbody>
  </table>
</div>
