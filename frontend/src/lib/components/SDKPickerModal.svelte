<script lang="ts">
  import Modal from '$lib/components/Modal.svelte';
  import { API_BASE } from '$lib/api';
  import { onMount } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';

  export let showModal: boolean = false;
  export let agentId: string = '';
  export let organizationId: string = '';
  export let action: string = '';

  // Dynamic tabs and languages loaded from backend
  let curatedTabs: string[] = ['node', 'python', 'go'];
  let codegenLangs: string[] = ['java','csharp','ruby','php','rust','swift','kotlin','dart','cpp'];
  let active = 'node'; // one of curatedTabs or 'other'
  let langsLoading = true;
  let langsFailed = false;
  let noLangs = false;
  let errorMsg = '';
  let downloading = false;
  let copied = '';

  // Other languages async codegen (populated via /sdk/supported-langs)
  let otherSelected = 'java';
  // Load allowed SDK languages from backend and hide disallowed in the UI
  onMount(async () => {
    langsLoading = true;
    langsFailed = false;
    try {
      const token = localStorage.getItem('aura_token');
      if (!token) { langsLoading = false; return; } // leave defaults if not logged in
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/sdk/supported-langs`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) { langsFailed = true; langsLoading = false; return; } // keep defaults on error
      const j = await res.json();
      if (Array.isArray(j.curated) && j.curated.length > 0) {
        curatedTabs = j.curated.map((s: string) => String(s).toLowerCase());
      }
      if (Array.isArray(j.codegen)) {
        codegenLangs = j.codegen.map((s: string) => String(s).toLowerCase());
      }
      // Ensure active tab is valid; otherwise pick first available or fallback to 'other' if only codegen exists
      const tabs = [...curatedTabs, ...(codegenLangs.length ? ['other'] : [])];
      if (tabs.length === 0) {
        noLangs = true;
        active = '' as any;
      } else if (!tabs.includes(active)) {
        active = tabs[0] || (codegenLangs.length ? 'other' : 'node');
      }
      // Ensure selected other language is valid
      if (!codegenLangs.includes(otherSelected) && codegenLangs.length) {
        otherSelected = codegenLangs[0];
      }
    } catch {
      langsFailed = true; // keep defaults
    } finally {
      langsLoading = false;
    }
  });
  let email: string = '';
  let jobId: string = '';
  let jobStatus: string = '';
  let jobTimer: any = null;

  function codeSnippet(lang: string): string {
    const act = action || 'your:action';
    switch (lang) {
      case 'node':
        return `// npm i @aura/node (example)\nimport Aura from '@aura/node';\nconst client = new Aura({ baseUrl: '${API_BASE}', apiKey: process.env.AURA_API_KEY });\n\nconst decision = await client.verify({\n  action: '${act}',\n  context: { /* optional constraints */ }\n});\nconsole.log(decision);`;
      case 'python':
        return `# pip install aura-sdk (example)\nfrom aura_sdk import Aura\nclient = Aura(base_url='${API_BASE}', api_key=os.environ['AURA_API_KEY'])\n\ndecision = client.verify({\n  'action': '${act}',\n  'context': { }\n})\nprint(decision)`;
      case 'go':
        return `// go get github.com/Armour007/aura/sdks/go/aura\npackage main\n\nimport (\n  "fmt"\n  aura "github.com/Armour007/aura/sdks/go/aura"\n)\n\nfunc main(){\n  c := aura.NewClient("${API_BASE}", os.Getenv("AURA_API_KEY"))\n  res, err := c.Verify(aura.VerifyRequest{Action: "${act}"})\n  fmt.Println(res, err)\n}`;
      default:
        return '';
    }
  }

  function installCmd(lang: string): string {
    switch (lang) {
      case 'node': return 'npm i @aura/node  # example package name';
      case 'python': return 'pip install aura-sdk  # example package name';
      case 'go': return 'go get github.com/Armour007/aura/sdks/go/aura';
      default: return '';
    }
  }

  async function copy(text: string) {
    try { await navigator.clipboard.writeText(text); copied = 'copied'; setTimeout(()=>copied='', 1200); } catch {}
  }

  async function downloadSDK(lang: string) {
    errorMsg = '';
    downloading = true;
    try {
      const token = localStorage.getItem('aura_token');
      if (!token) throw new Error('Not authenticated. Please log in.');
      const params = new URLSearchParams({ lang, agent_id: agentId || '', action: action || '' });
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/sdk/download?${params.toString()}`, {
        method: 'GET',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (!res.ok) {
        const txt = await res.text();
        throw new Error(`Failed to download SDK (${res.status}): ${txt}`);
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `aura-sdk-${lang}.zip`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (e: any) {
      errorMsg = e.message || 'Download failed';
    } finally {
      downloading = false;
    }
  }

  async function startGenerate() {
    errorMsg = '';
    jobId = '';
    jobStatus = '';
    const token = localStorage.getItem('aura_token');
    if (!token) { errorMsg = 'Not authenticated. Please log in.'; return; }
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/sdk/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ lang: otherSelected, agent_id: agentId, organization_id: organizationId, email })
      });
      const j = await res.json();
      if (!res.ok) throw new Error(j.error || `HTTP ${res.status}`);
      jobId = j.job_id; jobStatus = j.status || 'queued';
      if (jobTimer) { clearInterval(jobTimer); jobTimer = null; }
      jobTimer = setInterval(checkJob, 1000);
    } catch (e: any) {
      errorMsg = e.message || 'Failed to start generation';
    }
  }

  async function checkJob() {
    if (!jobId) return;
    const token = localStorage.getItem('aura_token');
    if (!token) return;
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/sdk/generate/${jobId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      const j = await res.json();
      if (res.ok) {
        jobStatus = j.status || '';
        if (jobStatus === 'ready' || jobStatus === 'error') {
          if (jobTimer) { clearInterval(jobTimer); jobTimer = null; }
        }
      } else {
        if (jobTimer) { clearInterval(jobTimer); jobTimer = null; }
        errorMsg = j.error || `Failed to fetch status (${res.status})`;
      }
    } catch {}
  }

  async function downloadGenerated() {
    if (!jobId) return;
    const token = localStorage.getItem('aura_token');
    if (!token) { errorMsg = 'Not authenticated. Please log in.'; return; }
    const url = `${API_BASE.replace(/\/$/, '')}/sdk/download-generated/${jobId}`;
    try {
      const res = await fetch(url, { headers: { 'Authorization': `Bearer ${token}` } });
      if (!res.ok) { const t = await res.text(); throw new Error(t || `HTTP ${res.status}`); }
      const blob = await res.blob();
      const blobUrl = URL.createObjectURL(blob);
      const a = document.createElement('a'); a.href = blobUrl; a.download = `aura-sdk-${otherSelected}-generated.zip`; document.body.appendChild(a); a.click(); a.remove(); URL.revokeObjectURL(blobUrl);
    } catch (e: any) { errorMsg = e.message || 'Download failed'; }
  }
</script>

<Modal title="Get Your SDK" bind:showModal>
  <div class="space-y-4">
    <p class="text-sm text-gray-300">Pick an SDK and plug it in where you enforce this rule.</p>
    {#if langsLoading}
      <p class="text-xs text-gray-400">Loading available SDKs…</p>
    {:else if langsFailed}
      <p class="text-xs text-amber-400">Couldn’t load SDK list; showing defaults.</p>
    {/if}

    {#if noLangs}
      <div class="py-2">
        <p class="text-xs text-amber-400">No SDKs are available right now. Ask your admin to enable SDK downloads.</p>
      </div>
    {:else}
      <div class="flex gap-2">
        {#each [...curatedTabs, ...(codegenLangs.length ? ['other'] : [])] as lang}
          <button type="button" class="px-3 py-1 rounded text-sm border border-gray-700 text-gray-300 hover:bg-white/10 {active===lang?'bg-[#7C3AED] text-white border-[#7C3AED]':''}" on:click={() => (active = lang)}>{lang.toUpperCase()}</button>
        {/each}
      </div>
    {/if}

  {#if !noLangs && active !== 'other'}
      <div class="space-y-2">
        <p class="text-xs text-gray-400">Install</p>
        <div class="flex items-center gap-2">
          <pre class="p-2 bg-[#111111] border border-[#333333] rounded font-mono text-xs overflow-auto">{installCmd(active)}</pre>
          <button class="text-xs px-2 py-1 rounded bg-white/10 hover:bg-white/20" on:click={() => copy(installCmd(active))}>Copy</button>
        </div>
      </div>
      <div class="space-y-2">
        <p class="text-xs text-gray-400">Snippet</p>
        <div class="flex items-center gap-2">
          <pre class="p-3 bg-[#111111] border border-[#333333] rounded font-mono text-xs overflow-auto"><code>{codeSnippet(active)}</code></pre>
          <button class="text-xs px-2 py-1 rounded bg-white/10 hover:bg-white/20" on:click={() => copy(codeSnippet(active))}>Copy</button>
        </div>
      </div>
  {:else if !noLangs}
      <div class="space-y-3">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <label class="block text-xs text-gray-400 mb-1" for="sdk-language">Language</label>
            <select id="sdk-language" bind:value={otherSelected} class="w-full p-2 bg-[#111111] border border-[#333333] rounded text-sm text-gray-200">
              {#each codegenLangs as l}<option value={l}>{l.toUpperCase()}</option>{/each}
            </select>
          </div>
          <div class="md:col-span-2">
            <label class="block text-xs text-gray-400 mb-1" for="sdk-email">Email (optional)</label>
            <input id="sdk-email" bind:value={email} class="w-full p-2 bg-[#111111] border border-[#333333] rounded text-sm text-gray-200" placeholder="you@example.com" />
          </div>
        </div>
        <div class="flex items-center gap-3">
          <button class="px-3 py-2 text-sm text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded" on:click={startGenerate} disabled={!!jobId && !!jobStatus && jobStatus !== 'error'}>{jobId ? 'Regenerating…' : 'Generate SDK'}</button>
          {#if jobId}
            <span class="text-xs text-gray-400">Job {jobId} — {jobStatus || 'queued'}</span>
            {#if jobStatus === 'ready'}
              <button class="px-2 py-1 text-xs rounded bg-white/10 hover:bg-white/20" on:click={downloadGenerated}>Download</button>
            {/if}
          {/if}
        </div>
      </div>
    {/if}

    {#if errorMsg}
      <Alert variant="error">{errorMsg}</Alert>
    {/if}

    {#if !noLangs && active !== 'other'}
      <div class="flex justify-end gap-3 pt-2">
        <button class="px-3 py-2 text-sm text-gray-300 hover:bg-white/10 rounded" on:click={() => (showModal = false)}>Close</button>
        <button class="px-3 py-2 text-sm text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded disabled:opacity-50" on:click={() => downloadSDK(active)} disabled={downloading}>{downloading ? 'Preparing…' : `Download ${active.toUpperCase()} SDK`}</button>
      </div>
    {:else if !noLangs}
      <div class="flex justify-end pt-2">
        <button class="px-3 py-2 text-sm text-gray-300 hover:bg-white/10 rounded" on:click={() => (showModal = false)}>Close</button>
      </div>
    {:else}
      <div class="flex justify-end pt-2">
        <button class="px-3 py-2 text-sm text-gray-300 hover:bg-white/10 rounded" on:click={() => (showModal = false)}>Close</button>
      </div>
    {/if}
  </div>
</Modal>
