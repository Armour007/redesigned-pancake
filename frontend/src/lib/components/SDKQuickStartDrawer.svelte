<script lang="ts">
  import { onMount } from 'svelte';
  import { onboarding } from '$lib/stores/onboarding';
  import { get } from 'svelte/store';

  export let open = false;
  export let agentId = '';
  export let secretKeyMasked = '';
  export let showFullKey = false; // only right after creation
  export let apiBase = (import.meta.env.PUBLIC_API_BASE || 'http://localhost:8081');
  export let version = '2025-10-01';

  const langs = ['cURL', 'Node', 'Python', 'Go']; // extend later via codegen
  let active = 'cURL';

  const snippet = (lang: string) => {
    switch (lang) {
      case 'Node':
        return `await fetch('${apiBase}/v1/verify', {\n  method: 'POST',\n  headers: {\n    'Content-Type': 'application/json',\n    'X-API-Key': '${showFullKey ? secretKeyMasked : '<YOUR_SECRET_KEY>'}',\n    'AURA-Version': '${version}'\n  },\n  body: JSON.stringify({\n    agent_id: '${agentId || '<YOUR_AGENT_ID>'}',\n    request_context: { action: 'deploy', env: 'prod' }\n  })\n})`;
      case 'Python':
        return `import requests\nrequests.post('${apiBase}/v1/verify', headers={\n  'X-API-Key': '${showFullKey ? secretKeyMasked : '<YOUR_SECRET_KEY>'}',\n  'AURA-Version': '${version}'\n}, json={\n  'agent_id': '${agentId || '<YOUR_AGENT_ID>'}',\n  'request_context': {'action':'deploy','env':'prod'}\n})`;
      case 'Go':
        return `req, _ := http.NewRequest('POST', '${apiBase}/v1/verify', bytes.NewBuffer([]byte(\n  '{"agent_id":"${agentId || '<YOUR_AGENT_ID>'}","request_context":{"action":"deploy","env":"prod"}}'\n)))\nreq.Header.Set("Content-Type","application/json")\nreq.Header.Set("X-API-Key", "${showFullKey ? secretKeyMasked : '<YOUR_SECRET_KEY>'}")\nreq.Header.Set("AURA-Version", "${version}")`;
      default: // cURL
        return `curl -X POST '${apiBase}/v1/verify' \n  -H 'Content-Type: application/json' \n  -H 'X-API-Key: ${showFullKey ? secretKeyMasked : '<YOUR_SECRET_KEY>'}' \n  -H 'AURA-Version: ${version}' \n  -d '{"agent_id":"${agentId || '<YOUR_AGENT_ID>'}","request_context":{"action":"deploy","env":"prod"}}'`;
    }
  }

  onMount(() => {
    onboarding.init();
  });
</script>

{#if open}
  <div class="fixed inset-0 bg-black/40 z-40" />
  <aside class="fixed right-0 top-0 h-full w-full sm:w-[560px] bg-white z-50 shadow-xl overflow-y-auto">
    <div class="p-5 border-b flex items-center justify-between">
      <h2 class="text-lg font-semibold">SDK Quick Start</h2>
      <button class="text-gray-500 hover:text-gray-800" on:click={() => open = false}>✕</button>
    </div>

    <div class="p-5 space-y-4">
      <ol class="list-decimal list-inside text-sm text-gray-700 space-y-1">
        <li>Create an agent</li>
        <li>Add an allow rule</li>
        <li>Generate an API key</li>
        <li>Call verify</li>
      </ol>

      <div class="rounded border p-3 bg-gray-50 text-sm">
        <div class="mb-2">Agent ID: <code>{agentId || '—'}</code></div>
        <div>API Key: <code>{showFullKey ? secretKeyMasked : secretKeyMasked || 'aura_sk_****'}</code></div>
      </div>

      <div class="flex gap-2 flex-wrap text-sm">
        {#each langs as l}
          <button class="px-3 py-1 rounded border {active === l ? 'bg-black text-white' : 'bg-white'}" on:click={() => active = l}>{l}</button>
        {/each}
      </div>

      <pre class="p-3 bg-black text-green-200 rounded overflow-auto text-xs"><code>{snippet(active)}</code></pre>

      <div class="pt-2 flex items-center justify-between">
        <button class="px-3 py-2 rounded bg-black text-white" on:click={() => { onboarding.complete(); open = false; }}>Done</button>
        <button class="px-3 py-2 rounded border" on:click={() => onboarding.reset()}>Reset first-time state</button>
      </div>
    </div>
  </aside>
{/if}

<style>
  code { word-break: break-all; }
</style>
