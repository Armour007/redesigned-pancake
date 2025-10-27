<script lang="ts">
	// Imports
	import type { PageData } from './$types';
	import { page } from '$app/stores'; // Used for potential future features like active link styling
	import RuleBuilderModal from '$lib/components/RuleBuilderModal.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { agentContext } from '$lib/stores/agentContext'; // Import the shared store
	import { API_BASE, authHeaders } from '$lib/api';
	import type { AgentDetailPageLoadData } from './+page'; // Import the specific type from +page.ts
  import Alert from '$lib/components/Alert.svelte';

	// Props from load function
	export let data: AgentDetailPageLoadData;

	// Component State
	let activeTab = 'rules'; // Default active tab
	let showRuleBuilderModal = false;

	// --- Helper function to format date strings ---
	function formatDate(dateString: string | undefined | null): string {
		if (!dateString) return 'N/A';
		try {
			return new Date(dateString).toLocaleString(undefined, {
				dateStyle: 'medium',
				timeStyle: 'short'
			});
		} catch (e) {
			console.error('Error formatting date:', dateString, e);
			return 'Invalid Date';
		}
	}

	// --- Placeholder functions for button actions ---
	function handleAddNewRule() {
		console.log('Add New Rule button clicked!');
		showRuleBuilderModal = true;
		// No need to log showRuleBuilderModal here, Svelte handles reactivity
	}
	function handleDeleteRule(ruleId: string) {
		alert(`Delete Rule ${ruleId} functionality coming soon!`);
		// TODO: Add API call to delete rule, then refresh data
	}
	// --- API Keys Modal ---
	let showCreateKeyModal = false;
	let newKeyName = '';
	let newKeyExpiry = '';
	let createKeyLoading = false;
	let createKeyError = '';
	let createdSecretKey: string | null = null; // shown once

	function handleGenerateNewKey() {
		showCreateKeyModal = true;
		newKeyName = '';
		newKeyExpiry = '';
		createdSecretKey = null;
		createKeyError = '';
	}

	async function createApiKey() {
		if (!data.organizationId) {
			createKeyError = 'Missing organization id';
			return;
		}
		if (!newKeyName.trim()) {
			createKeyError = 'Name is required';
			return;
		}
		createKeyLoading = true;
		createKeyError = '';
		try {
			const token = localStorage.getItem('aura_token');
			if (!token) {
				createKeyError = 'Authentication token missing. Please log in again.';
				createKeyLoading = false;
				return;
			}
			const body: any = { name: newKeyName.trim() };
			if (newKeyExpiry) {
				// Expecting YYYY-MM-DD format
				body.expires_at = new Date(newKeyExpiry).toISOString();
			}
			const res = await fetch(`${API_BASE}/organizations/${data.organizationId}/apikeys`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', ...authHeaders(token) },
				body: JSON.stringify(body)
			});
			const json = await res.json();
			if (!res.ok) throw new Error(json.error || `HTTP ${res.status}`);
			createdSecretKey = json.secret_key;
		} catch (e: any) {
			createKeyError = e.message || 'Failed to create key';
		} finally {
			createKeyLoading = false;
		}
	}

	function closeCreateKeyModal() {
		showCreateKeyModal = false;
		createdSecretKey = null;
		window.location.reload();
	}
	async function handleRevokeKey(keyId: string) {
		if (!confirm('Revoke this API key? This cannot be undone.')) return;
		try {
			const token = localStorage.getItem('aura_token');
			if (!token) {
				alert('Authentication token missing. Please log in again.');
				return;
			}
			const res = await fetch(`${API_BASE}/organizations/${data.organizationId}/apikeys/${keyId}`, {
				method: 'DELETE',
				headers: authHeaders(token)
			});
			if (!res.ok && res.status !== 204) {
				const json = await res.json().catch(() => ({}));
				throw new Error(json.error || `HTTP ${res.status}`);
			}
			window.location.reload();
		} catch (e) {
			alert('Failed to revoke key');
		}
	}

	// --- Reactive statement to UPDATE the shared store ---
	// This runs whenever the 'data' prop changes (e.g., after the load function finishes)
	$: {
		if (data && data.agent && typeof data.agent === 'object' && !Array.isArray(data.agent)) {
			// If we have valid single agent data, update the store
			agentContext.set({
				agentId: data.agent.id,
				organizationId: data.agent.organization_id
			});
			// console.log('Agent context store updated:', $agentContext); // Optional log
		} else {
			// If agent data is missing, invalid, or an array, reset the store
			agentContext.set({ agentId: null, organizationId: null });
			if (data && data.agent) { // Log if it's invalid type
				console.error('Error in reactive update: data.agent is not a single object:', data.agent);
			}
		}
	}
	// --- End store update ---
