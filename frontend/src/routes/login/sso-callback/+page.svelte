<script lang="ts">
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';

  onMount(() => {
    if (!browser) return;
    const url = new URL(window.location.href);
    const token = url.searchParams.get('token');
    const orgId = url.searchParams.get('orgId');
    if (token) localStorage.setItem('aura_token', token);
    if (orgId) localStorage.setItem('aura_org_id', orgId);
  // Redirect to app dashboard
  goto('/dashboard');
  });
</script>

<div class="p-6 text-sm text-gray-500">Completing SSO…</div>
