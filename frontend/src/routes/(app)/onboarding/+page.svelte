<script lang="ts">
  import { API_BASE, authHeaders } from '$lib/api';
  import { onMount } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';

  let step = 1;
  let orgId: string | null = null;
  let token: string | null = null;

  // Step 1: Agent
  let agentName = '';
  let agentDescription = '';
  let createdAgentId: string | null = null;

  // Step 2: Rule
  let actionName = 'deploy:prod';
  let createdRuleId: string | null = null;

  // Step 3: API key
  let apiKeyName = 'Default key';
  let createdSecretKey: string | null = null;

  let loading = false;
  let error: string | null = null;

  onMount(() => {
    token = localStorage.getItem('aura_token');
    orgId = localStorage.getItem('aura_org_id');
  });

  async function createAgent() {
    if (!token || !orgId) {
      error = 'Missing auth. Please log in again.';
      return;
    }
    if (!agentName.trim()) {
      error = 'Agent name is required.';
      return;
    }
    loading = true; error = null;
    try {
      const res = await fetch(`${API_BASE}/organizations/${orgId}/agents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ name: agentName, ...(agentDescription.trim() && { description: agentDescription }) })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || `Failed (${res.status})`);
      createdAgentId = data.id;
      step = 2;
    } catch (e: any) {
      error = e.message || 'Failed to create agent';
    } finally { loading = false; }
  }

  async function addRule() {
    if (!token || !orgId || !createdAgentId) return;
    loading = true; error = null;
    try {
      const rule = { effect: 'allow', action: actionName };
      const res = await fetch(`${API_BASE}/organizations/${orgId}/agents/${createdAgentId}/permissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ rule })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || `Failed (${res.status})`);
      createdRuleId = data.id;
      step = 3;
    } catch (e: any) {
      error = e.message || 'Failed to add rule';
    } finally { loading = false; }
  }

  async function createApiKey() {
    if (!token || !orgId) return;
    loading = true; error = null;
    try {
      const res = await fetch(`${API_BASE}/organizations/${orgId}/apikeys`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
        body: JSON.stringify({ name: apiKeyName })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || `Failed (${res.status})`);
      createdSecretKey = data.secret_key;
      step = 4;
    } catch (e: any) {
      error = e.message || 'Failed to create API key';
    } finally { loading = false; }
  }

  function sdkBase() {
    // Prefer configured base, fall back to same-origin
    return (API_BASE || window.location.origin);
  }
</script>

