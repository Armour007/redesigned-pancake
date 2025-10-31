<script lang="ts">
  export let data: { logs: any[]; error?: string };
  import Alert from '$lib/components/Alert.svelte';

  function fmt(d?: string) {
    if (!d) return '-';
    try {
      return new Date(d).toLocaleString();
    } catch {
      return d;
    }
  }
</script>

<h1 class="text-3xl font-bold mb-6 a-text-gradient">Event Logs</h1>
{#if data?.error}
  <Alert variant="error">{data.error}</Alert>
{/if}

<div class="overflow-x-auto a-card a-ribbon">
  <table class="min-w-full text-left text-sm">
    <thead class="bg-[#1A1A1A] text-gray-300">
      <tr>
        <th class="px-4 py-3">Time</th>
        <th class="px-4 py-3">Event</th>
        <th class="px-4 py-3">Agent</th>
        <th class="px-4 py-3">Request ID</th>
        <th class="px-4 py-3">Details</th>
      </tr>
    </thead>
    <tbody>
      {#each data.logs as log}
        <tr class="border-b border-white/10 align-top">
          <td class="px-4 py-3 whitespace-nowrap">{fmt(log.created_at)}</td>
          <td class="px-4 py-3">{log.event_type || '-'}</td>
          <td class="px-4 py-3">{log.agent_id || '-'}</td>
          <td class="px-4 py-3">{log.request_id || '-'}</td>
          <td class="px-4 py-3 max-w-[520px]">
            <pre class="text-xs text-gray-300 whitespace-pre-wrap break-words">{JSON.stringify(log.request_details ?? log.details ?? log, null, 2)}</pre>
          </td>
        </tr>
      {/each}
      {#if !data.logs || data.logs.length === 0}
        <tr><td colspan="5" class="px-4 py-6 text-gray-400">No logs available.</td></tr>
      {/if}
    </tbody>
  </table>
</div>
 
