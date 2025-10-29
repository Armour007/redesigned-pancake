<script lang="ts">
  import { API_BASE } from '$lib/api';
  export let data: { me: any; org: any; error?: string };
  import Alert from '$lib/components/Alert.svelte';
  import { onMount } from 'svelte';

  let full_name = data?.me?.full_name || '';
  let org_name = data?.org?.name || '';

  let password = { current: '', next: '' };
  let savingProfile = false, savingOrg = false, savingPassword = false;
  let successMessage = '';
  let errorMessage = '';
  let orgId: string | null = null;

  onMount(() => {
    try { orgId = localStorage.getItem('aura_org_id'); } catch {}
  });

  async function saveProfile() {
    successMessage = '';
    errorMessage = '';
    savingProfile = true;
    try {
      const token = localStorage.getItem('aura_token');
      const res = await fetch(`${API_BASE}/me`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ full_name })
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      successMessage = 'Profile updated';
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to update profile';
    } finally {
      savingProfile = false;
    }
  }

  async function savePassword() {
    successMessage = '';
    errorMessage = '';
    savingPassword = true;
    try {
      const token = localStorage.getItem('aura_token');
      const res = await fetch(`${API_BASE}/me/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ current_password: password.current, new_password: password.next })
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      successMessage = 'Password updated';
      password = { current: '', next: '' };
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to update password';
    } finally {
      savingPassword = false;
    }
  }

  async function saveOrg() {
    successMessage = '';
    errorMessage = '';
    savingOrg = true;
    try {
      const token = localStorage.getItem('aura_token');
      const orgId = localStorage.getItem('aura_org_id');
      const res = await fetch(`${API_BASE}/organizations/${orgId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ name: org_name })
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      successMessage = 'Organization updated';
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to update organization';
    } finally {
      savingOrg = false;
    }
  }
</script>

<h1 class="text-3xl font-bold mb-6">Settings</h1>
{#if data?.error}
  <Alert variant="error">{data.error}</Alert>
{/if}
{#if errorMessage}
  <Alert variant="error">{errorMessage}</Alert>
{/if}
{#if successMessage}
  <Alert variant="success">{successMessage}</Alert>
{/if}

<div class="grid grid-cols-1 xl:grid-cols-2 gap-8">
  <section class="bg-[#151515] rounded-lg border border-white/10 p-6">
    <h2 class="text-xl font-semibold mb-4">Profile</h2>
    <div class="space-y-4">
      <div>
        <label for="full_name" class="block text-sm text-gray-300 mb-1">Full name</label>
        <input id="full_name" class="w-full bg-[#0f0f0f] border border-white/10 rounded px-3 py-2" bind:value={full_name} />
      </div>
      <div>
        <label for="email" class="block text-sm text-gray-300 mb-1">Email</label>
        <input id="email" class="w-full bg-[#0f0f0f] border border-white/10 rounded px-3 py-2 opacity-60" value={data?.me?.email || ''} disabled />
      </div>
      <button class="px-4 py-2 rounded bg-[#7C3AED] hover:bg-[#6B2FD3]" on:click|preventDefault={saveProfile} disabled={savingProfile}>
        {savingProfile ? 'Saving…' : 'Save profile'}
      </button>
    </div>
  </section>

  <section class="bg-[#151515] rounded-lg border border-white/10 p-6">
    <h2 class="text-xl font-semibold mb-4">Change password</h2>
    <div class="space-y-4">
      <div>
        <label for="current_pw" class="block text-sm text-gray-300 mb-1">Current password</label>
        <input id="current_pw" type="password" class="w-full bg-[#0f0f0f] border border-white/10 rounded px-3 py-2" bind:value={password.current} />
      </div>
      <div>
        <label for="new_pw" class="block text-sm text-gray-300 mb-1">New password</label>
        <input id="new_pw" type="password" class="w-full bg-[#0f0f0f] border border-white/10 rounded px-3 py-2" bind:value={password.next} />
      </div>
      <button class="px-4 py-2 rounded bg-[#7C3AED] hover:bg-[#6B2FD3]" on:click|preventDefault={savePassword} disabled={savingPassword}>
        {savingPassword ? 'Updating…' : 'Update password'}
      </button>
    </div>
  </section>

  <section class="bg-[#151515] rounded-lg border border-white/10 p-6 xl:col-span-2">
    <h2 class="text-xl font-semibold mb-4">Organization</h2>
    <div class="space-y-4 max-w-xl">
      <div>
        <label for="org_name" class="block text-sm text-gray-300 mb-1">Organization name</label>
        <input id="org_name" class="w-full bg-[#0f0f0f] border border-white/10 rounded px-3 py-2" bind:value={org_name} />
      </div>
      <button class="px-4 py-2 rounded bg-[#7C3AED] hover:bg-[#6B2FD3]" on:click|preventDefault={saveOrg} disabled={savingOrg}>
        {savingOrg ? 'Saving…' : 'Save organization'}
      </button>
    </div>
  </section>

  <section class="bg-[#151515] rounded-lg border border-white/10 p-6 xl:col-span-2">
    <h2 class="text-xl font-semibold mb-4">Organization admin</h2>
    {#if orgId}
      <div class="grid sm:grid-cols-2 gap-4">
        <a class="block p-4 rounded border border-white/10 hover:bg-white/5" href={`/organizations/${orgId}/webhooks`}>
          <div class="font-medium">Webhook endpoints</div>
          <div class="text-sm text-gray-400 mt-1">Manage outbound webhooks for your org.</div>
        </a>
        <a class="block p-4 rounded border border-white/10 hover:bg-white/5" href={`/organizations/${orgId}/trust-keys`}>
          <div class="font-medium">Trust keys (JWKS)</div>
          <div class="text-sm text-gray-400 mt-1">Rotate and manage Ed25519 trust keys.</div>
        </a>
      </div>
    {:else}
      <div class="text-gray-400 text-sm">No organization selected.</div>
    {/if}
  </section>
</div>
