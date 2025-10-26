<script lang="ts">
	import type { PageData } from './$types'; // SvelteKit type for load function data
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import Modal from '$lib/components/Modal.svelte'; // Import Modal component

	// Define the Agent type again here to match the data from +page.ts
	// (Or import from a shared types file later: import type { Agent } from '$lib/types';)
	interface Agent {
		id: string;
		organization_id: string;
		name: string;
		description?: string | null;
		created_at: string;
	}

	// The 'data' prop now uses the inferred PageData type from +page.ts
	export let data: PageData;

	// Use optional chaining and nullish coalescing for safety when accessing data
	const organizationId = data?.organizationId ?? null;
	const agents: Agent[] = data?.agents ?? []; // Default to empty array
	const error = data?.error ?? null; // Default to null

	// --- Modal Logic ---
	let showCreateModal = false;
	let newAgentName = '';
	let newAgentDescription = '';
	let createErrorMessage = '';
	let isCreating = false;

	async function handleCreateAgent() {
		// Ensure organizationId exists before proceeding
		if (!organizationId) {
			createErrorMessage = 'Organization ID is missing or could not be loaded.';
			console.error('Organization ID missing in handleCreateAgent');
			return;
		}

		if (!newAgentName.trim()) {
			createErrorMessage = 'Agent name is required.';
			return;
		}

		isCreating = true;
		createErrorMessage = '';
		const token = localStorage.getItem('aura_token'); // Get token

		if (!token) {
			createErrorMessage = 'Authentication token missing. Please log in again.';
			isCreating = false;
			// Optional: Redirect to login
			// await goto('/login');
			return;
		}

		try {
			const response = await fetch(`http://localhost:8080/organizations/${organizationId}/agents`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}` // Send the token
				},
				body: JSON.stringify({
					name: newAgentName,
					// Only send description if it's not empty
					...(newAgentDescription.trim() && { description: newAgentDescription })
				})
			});

			const responseData = await response.json();

			if (!response.ok) {
				throw new Error(responseData.error || `HTTP error! status: ${response.status}`);
			}

			// SUCCESS!
			console.log('Agent created:', responseData);
			showCreateModal = false; // Close modal
			newAgentName = ''; // Reset form
			newAgentDescription = '';
			// Refresh the agent list (simplest way for now is a page reload)
			// A better way involves invalidating load data: import { invalidate } from '$app/navigation'; invalidate(`http://localhost:8080/organizations/${organizationId}/agents`);
			window.location.reload();
		} catch (error: any) {
			createErrorMessage = error.message || 'Failed to create agent. Please try again.';
			console.error('Create agent error:', error);
		} finally {
			isCreating = false;
		}
	}
	// --- End Modal Logic ---
</script>

<div class="space-y-8">
	<div class="flex items-center justify-between">
		<h1 class="text-3xl font-bold text-white">Agents</h1>
		<button
			class="inline-flex items-center justify-center rounded-lg h-10 px-4 bg-[#7C3AED] text-sm font-medium text-white shadow-sm transition-colors hover:bg-[#6d28d9] focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[#7C3AED] focus:ring-offset-[#111111]"
			on:click={() => (showCreateModal = true)}
		>
			+ Create New Agent
		</button>
	</div>

	{#if error}
		<p class="text-red-400">Error loading agents: {error}</p>
	{:else if agents && agents.length > 0}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
{#each agents as agent (agent.id)}
  <a href={`/agents/${agent.id}`} class="block group">
	<div
	  class="bg-[#1A1A1A] rounded-xl p-6 border border-[#333333] transition-all duration-200 group-hover:-translate-y-1 group-hover:shadow-lg group-hover:border-[#7C3AED]/50"
	>
	  <!-- Use group-hover for effects -->
	  <div class="flex justify-between items-start">
		 <!-- ... card content (icon, name, id) ... -->
		 <div class="flex items-center gap-4 min-w-0">
			<!-- ... icon div ... -->
			<div class="min-w-0">
			  <p class="text-base font-bold text-white truncate group-hover:text-[#7C3AED]" title={agent.name || 'Unnamed Agent'}>
				<!-- Added group-hover effect -->
				{agent.name || 'Unnamed Agent'}
			  </p>
			  <p class="text-xs text-gray-400 font-mono mt-1 truncate" title={agent.id}>
				id: {agent.id}
			  </p>
			</div>
		 </div>
		 <!-- ... kebab menu button ... -->
	  </div>
	  <div class="mt-4 flex items-center justify-between">
		 <!-- ... status pill ... -->
	  </div>
	</div>
  </a>
{/each}
		</div>
	{:else}
		<div class="text-center py-12 border-2 border-dashed border-gray-700 rounded-lg">
			<span class="material-symbols-outlined text-5xl text-gray-500">memory</span>
			<h3 class="mt-2 text-sm font-semibold text-gray-400">No agents found</h3>
			<p class="mt-1 text-sm text-gray-500">Get started by creating your first agent.</p>
		</div>
	{/if}
</div> <Modal title="Create New Agent" bind:showModal={showCreateModal} on:close={() => (showCreateModal = false)}>
	<form on:submit|preventDefault={handleCreateAgent} class="space-y-4">
		<div>
			<label for="agentName" class="block text-sm font-medium text-gray-300"
				>Agent Name <span class="text-red-400">*</span></label
			>
			<input
				id="agentName"
				type="text"
				bind:value={newAgentName}
				required
				disabled={isCreating}
				class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
				placeholder="e.g., Production Deployer"
			/>
		</div>
		<div>
			<label for="agentDescription" class="block text-sm font-medium text-gray-300"
				>Description (Optional)</label
			>
			<textarea
				id="agentDescription"
				bind:value={newAgentDescription}
				rows={3}
				disabled={isCreating}
				class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm resize-none"
				placeholder="What does this agent do?"
			/>
		</div>

		{#if createErrorMessage}
			<p class="text-sm text-red-400">{createErrorMessage}</p>
		{/if}

		<div class="flex justify-end pt-2 space-x-3">
			<button
				type="button"
				disabled={isCreating}
				on:click={() => (showCreateModal = false)}
				class="px-4 py-2 text-sm font-medium text-gray-300 bg-transparent rounded-lg hover:bg-white/10 disabled:opacity-50"
			>
				Cancel
			</button>
			<button
				type="submit"
				disabled={isCreating}
				class="px-4 py-2 text-sm font-medium text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded-lg disabled:opacity-50 disabled:cursor-wait transition-colors"
			>
				{#if isCreating}
					Creating...
				{:else}
					Create Agent
				{/if}
			</button>
		</div>
	</form>
</Modal>