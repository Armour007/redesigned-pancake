<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import SDKQuickStartDrawer from '$lib/components/SDKQuickStartDrawer.svelte';
  import { onboarding } from '$lib/stores/onboarding';
  import { get } from 'svelte/store';

  let agentId = '';
  let keyPrefix = '';
  let secretKeyMasked = '';
  let open = false;

  onMount(() => {
    onboarding.init();
    const url = new URL(window.location.href);
    agentId = url.searchParams.get('agent_id') || '';
    keyPrefix = url.searchParams.get('key_prefix') || '';
    // Masked preview for safety; full key is only shown once at creation time.
    secretKeyMasked = keyPrefix ? `${keyPrefix}********` : '';

    // First-time users only: show the drawer if onboarding not completed yet.
    const isDone = get(onboarding);
    open = !isDone;
  });
</script>

<svelte:head>
  <title>AURA Quick Start</title>
  <meta name="robots" content="noindex" />
</svelte:head>

<section class="max-w-3xl mx-auto p-6 space-y-4">
  <h1 class="text-2xl font-semibold">Quick Start</h1>
  <p class="text-gray-700">Follow these steps to integrate AURA in minutes.</p>

  <div class="rounded border p-4 bg-gray-50 text-sm">
    <p class="mb-2">This guide is shown only for first-time users. You can always revisit it here.</p>
    <ul class="list-disc list-inside space-y-1">
      <li>Create an agent</li>
      <li>Add an allow rule</li>
      <li>Generate an API key</li>
      <li>Call verify</li>
    </ul>
  </div>

  <div class="text-sm text-gray-700">
    <p>Prefilled values:</p>
    <ul class="list-disc list-inside">
      <li>Agent ID: <code>{agentId || '—'}</code></li>
      <li>API Key (masked): <code>{secretKeyMasked || 'aura_sk_****'}</code></li>
    </ul>
  </div>
</section>

<SDKQuickStartDrawer {open} {agentId} {secretKeyMasked} showFullKey={false} />