</script>

<div class="max-w-5xl mx-auto">
	<div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4 mb-6">
		{#if data.error && !data.agent}
			<div>
				<h2 class="text-3xl font-bold text-red-400">Error Loading Agent</h2>
				<p class="text-sm text-gray-400 font-mono mt-1 break-all">{data.agentId ?? 'ID Unavailable'}</p>
				<Alert variant="error">{data.error}</Alert>
			</div>
		{:else if data.agent}
			<div>
				<h2 class="text-3xl font-bold text-white">{data.agent.name}</h2>
				<p class="text-sm text-gray-400 font-mono mt-1 break-all">{data.agent.id}</p>
				{#if data.agent.description}
					<p class="text-sm text-gray-300 mt-2 max-w-xl">{data.agent.description}</p>
				{/if}
			</div>
			<div
				class="flex items-center gap-2 bg-green-500/20 text-green-400 text-sm font-medium px-3 py-1 rounded-full self-start md:self-center shrink-0"
			>
				<span class="size-2 bg-green-400 rounded-full"></span>
				Active </div>
		{:else}
			<div>
				<div class="h-8 w-48 bg-gray-700 rounded animate-pulse mb-2"></div>
				<div class="h-4 w-72 bg-gray-700 rounded animate-pulse"></div>
			</div>
			<div
				class="w-20 h-6 bg-gray-700 rounded-full animate-pulse self-start md:self-center shrink-0"
			></div>
		{/if}
	</div>

	{#if data.agent}
		<div class="border-b border-gray-700 mb-6">
			<nav aria-label="Tabs" class="-mb-px flex space-x-6">
				<button
					type="button"
					class="shrink-0 border-b-2 px-1 pb-3 text-sm font-medium transition-colors {activeTab ===
					'rules'
						? 'border-[#7C3AED] text-[#7C3AED] font-semibold'
						: 'border-transparent text-gray-400 hover:border-gray-600 hover:text-gray-200'}"
					on:click={() => (activeTab = 'rules')}> Rules </button
				>
				<button
					type="button"
					class="shrink-0 border-b-2 px-1 pb-3 text-sm font-medium transition-colors {activeTab ===
					'apikeys'
						? 'border-[#7C3AED] text-[#7C3AED] font-semibold'
						: 'border-transparent text-gray-400 hover:border-gray-600 hover:text-gray-200'}"
					on:click={() => (activeTab = 'apikeys')}> API Keys </button
				>
				<button
					type="button"
					class="shrink-0 border-b-2 px-1 pb-3 text-sm font-medium transition-colors {activeTab ===
					'logs'
						? 'border-[#7C3AED] text-[#7C3AED] font-semibold'
						: 'border-transparent text-gray-400 hover:border-gray-600 hover:text-gray-200'}"
					on:click={() => (activeTab = 'logs')}> Logs </button
				>
			</nav>
		</div>

		<div id="tab-content">
			{#if activeTab === 'rules'}
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-xl font-semibold text-white">Permission Rules</h3>
						<button
							class="inline-flex items-center justify-center rounded-lg h-9 px-3 bg-[#7C3AED] text-xs font-medium text-white shadow-sm hover:bg-[#6d28d9] transition-colors"
							on:click={handleAddNewRule}> + Add New Rule </button
						>
					</div>
					{#if data.rules && data.rules.length > 0}
						<div class="overflow-hidden rounded-lg border border-gray-700">
							<table class="min-w-full divide-y divide-gray-700">
								<thead class="bg-gray-800/50">
									<tr>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Effect</th>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Rule Logic (JSON)</th>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Created</th>
										<th scope="col" class="relative px-4 py-2"><span class="sr-only">Actions</span></th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-800 bg-[#1A1A1A]">
									{#each data.rules as rule (rule.id)}
										{@const effect = rule.rule?.effect}
										<tr>
											<td class="px-4 py-3 whitespace-nowrap text-sm">
												{#if effect === 'allow'}
													<span class="inline-flex items-center rounded-full bg-green-900/40 px-2.5 py-0.5 text-xs font-medium text-green-300">ALLOW</span>
												{:else if effect === 'deny'}
													<span class="inline-flex items-center rounded-full bg-red-900/40 px-2.5 py-0.5 text-xs font-medium text-red-300">DENY</span>
												{:else}
													<span class="inline-flex items-center rounded-full bg-gray-700 px-2.5 py-0.5 text-xs font-medium text-gray-300">UNKNOWN</span>
												{/if}
											</td>
											<td class="px-4 py-3 text-sm text-gray-300 font-mono">
												{#key rule.id}
													{@const ruleJsonString = (() => { try { return JSON.stringify(rule.rule, null, 2); } catch { return 'Invalid JSON'; } })()}
													<pre class="whitespace-pre-wrap max-w-md overflow-x-auto text-xs p-2 bg-[#111111] rounded"><code>{ruleJsonString}</code></pre>
												{/key}
											</td>
											<td class="px-4 py-3 whitespace-nowrap text-sm text-gray-400">{formatDate(rule.created_at)}</td>
											<td class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
												<button class="text-red-500 hover:text-red-400 transition-colors" on:click={() => handleDeleteRule(rule.id)}>Delete</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="text-gray-400 text-sm italic py-4">No rules defined for this agent yet.</p>
					{/if}
				</div>
			{:else if activeTab === 'apikeys'}
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-xl font-semibold text-white">API Keys</h3>
						<button
							class="inline-flex items-center justify-center rounded-lg h-9 px-3 bg-[#7C3AED] text-xs font-medium text-white shadow-sm hover:bg-[#6d28d9] transition-colors"
							on:click={handleGenerateNewKey}> + Generate New Key </button
						>
					</div>
					{#if data.apiKeys && data.apiKeys.length > 0}
						<div class="overflow-hidden rounded-lg border border-gray-700">
							<table class="min-w-full divide-y divide-gray-700">
								<thead class="bg-gray-800/50">
									<tr>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Name</th>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Prefix</th>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Created</th>
										<th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Last Used</th>
										<th scope="col" class="relative px-4 py-2"><span class="sr-only">Actions</span></th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-800 bg-[#1A1A1A]">
									{#each data.apiKeys as apiKey (apiKey.id)}
										<tr>
											<td class="px-4 py-3 whitespace-nowrap text-sm font-medium text-white">{apiKey.name || 'Untitled Key'}</td>
											<td class="px-4 py-3 whitespace-nowrap text-sm text-gray-300 font-mono">{apiKey.key_prefix}...</td>
											<td class="px-4 py-3 whitespace-nowrap text-sm text-gray-400">{formatDate(apiKey.created_at)}</td>
											<td class="px-4 py-3 whitespace-nowrap text-sm text-gray-400">{formatDate(apiKey.last_used_at)}</td>
											<td class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
												<button class="text-red-500 hover:text-red-400 transition-colors" on:click={() => handleRevokeKey(apiKey.id)}>Revoke</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="text-gray-400 text-sm italic py-4">No API keys generated for this organization yet.</p>
					{/if}
				</div>
			{:else if activeTab === 'logs'}
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-xl font-semibold text-white">Event Logs</h3>
					</div>
					{#if data.logs && data.logs.length > 0}
						<div class="overflow-hidden rounded-lg border border-gray-700">
							<table class="min-w-full divide-y divide-gray-700">
								<thead class="bg-gray-800/50">
									<tr>
										<th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Timestamp</th>
										<th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Decision</th>
										<th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Reason</th>
										<th class="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Context</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-gray-800 bg-[#1A1A1A]">
									{#each data.logs as log (log.id)}
										<tr>
											<td class="px-4 py-3 whitespace-nowrap text-sm text-gray-400">{formatDate(log.timestamp)}</td>
											<td class="px-4 py-3 whitespace-nowrap text-sm">
												{#if log.decision === 'ALLOWED'}
													<span class="inline-flex items-center rounded-full bg-green-900/40 px-2.5 py-0.5 text-xs font-medium text-green-300">ALLOWED</span>
												{:else if log.decision === 'DENIED'}
													<span class="inline-flex items-center rounded-full bg-red-900/40 px-2.5 py-0.5 text-xs font-medium text-red-300">DENIED</span>
												{:else}
													<span class="inline-flex items-center rounded-full bg-gray-700 px-2.5 py-0.5 text-xs font-medium text-gray-300">{log.event_type}</span>
												{/if}
											</td>
											<td class="px-4 py-3 whitespace-pre-wrap text-xs text-gray-300">{log.decision_reason || ''}</td>
											<td class="px-4 py-3 text-xs text-gray-300 font-mono">
												<pre class="whitespace-pre-wrap max-w-lg overflow-x-auto"><code>{JSON.stringify(log.request_details, null, 2)}</code></pre>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="text-gray-400 text-sm italic py-4">No event logs found for this agent yet.</p>
					{/if}
				</div>
			{/if}
		</div>
	{/if} {#if data.agent}
		<RuleBuilderModal
			title="Add New Rule"
			bind:showModal={showRuleBuilderModal}
			on:close={() => (showRuleBuilderModal = false)}
			on:save={(event) => {
				console.log('Rule Saved event received (from parent):', event.detail);
				window.location.reload(); // Simple reload for MVP
			}}
		/>
	{/if}

	<!-- Create API Key Modal -->
	{#if showCreateKeyModal}
		<Modal title="Generate API Key" bind:showModal={showCreateKeyModal} on:close={closeCreateKeyModal}>
			{#if createdSecretKey}
				<div class="space-y-3">
					<p class="text-sm text-gray-300">Copy this secret now. You won’t be able to see it again.</p>
					<div class="p-3 bg-[#111111] border border-[#333333] rounded font-mono text-xs break-all">{createdSecretKey}</div>
					<div class="flex justify-end">
						<button class="px-4 py-2 text-sm font-medium text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded" on:click={closeCreateKeyModal}>Done</button>
					</div>
				</div>
			{:else}
				<form on:submit|preventDefault={createApiKey} class="space-y-4">
					<div>
						<label class="block text-sm text-gray-300" for="keyName">Name</label>
						<input id="keyName" class="mt-1 w-full p-2 bg-[#111111] text-white rounded border border-[#333333] text-sm" bind:value={newKeyName} placeholder="e.g., Production Key" />
					</div>
					<div>
						<label class="block text-sm text-gray-300" for="keyExpiry">Expiry (optional)</label>
						<input id="keyExpiry" type="date" class="mt-1 w-full p-2 bg-[#111111] text-white rounded border border-[#333333] text-sm" bind:value={newKeyExpiry} />
					</div>
					{#if createKeyError}
						<Alert variant="error">{createKeyError}</Alert>
					{/if}
					<div class="flex justify-end gap-3">
						<button type="button" class="px-4 py-2 text-sm font-medium text-gray-300 bg-transparent rounded hover:bg-white/10" on:click={() => (showCreateKeyModal = false)}>Cancel</button>
						<button type="submit" class="px-4 py-2 text-sm font-medium text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded disabled:opacity-50" disabled={createKeyLoading}>{createKeyLoading ? 'Creating…' : 'Create Key'}</button>
					</div>
				</form>
			{/if}
		</Modal>
	{/if}

	</div>



