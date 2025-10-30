<script lang="ts">
  import { onMount } from 'svelte';
  import { API_BASE } from '$lib/api';
  import { toast } from '$lib/toast';

  // Simple auth header helper (reuse local storage token like other admin pages)
  function authHeaders() {
    const token = localStorage.getItem('aura_token') || '';
    return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } as Record<string, string>;
  }

  // Authoring state
  let engine: 'aurajson' | 'rego' = 'aurajson';
  let nlPrompt = '';
  let notes: string[] = [];

  // Policy body (as pretty JSON in textarea)
  let bodyText = '{\n  "rules": [\n    { "id": "allow_all", "effect": "allow" }\n  ],\n  "precedence": { "deny_overrides": true }\n}';
  let bodyParseError: string | null = null;
  $: parsedBody = (() => {
    bodyParseError = null;
    try {
      const j = JSON.parse(bodyText);
      return j as Record<string, any>;
    } catch (e: any) {
      bodyParseError = e?.message || 'Invalid JSON';
      return null;
    }
  })();

  // Tests state
  type TestCase = { input: string; expect: 'allow' | 'deny' | 'needs_approval' };
  let tests: TestCase[] = [
    { input: '{\n  "action": "example",\n  "agent": { "id": "00000000-0000-0000-0000-000000000000" }\n}', expect: 'allow' }
  ];
  let running = false;
  let testResults: Array<{ index: number; status: string; reason?: string; hints?: string[]; pass: boolean }> = [];

  // Preview state
  let previewLimit = 100;
  let previewSummary: { allow: number; deny: number; needs_approval: number; changed: number } | null = null;
  let previewSamples: Array<{ original: string; now: string; reason?: string }> = [];

  async function compileFromNL() {
    if (!nlPrompt.trim()) { toast('Enter a natural-language prompt', 'warning'); return; }
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/policy/author/nl-compile`, {
        method: 'POST', headers: authHeaders(),
        body: JSON.stringify({ engine, nl: nlPrompt })
      });
      if (!res.ok) throw new Error(await res.text());
      const json = await res.json();
      notes = json.notes || [];
      // pretty print returned body
      bodyText = JSON.stringify(json.body ? json.body : {}, null, 2);
      toast('Generated from NL', 'success', 2000);
    } catch (e: any) {
      toast(`Compile failed: ${e?.message || e}`, 'error');
    }
  }

  function addTest() {
    tests = [...tests, { input: '{\n  \n}', expect: 'deny' }];
  }
  function removeTest(i: number) {
    tests = tests.filter((_, idx) => idx !== i);
  }

  async function runTests() {
    if (!parsedBody) { toast('Fix policy JSON first', 'error'); return; }
    running = true; testResults = [];
    try {
      const formattedTests = tests.map((t) => ({ input: safeParseJSON(t.input), expect: t.expect }));
      if (formattedTests.some((t) => t.input === null)) { toast('One or more test inputs are invalid JSON', 'error'); return; }
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/policy/tests/run`, {
        method: 'POST', headers: authHeaders(),
        body: JSON.stringify({ engine, body: parsedBody, tests: formattedTests })
      });
      if (!res.ok) throw new Error(await res.text());
      const json = await res.json();
      testResults = json.results || [];
      const passCnt = testResults.filter((r: any) => r.pass).length;
      toast(`Tests: ${passCnt}/${testResults.length} passed`, passCnt === testResults.length ? 'success' : 'warning');
    } catch (e: any) {
      toast(`Tests failed: ${e?.message || e}`, 'error');
    } finally {
      running = false;
    }
  }

  function safeParseJSON(s: string): any | null {
    try { return JSON.parse(s); } catch { return null; }
  }

  async function previewAgainstTraces() {
    if (!parsedBody) { toast('Fix policy JSON first', 'error'); return; }
    try {
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/v2/policy/preview`, {
        method: 'POST', headers: authHeaders(),
        body: JSON.stringify({ engine, body: parsedBody, limit: previewLimit })
      });
      if (!res.ok) throw new Error(await res.text());
      const json = await res.json();
      previewSummary = json.summary || null;
      previewSamples = json.samples || [];
      toast('Preview complete', 'success');
    } catch (e: any) {
      toast(`Preview failed: ${e?.message || e}`, 'error');
    }
  }
  onMount(() => { /* no-op */ });
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-lg font-semibold">Cognitive Firewall — Author, Test, Preview</h1>
    <div class="flex items-center gap-3">
      <label class="text-sm text-gray-400" for="engineSel">Engine</label>
      <select id="engineSel" bind:value={engine} class="px-3 py-2 rounded bg-black/30 border border-white/10">
        <option value="aurajson">AuraJSON</option>
        <option value="rego">Rego (OPA)</option>
      </select>
    </div>
  </div>

  <!-- NL Authoring -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <div>
      <h2 class="text-sm font-semibold text-gray-300 mb-2">Describe your policy</h2>
      <textarea bind:value={nlPrompt} placeholder="e.g., Allow only read actions; write requires approval; deny deletions" class="w-full min-h-[140px] px-3 py-2 rounded bg-black/30 border border-white/10"></textarea>
      <div class="mt-2 flex gap-2">
        <button on:click={compileFromNL} class="px-3 py-2 rounded bg-white/10 hover:bg-white/20">Generate policy</button>
      </div>
      {#if notes.length}
        <ul class="mt-2 text-xs text-gray-400 list-disc pl-4">
          {#each notes as n}<li>{n}</li>{/each}
        </ul>
      {/if}
    </div>
    <div>
      <h2 class="text-sm font-semibold text-gray-300 mb-2">Policy JSON</h2>
      <textarea bind:value={bodyText} class={`w-full min-h-[240px] px-3 py-2 rounded ${bodyParseError ? 'bg-red-900/30 border border-red-500/60' : 'bg-black/30 border border-white/10'}`}></textarea>
      {#if bodyParseError}
        <div class="text-xs text-red-400 mt-1">{bodyParseError}</div>
      {/if}
    </div>
  </div>

  <!-- Test Runner -->
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h2 class="text-sm font-semibold text-gray-300">Run tests</h2>
      <button on:click={addTest} class="px-2 py-1 rounded bg-white/10 hover:bg-white/20 text-sm">Add test</button>
    </div>
    <div class="space-y-4">
      {#each tests as t, i}
        <div class="p-3 rounded border border-white/10 bg-black/20">
          <div class="flex items-center justify-between mb-2">
            <div class="text-xs text-gray-400">Test #{i + 1}</div>
            <div class="flex items-center gap-2">
              <label class="text-xs text-gray-400" for={`expect-${i}`}>Expect</label>
              <select id={`expect-${i}`} bind:value={t.expect} class="px-2 py-1 rounded bg-black/30 border border-white/10 text-sm">
                <option value="allow">allow</option>
                <option value="deny">deny</option>
                <option value="needs_approval">needs_approval</option>
              </select>
              <button on:click={() => removeTest(i)} class="text-xs text-red-300 hover:underline">remove</button>
            </div>
          </div>
          <textarea bind:value={t.input} class="w-full min-h-[120px] px-3 py-2 rounded bg-black/30 border border-white/10 font-mono text-[12px]"></textarea>
          {#if testResults.length}
            {#each testResults.filter(r => r.index === i) as r}
              <div class="mt-2 text-xs">
                <span class={`px-2 py-0.5 rounded ${r.pass ? 'bg-emerald-600/50' : 'bg-yellow-600/50'}`}>{r.pass ? 'PASS' : 'MISMATCH'}</span>
                <span class="ml-2 text-gray-300">status: {r.status}</span>
                {#if r.reason}
                  <span class="ml-2 text-gray-400">reason: {r.reason}</span>
                {/if}
                {#if r.hints?.length}
                  <div class="text-gray-400 mt-1">hints: {r.hints.join(', ')}</div>
                {/if}
              </div>
            {/each}
          {/if}
        </div>
      {/each}
    </div>
    <button disabled={running} on:click={runTests} class="px-3 py-2 rounded bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60">{running ? 'Running…' : 'Run tests'}</button>
  </div>

  <!-- Preview -->
  <div class="space-y-3">
    <div class="flex items-center gap-3">
      <h2 class="text-sm font-semibold text-gray-300">Preview against recent traces</h2>
      <label class="text-sm text-gray-400" for="previewLimit">Limit</label>
      <input id="previewLimit" type="number" min="1" max="1000" bind:value={previewLimit} class="px-3 py-1 rounded bg-black/30 border border-white/10 w-[120px]" />
      <button on:click={previewAgainstTraces} class="px-3 py-2 rounded bg-white/10 hover:bg-white/20">Preview</button>
    </div>
    {#if previewSummary}
      <div class="text-sm text-gray-300">Summary: allow {previewSummary.allow} • deny {previewSummary.deny} • needs_approval {previewSummary.needs_approval} • changed {previewSummary.changed}</div>
      {#if previewSamples.length}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          {#each previewSamples as s}
            <div class="p-3 rounded border border-white/10 bg-black/20 text-sm">
              <div>original: <span class="font-mono">{s.original}</span> → now: <span class="font-mono">{s.now}</span></div>
              {#if s.reason}
                <div class="text-xs text-gray-400 mt-1">reason: {s.reason}</div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>

<style>
  textarea { outline: none; }
</style>
