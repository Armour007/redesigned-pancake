<script lang="ts">
  export let data: {
    agentCount: number;
    verificationCountToday: number;
    apiKeyCount: number;
    error?: string;
  };
  import Alert from '$lib/components/Alert.svelte';
</script>

<h1 class="text-3xl font-bold mb-6">Overview</h1>

{#if data?.error}
  <Alert variant="error">{data.error}</Alert>
{/if}

{#if (data?.agentCount ?? 0) === 0}
  <div class="mb-6 p-4 rounded-xl border border-[#333333] bg-[#141414] flex items-center justify-between">
    <div>
      <div class="text-white font-semibold">Get started with AURA</div>
      <div class="text-sm text-gray-400">Create your first agent, add a rule, issue an API key, and copy an SDK snippet.</div>
    </div>
    <a href="/onboarding" class="px-4 py-2 rounded-lg bg-[#7C3AED] hover:bg-[#6d28d9] text-white text-sm">Start onboarding</a>
  </div>
{/if}

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
  <div class="bg-[#1A1A1A] p-6 rounded-xl border border-[#333333]">
    <h2 class="text-lg font-semibold mb-2">Total Agents</h2>
    <p class="text-4xl font-bold">{data.agentCount}</p>
  </div>

  <div class="bg-[#1A1A1A] p-6 rounded-xl border border-[#333333]">
    <h2 class="text-lg font-semibold mb-2">Verifications Today</h2>
    <p class="text-4xl font-bold">{data.verificationCountToday}</p>
  </div>

  <div class="bg-[#1A1A1A] p-6 rounded-xl border border-[#333333]">
    <h2 class="text-lg font-semibold mb-2">Active API Keys</h2>
    <p class="text-4xl font-bold">{data.apiKeyCount}</p>
  </div>
</div>