<div class="space-y-8">
  <h1 class="text-3xl font-bold text-white">Getting Started</h1>
  <p class="text-gray-400">Follow these steps to create an agent, add a rule, issue an API key, and integrate with your app.</p>

  <!-- Progress -->
  <div class="flex items-center gap-4">
    {#each [1,2,3,4] as s}
      <div class="flex items-center">
        <div class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
             class:bg-[#7C3AED]={s <= step}
             class:bg-[#2a2a2a]={s > step}
             class:text-white={s<=step}
             class:text-gray-400={s>step}>{s}</div>
        {#if s<4}<div class="w-12 h-[2px] bg-[#333333] mx-2"></div>{/if}
      </div>
    {/each}
  </div>

  {#if error}
    <Alert variant="error">{error}</Alert>
  {/if}

  {#if step === 1}
    <div class="space-y-4 bg-[#131313] border border-[#333] rounded-xl p-6">
      <h2 class="text-xl font-semibold text-white">Step 1 — Create an agent</h2>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="block text-sm text-gray-300" for="agentNameInput">Name</label>
          <input id="agentNameInput" class="mt-1 w-full p-3 bg-[#0f0f0f] border border-[#333] rounded-lg text-white" bind:value={agentName} placeholder="Production Deployer" />
        </div>
        <div>
          <label class="block text-sm text-gray-300" for="agentDescriptionInput">Description (optional)</label>
          <input id="agentDescriptionInput" class="mt-1 w-full p-3 bg-[#0f0f0f] border border-[#333] rounded-lg text-white" bind:value={agentDescription} placeholder="Bot that deploys services" />
        </div>
      </div>
      <button class="px-4 py-2 bg-[#7C3AED] hover:bg-[#6d28d9] rounded-lg text-white" on:click|preventDefault={createAgent} disabled={loading}> {loading ? 'Creating…' : 'Create Agent'} </button>
    </div>
  {/if}

  {#if step === 2}
    <div class="space-y-4 bg-[#131313] border border-[#333] rounded-xl p-6">
      <h2 class="text-xl font-semibold text-white">Step 2 — Add a rule</h2>
      <p class="text-gray-400 text-sm">We’ll create a simple allow rule for a named action. You can add more conditions later.</p>
      <div class="max-w-md">
        <label class="block text-sm text-gray-300" for="actionNameInput">Action</label>
        <input id="actionNameInput" class="mt-1 w-full p-3 bg-[#0f0f0f] border border-[#333] rounded-lg text-white" bind:value={actionName} placeholder="deploy:prod" />
      </div>
      <button class="px-4 py-2 bg-[#7C3AED] hover:bg-[#6d28d9] rounded-lg text-white" on:click|preventDefault={addRule} disabled={loading}> {loading ? 'Saving…' : 'Add Rule'} </button>
    </div>
  {/if}

  {#if step === 3}
    <div class="space-y-4 bg-[#131313] border border-[#333] rounded-xl p-6">
      <h2 class="text-xl font-semibold text-white">Step 3 — Create an API key</h2>
      <p class="text-gray-400 text-sm">This key authenticates calls from your app to AURA.</p>
      <div class="max-w-md">
        <label class="block text-sm text-gray-300" for="apiKeyNameInput">Key name</label>
        <input id="apiKeyNameInput" class="mt-1 w-full p-3 bg-[#0f0f0f] border border-[#333] rounded-lg text-white" bind:value={apiKeyName} placeholder="Default key" />
      </div>
      <button class="px-4 py-2 bg-[#7C3AED] hover:bg-[#6d28d9] rounded-lg text-white" on:click|preventDefault={createApiKey} disabled={loading}> {loading ? 'Creating…' : 'Create API Key'} </button>
      {#if createdSecretKey}
        <Alert variant="success">
          <div>Save this secret now; it won’t be shown again.</div>
          <div class="mt-1 font-mono break-all">{createdSecretKey}</div>
        </Alert>
      {/if}
    </div>
  {/if}

  {#if step === 4}
    <div class="space-y-4 bg-[#131313] border border-[#333] rounded-xl p-6">
      <h2 class="text-xl font-semibold text-white">Step 4 — Plug in the SDK</h2>
      <p class="text-gray-400 text-sm">Copy a snippet for your stack. Replace values as needed.</p>

      <div class="grid md:grid-cols-3 gap-4">
        <div class="bg-[#0f0f0f] border border-[#333] rounded-lg p-4">
          <div class="font-semibold text-white mb-2">Node</div>
          <pre class="text-xs text-gray-300 whitespace-pre-wrap"><code>{`import { AuraClient } from '@auraai/sdk-node';
const client = new AuraClient({ apiKey: '${createdSecretKey ?? 'aura_sk_...'}', baseURL: '${sdkBase()}', version: '2025-10-01' });
const res = await client.verify('${createdAgentId ?? '<agent-uuid>'}', { action: '${actionName}' });
console.log(res);`}</code></pre>
        </div>
        <div class="bg-[#0f0f0f] border border-[#333] rounded-lg p-4">
          <div class="font-semibold text-white mb-2">Python</div>
          <pre class="text-xs text-gray-300 whitespace-pre-wrap"><code>{`from aura_sdk import AuraClient
client = AuraClient(api_key='${createdSecretKey ?? 'aura_sk_...'}', base_url='${sdkBase()}', version='2025-10-01')
res = client.verify('${createdAgentId ?? '<agent-uuid>'}', { 'action': '${actionName}' })
print(res)`}</code></pre>
        </div>
        <div class="bg-[#0f0f0f] border border-[#333] rounded-lg p-4">
          <div class="font-semibold text-white mb-2">Go</div>
          <pre class="text-xs text-gray-300 whitespace-pre-wrap"><code>{`c := aura.NewClient("${createdSecretKey ?? 'aura_sk_...'}", "${sdkBase()}", "2025-10-01")
res, err := c.Verify("${createdAgentId ?? '<agent-uuid>'}", map[string]any{"action":"${actionName}"})
_ = res; _ = err`}</code></pre>
        </div>
      </div>

      <div class="text-sm text-gray-400">
        Need webhooks? See the Client Guide for signed webhook verification examples.
      </div>
    </div>
  {/if}
</div>

<style>
  code { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
  pre { overflow-x: auto; }
</style>
