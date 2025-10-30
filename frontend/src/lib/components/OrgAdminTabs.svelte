<script lang="ts">
  import { page } from '$app/stores';
  export let orgId: string | undefined;
  $: path = $page.url.pathname;
  // Resolve org id from prop or page params as a fallback to satisfy strict types
  $: _org = orgId ?? $page.params.orgId ?? '';
</script>

<div class="inline-flex rounded-lg border border-white/10 bg-black/20 p-1 text-sm">
  <a class={`px-3 py-1.5 rounded-md ${path.endsWith('/webhooks') ? 'bg-white/10 text-white' : 'text-gray-300 hover:text-white hover:bg-white/5'}`}
    href={`/organizations/${_org}/webhooks`}>Webhooks</a>
  <a class={`px-3 py-1.5 rounded-md ${path.endsWith('/trust-keys') ? 'bg-white/10 text-white' : 'text-gray-300 hover:text-white hover:bg-white/5'}`}
    href={`/organizations/${_org}/trust-keys`}>Trust keys</a>
  <a class={`px-3 py-1.5 rounded-md ${path.endsWith('/regulator/audit-export') ? 'bg-white/10 text-white' : 'text-gray-300 hover:text-white hover:bg-white/5'}`}
    href={`/organizations/${_org}/regulator/audit-export`}>Audit export</a>
  <slot />
</div>
