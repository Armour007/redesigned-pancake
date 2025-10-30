<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { API_BASE } from '$lib/api';
  import OrgAdminTabs from '$lib/components/OrgAdminTabs.svelte';

  type TrustKey = {
    id: string;
    kid: string;
    alg: string;
    active: boolean;
    created_at: string;
    deactivate_after?: string;
    provider?: string;
    key_ref?: string;
    key_version?: string;
  };
  type JWK = { kty: string; crv?: string; alg?: string; use?: string; kid?: string; x?: string };

  let keys: TrustKey[] = [];
  let jwks: JWK[] = [];
  let loading = false;
  let error: string | null = null;
  let exclusive = true;
  let newKid = '';
  let grace = '1h'; // default grace window displayed to user
  let graceSeconds: number | '' = '';

  // KMS provider form state
  const providers = [
    { value: 'local', label: 'Local (Ed25519)' },
    { value: 'vault', label: 'HashiCorp Vault (transit)' },
    { value: 'aws', label: 'AWS KMS' },
    { value: 'gcp', label: 'GCP KMS' },
    { value: 'azure', label: 'Azure Key Vault' }
  ];
  let provider: 'local' | 'vault' | 'aws' | 'gcp' | 'azure' = 'local';
  let keyRef = '';
  let keyVersion = '';
  let alg: 'EdDSA' | 'ES256' | '' = '';
  let jwkPubText = '';
  let jwkParseError: string | null = null;
  const jwkPlaceholder = '{"kty":"OKP","crv":"Ed25519","kid":"...","x":"..."}';
  $: parsedJwk = (() => {
    jwkParseError = null;
    const t = jwkPubText.trim();
    if (!t) return null;
    try {
      const j = JSON.parse(t);
      if (typeof j !== 'object' || Array.isArray(j)) throw new Error('JWK must be an object');
      return j as Record<string, any>;
    } catch (e: any) {
      jwkParseError = e?.message || 'Invalid JSON';
      return null;
    }
  })();
  $: jwkTextareaClass = 'px-3 py-2 rounded bg-black/30 border w-full min-w-[360px] min-h-[72px] ' + (jwkParseError ? 'border-red-500/60' : 'border-white/10');

  // derive current orgId and path from $page store for template use
  $: currentOrgId = $page.params.orgId;
  $: currentPath = $page.url.pathname;

  // simple grace validation: allow sequences like 30m, 1h30m, 2h15m10s, 1d
  function validateGrace(input: string): { ok: boolean; normalized: string; error?: string } {
    const s = (input || '').toLowerCase().replace(/\s+/g, '');
    if (!s) return { ok: true, normalized: '' };
    // only allow digits followed by unit, repeated; units: s,m,h,d
    const valid = /^(\d+[smhd])+$/i.test(s);
    if (!valid) {
      return { ok: false, normalized: '', error: 'Use durations like 30m, 1h, or 1h30m' };
    }
    // reject all-zero like 0s or 0h0m
    const numParts = s.match(/\d+/g) || [];
    const nonZero = numParts.some((n) => parseInt(n, 10) > 0);
    if (!nonZero) return { ok: false, normalized: '', error: 'Duration must be > 0' };
    return { ok: true, normalized: s };
  }

  let graceValid = true; let graceError: string = ''; let graceNormalized = '';
  $: { const v = validateGrace(grace); graceValid = v.ok; graceError = v.error || ''; graceNormalized = v.normalized; }

  // compute a preview timestamp for deactivation
  function durationToSeconds(s: string): number {
    if (!s) return 0;
    const re = /(\d+)([smhd])/gi;
    let total = 0; let m: RegExpExecArray | null;
    while ((m = re.exec(s)) !== null) {
      const n = parseInt(m[1], 10);
      const u = m[2].toLowerCase();
      if (u === 's') total += n;
      else if (u === 'm') total += n * 60;
      else if (u === 'h') total += n * 3600;
      else if (u === 'd') total += n * 86400;
    }
    return total;
  }
  $: effectiveSeconds = (typeof graceSeconds === 'number' && graceSeconds > 0)
    ? graceSeconds
    : durationToSeconds(graceNormalized);
  $: previewText = effectiveSeconds > 0 ? new Date(Date.now() + effectiveSeconds * 1000).toLocaleString() : '';

  function authHeaders() {
    const token = localStorage.getItem('aura_token') || '';
    return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } as Record<string, string>;
  }

  async function loadAll() {
    loading = true; error = null;
    try {
      const orgId = $page.params.orgId;
      const [kres, jres] = await Promise.all([
        fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys`, { headers: authHeaders() }),
        fetch(`${API_BASE.replace(/\/$/, '')}/.well-known/aura/${orgId}/jwks.json`)
      ]);
      if (!kres.ok) throw new Error(`keys failed: ${kres.status}`);
      if (!jres.ok) throw new Error(`jwks failed: ${jres.status}`);
      const kjson = await kres.json();
      const jjson = await jres.json();
      keys = (kjson.keys || []) as TrustKey[];
      jwks = (jjson.keys || []) as JWK[];
    } catch (e: any) {
      error = e?.message || 'load failed';
    } finally {
      loading = false;
    }
  }

  async function createKey() {
    const orgId = $page.params.orgId;
    const isLocal = provider === 'local';
    const payload: any = { kid: newKid || undefined, active: false };
    if (!isLocal) {
      if (!keyRef.trim()) { toast('key_ref is required for KMS providers', 'error'); return; }
      payload.provider = provider;
      payload.key_ref = keyRef.trim();
      if (keyVersion.trim()) payload.key_version = keyVersion.trim();
      if (alg) payload.alg = alg;
      if (parsedJwk) payload.jwk_pub = parsedJwk;
    }
    const body = JSON.stringify(payload);
    const res = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys`, { method: 'POST', headers: authHeaders(), body });
    if (!res.ok) return alert('create failed');
    newKid = '';
    if (!isLocal) { keyRef = ''; keyVersion = ''; jwkPubText = ''; alg = ''; }
    await loadAll();
  }

  async function rotateKey() {
    const orgId = $page.params.orgId;
    if (!graceValid) { toast('Invalid grace window', 'error'); return; }
    const isLocal = provider === 'local';
    if (isLocal) {
      const body = JSON.stringify({
        kid: newKid || undefined,
        ...(typeof graceSeconds === 'number' && graceSeconds > 0 ? { grace_seconds: graceSeconds } : { grace: graceNormalized || undefined })
      });
      const res = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys/rotate`, { method: 'POST', headers: authHeaders(), body });
      if (!res.ok) { toast('Rotate failed', 'error'); return; }
      newKid = '';
      await loadAll();
    } else {
      if (!keyRef.trim()) { toast('key_ref is required for KMS rotation', 'error'); return; }
      if (!parsedJwk) { toast(jwkParseError || 'Public JWK required for KMS rotation', 'error'); return; }
      // Create inactive KMS key, then activate exclusively with grace
      const createPayload: any = {
        kid: newKid || undefined,
        active: false,
        provider,
        key_ref: keyRef.trim(),
        ...(keyVersion.trim() ? { key_version: keyVersion.trim() } : {}),
        ...(alg ? { alg } : {}),
        jwk_pub: parsedJwk
      };
      const cRes = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys`, { method: 'POST', headers: authHeaders(), body: JSON.stringify(createPayload) });
      if (!cRes.ok) { toast('Create (KMS) failed', 'error'); return; }
      const cJson = await cRes.json();
      const q = new URLSearchParams({ exclusive: '1' });
      if (typeof graceSeconds === 'number' && graceSeconds > 0) q.set('grace_seconds', String(graceSeconds));
      else if (graceNormalized) q.set('grace', graceNormalized);
      const aRes = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys/${encodeURIComponent(cJson.id)}/activate?${q.toString()}`, { method: 'POST', headers: authHeaders() });
      if (!aRes.ok) { toast('Activate (KMS) failed', 'error'); return; }
      newKid = ''; keyRef = ''; keyVersion = ''; jwkPubText = ''; alg = '';
      await loadAll();
    }
  }

  async function activate(id: string) {
    const orgId = $page.params.orgId;
    if (!graceValid) { toast('Invalid grace window', 'error'); return; }
    const key = keys.find(k => k.id === id);
    if (exclusive && key && key.provider && key.provider !== 'local') {
      const proceed = confirm('This key uses a KMS provider. Ensure its public JWK is set before exclusive activation, or verification may fail. Continue?');
      if (!proceed) return;
    }
    const q = new URLSearchParams({ exclusive: exclusive ? '1' : '0' });
    if (typeof graceSeconds === 'number' && graceSeconds > 0) q.set('grace_seconds', String(graceSeconds));
    else if (graceNormalized) q.set('grace', graceNormalized);
    const res = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys/${id}/activate?${q.toString()}`, { method: 'POST', headers: authHeaders() });
    if (!res.ok) { toast('Activate failed', 'error'); return; }
    await loadAll();
  }
  async function deactivate(id: string) {
    const orgId = $page.params.orgId;
    const res = await fetch(`${API_BASE.replace(/\/$/, '')}/organizations/${orgId}/trust-keys/${id}/deactivate`, { method: 'POST', headers: authHeaders() });
    if (!res.ok) return alert('deactivate failed');
    await loadAll();
  }

  import { toast } from '$lib/toast';
  function copy(text: string) { navigator.clipboard.writeText(text).then(() => toast('Copied', 'success', 1500)).catch(() => toast('Copy failed', 'error')); }

  onMount(loadAll);
</script>

<div class="space-y-6">
  <!-- Org admin subnav (2-tab style) -->
  <OrgAdminTabs orgId={currentOrgId} />
  
  <div class="flex items-center justify-between">
    <h1 class="text-lg font-semibold">Trust Keys</h1>
    <div class="flex items-center gap-3">
      <label class="flex items-center gap-2 text-sm"><input type="checkbox" bind:checked={exclusive}> Exclusive activation</label>
      <button disabled={loading} on:click={loadAll} class="px-3 py-2 rounded bg-white/10 hover:bg-white/20 disabled:opacity-50">Refresh</button>
    </div>
  </div>

  <!-- Create / Rotate -->
  <div class="flex items-end gap-3 flex-wrap">
    <div>
      <label for="kid" class="block text-sm text-gray-400 mb-1">Optional kid</label>
      <input id="kid" bind:value={newKid} placeholder="custom kid (optional)" class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[260px]">
    </div>
    <div>
      <label for="provider" class="block text-sm text-gray-400 mb-1">Provider</label>
      <select id="provider" bind:value={provider} class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[220px]">
        {#each providers as p}
          <option value={p.value}>{p.label}</option>
        {/each}
      </select>
    </div>
    {#if provider !== 'local'}
      <div>
        <label for="keyRef" class="block text-sm text-gray-400 mb-1">Key Ref</label>
        <input id="keyRef" bind:value={keyRef} placeholder="e.g., arn:aws:kms:... or projects/..." class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[300px]">
      </div>
      <div>
        <label for="keyVersion" class="block text-sm text-gray-400 mb-1">Key Version (optional)</label>
        <input id="keyVersion" bind:value={keyVersion} placeholder="e.g., 1 or version-id" class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[160px]">
      </div>
      <div>
        <label for="alg" class="block text-sm text-gray-400 mb-1">Algorithm</label>
        <select id="alg" bind:value={alg} class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[140px]">
          <option value="">Auto</option>
          <option value="EdDSA">EdDSA</option>
          <option value="ES256">ES256</option>
        </select>
      </div>
      <div class="basis-full"></div>
      <div class="grow">
  <label for="jwkPub" class="block text-sm text-gray-400 mb-1">Public JWK (JSON)</label>
  <textarea id="jwkPub" bind:value={jwkPubText} placeholder={jwkPlaceholder} class={jwkTextareaClass}></textarea>
        {#if jwkParseError}
          <div class="text-xs text-red-400 mt-1">{jwkParseError}</div>
        {:else if parsedJwk?.kid}
          <div class="text-xs text-gray-400 mt-1">Detected kid: <span class="font-mono">{parsedJwk.kid}</span></div>
        {/if}
      </div>
    {/if}
    <div class="min-w-[200px]">
      <label for="grace" class="block text-sm text-gray-400 mb-1">Grace window</label>
      <input id="grace" bind:value={grace} placeholder="e.g. 30m, 2h" class={`px-3 py-2 rounded w-[160px] ${graceValid ? 'bg-black/30 border border-white/10' : 'bg-red-900/30 border border-red-500/60'}`} title="Duration like 15m, 1h; used for rotation/activation overlap" />
      {#if !graceValid}
        <div class="text-xs text-red-400 mt-1">{graceError}</div>
      {/if}
    </div>
    <div>
      <label for="graceSeconds" class="block text-sm text-gray-400 mb-1">Or seconds</label>
      <input id="graceSeconds" type="number" min="0" placeholder="e.g. 3600" bind:value={graceSeconds} class="px-3 py-2 rounded bg-black/30 border border-white/10 w-[140px]" />
      <div class="text-xs text-gray-400 mt-1">If set > 0, overrides duration.</div>
    </div>
    <button on:click={createKey} class="px-3 py-2 rounded bg-white/10 hover:bg-white/20">Create (inactive)</button>
    <button disabled={!graceValid} on:click={rotateKey} class="px-3 py-2 rounded bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 disabled:cursor-not-allowed">Rotate (create active)</button>
  </div>

  <!-- Preview of deactivation time based on grace -->
  <div class="text-xs text-gray-400">
    {#if previewText}
      Preview: Other active keys will be deactivated after {previewText}.
    {:else}
      Preview: Immediate deactivation (no grace).
    {/if}
  </div>

  {#if loading}
    <div class="text-gray-400">Loading…</div>
  {:else if error}
    <div class="text-red-400">{error}</div>
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <!-- Keys table -->
      <div>
        <h2 class="text-sm font-semibold text-gray-300 mb-2">Org Keys</h2>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="text-left text-gray-400">
                <th class="py-2 px-3">ID</th>
                <th class="py-2 px-3">KID</th>
                <th class="py-2 px-3">Active</th>
                <th class="py-2 px-3">Provider</th>
                <th class="py-2 px-3">Ref</th>
                <th class="py-2 px-3">Ver</th>
                <th class="py-2 px-3">Created</th>
                <th class="py-2 px-3">Deactivates</th>
                <th class="py-2 px-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each keys as k}
                <tr class="border-t border-white/10">
                  <td class="py-2 px-3 font-mono text-xs max-w-[240px] truncate" title={k.id}>{k.id}</td>
                  <td class="py-2 px-3"><span class="font-mono text-xs">{k.kid}</span> <button on:click={() => copy(k.kid)} class="ml-2 text-xs text-indigo-400 hover:underline">copy</button></td>
                  <td class="py-2 px-3">{k.active ? 'Yes' : 'No'}</td>
                  <td class="py-2 px-3">{k.provider || 'local'}</td>
                  <td class="py-2 px-3 max-w-[220px] truncate" title={k.key_ref || ''}>{k.key_ref || '—'}</td>
                  <td class="py-2 px-3">{k.key_version || '—'}</td>
                  <td class="py-2 px-3">{k.created_at}</td>
                  <td class="py-2 px-3">{k.deactivate_after || '—'}</td>
                  <td class="py-2 px-3 space-x-2">
                    {#if !k.active}
                      <button on:click={() => activate(k.id)} class="px-2 py-1 rounded bg-white/10 hover:bg-white/20">Activate</button>
                    {:else}
                      <button on:click={() => deactivate(k.id)} class="px-2 py-1 rounded bg-white/10 hover:bg-white/20">Deactivate</button>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <!-- JWKS viewer -->
      <div>
        <h2 class="text-sm font-semibold text-gray-300 mb-2">Published JWKS</h2>
        {#if jwks.length === 0}
          <div class="text-gray-400">No active keys published.</div>
        {:else}
          <div class="space-y-3">
            {#each jwks as j}
              <div class="p-3 rounded border border-white/10 bg-black/30">
                <div class="text-xs text-gray-400">kid</div>
                <div class="font-mono text-xs break-all">{j.kid}<button on:click={() => copy(j.kid || '')} class="ml-2 text-xs text-indigo-400 hover:underline">copy</button></div>
                <div class="text-xs mt-1">alg: {j.alg} • crv: {j.crv} • kty: {j.kty}</div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
