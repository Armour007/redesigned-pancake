<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import Modal from '$lib/components/Modal.svelte'; // Assuming Modal.svelte is in components
	import { agentContext } from '$lib/stores/agentContext'; // <-- Import the store

	// --- REMOVED agentId and organizationId props ---
	export let showModal: boolean = false;
	export let title: string = 'Modal Title';

	const dispatch = createEventDispatcher();

	// --- Form State Variables ---
	let ruleType: 'template' | 'custom' = 'template';
	let effect: 'allow' | 'deny' = 'allow';
	let action = ''; // Initialize action state
	let conditions: Array<{ key: string; operator: string; value: string }> = [];
	let customJson = '';
	let jsonPreview = '{}'; // Start with empty object
	let errorMessage = '';
	let isLoading = false; // For API call later

	// --- onMount ---
	// Resets form state when the component mounts
	onMount(() => {
		resetFormState();
		console.log('RuleBuilderModal mounted, form state reset.');
	});

	// --- Reactive block for Rule Type Change ---
	$: {
		// This runs whenever ruleType changes
		if (ruleType === 'template') {
			customJson = '';
			// console.log('Switched to template mode, cleared custom JSON.');
		} else if (ruleType === 'custom') {
			resetGuidedFields();
			// console.log('Switched to custom mode, cleared guided fields.');
		}
		errorMessage = ''; // Reset error on switch
	}

	// --- Constants ---
	const operators = [
		{ id: 'eq', name: 'is equal to' },
		{ id: 'neq', name: 'is not equal to' },
		{ id: 'gt', name: 'is greater than' },
		{ id: 'gte', name: 'is greater than or equal to' },
		{ id: 'lt', name: 'is less than' },
		{ id: 'lte', name: 'is less than or equal to' },
		{ id: 'contains', name: 'contains' }
	];

	// --- Helper Functions ---
	function addCondition() {
		conditions = [...conditions, { key: '', operator: 'eq', value: '' }];
	}
	function removeCondition(index: number) {
		conditions = conditions.filter((_, i) => i !== index);
	}

	function resetGuidedFields() {
		effect = 'allow';
		action = '';
		conditions = [];
	}

	function resetFormState() {
		ruleType = 'template';
		resetGuidedFields();
		customJson = '';
		jsonPreview = '{}';
		errorMessage = '';
		isLoading = false;
	}

	// --- Reactive statement to generate JSON preview ---
	$: {
		// This block runs whenever its dependencies (ruleType, customJson, effect, action, conditions) change
		errorMessage = ''; // **FIX: Reset error message at the start of EVERY recalculation**
		try {
			if (ruleType === 'custom') {
				// Handle Custom JSON Input
				if (!customJson || customJson.trim() === '') {
					jsonPreview = '{}';
				} else {
					const parsed = JSON.parse(customJson);
					jsonPreview = JSON.stringify(parsed, null, 2);
				}
			} else {
				// Generate JSON from Guided Builder
				const ruleObject: any = {
					effect: effect.toLowerCase()
				};
				const trimmedAction = action.trim();
				if (trimmedAction) {
					ruleObject.action = trimmedAction;
				} else {
					// Set error if action is empty, as it's required
					errorMessage = 'Action field is required.';
				}

				// Process conditions
				const validConditions = conditions.filter((cond) => cond.key.trim() && cond.value.trim());
				if (validConditions.length > 0) {
					ruleObject.context = {};
					if (validConditions.length === 1) {
						const cond = validConditions[0];
						const operator = cond.operator || 'eq';
						ruleObject.context[cond.key.trim()] = { [operator]: cond.value.trim() };
					} else {
						ruleObject.context['AND'] = validConditions.map((cond) => {
							const operator = cond.operator || 'eq';
							return { [cond.key.trim()]: { [operator]: cond.value.trim() } };
						});
					}
				}
				// Generate Final Preview
				jsonPreview = JSON.stringify(ruleObject, null, 2);
				// Set preview text if action is missing, but error is already set
				if (!trimmedAction) {
					jsonPreview = '// Action is required';
				}
			} // End Guided Builder else
		} catch (e: any) {
			if (ruleType === 'custom') {
				errorMessage = 'Invalid Custom JSON syntax.';
			} else {
				errorMessage = 'Error generating rule preview.';
				console.error('Error in reactive rule generation:', e);
			}
			jsonPreview = '// Invalid Input';
		}
	} // End Reactive Block

	// --- API Call Logic (Using Store) ---
	async function handleSave() {
		const currentAgentContext = $agentContext;
		const agentIdFromStore = currentAgentContext.agentId;
		const orgIdFromStore = currentAgentContext.organizationId;

		// Validation
		if (!agentIdFromStore || !orgIdFromStore) {
			errorMessage = 'Agent or Organization ID context is missing. Cannot save.';
			console.error('Missing IDs from store in handleSave:', { agentIdFromStore, orgIdFromStore });
			return;
		}

		isLoading = true;
		// Re-check error message *at the moment of saving*
		if (errorMessage) {
			isLoading = false;
			return; // Don't proceed if there's a known error
		}

		const token = localStorage.getItem('aura_token');
		if (!token) {
			errorMessage = 'Authentication token missing. Please log in again.';
			isLoading = false;
			return;
		}

		let ruleToSend: any;
		try {
			ruleToSend = JSON.parse(jsonPreview);
		} catch (e) {
			errorMessage = 'Cannot save invalid JSON rule.';
			isLoading = false;
			return;
		}

		if (!ruleToSend || !ruleToSend.action) {
			errorMessage = 'Rule must include a valid "action". Check inputs.';
			isLoading = false;
			return;
		}

		// --- API Call ---
		try {
			const response = await fetch(
				`http://localhost:8080/organizations/${orgIdFromStore}/agents/${agentIdFromStore}/permissions`,
				{
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
						Authorization: `Bearer ${token}`
					},
					body: JSON.stringify({ rule: ruleToSend })
				}
			);
			const responseData = await response.json();
			if (!response.ok) {
				throw new Error(responseData.error || `HTTP error! status: ${response.status}`);
			}
			console.log('Rule saved:', responseData);
			dispatch('save', responseData);
			closeModal();
		} catch (error: any) {
			errorMessage = error.message || 'Failed to save rule. Please try again.';
			console.error('Save rule error:', error);
		} finally {
			isLoading = false;
		}
	}

	// --- Close Modal Logic ---
	function closeModal() {
		resetFormState(); // Use the dedicated reset function
		dispatch('close');
	}
</script>

<Modal {title} bind:showModal on:close={closeModal}>
	<form on:submit|preventDefault={handleSave} class="space-y-4">
		<div class="space-y-1">
			<label class="block text-sm font-medium text-gray-300">Rule Type</label>
			<div class="flex rounded-lg bg-gray-800 p-1">
				<label class="w-full">
					<input bind:group={ruleType} type="radio" value="template" class="sr-only peer" checked />
					<span
						class="block text-center py-2 px-4 rounded-md cursor-pointer text-sm font-semibold transition-colors duration-200 text-gray-400 peer-checked:bg-[#7C3AED] peer-checked:text-white"
					>
						Guided Builder
					</span>
				</label>
				<label class="w-full">
					<input bind:group={ruleType} type="radio" value="custom" class="sr-only peer" />
					<span
						class="block text-center py-2 px-4 rounded-md cursor-pointer text-sm font-semibold transition-colors duration-200 text-gray-400 peer-checked:bg-[#7C3AED] peer-checked:text-white"
					>
						Custom JSON
					</span>
				</label>
			</div>
		</div>

		{#if ruleType === 'template'}
			<div class="space-y-4 border border-gray-700 p-4 rounded-lg bg-gray-800/30">
				<div class="space-y-1">
					<label class="block text-sm font-medium text-gray-300">Effect</label>
					<div class="flex rounded-lg bg-gray-700 p-1 w-full">
						<label class="w-1/2">
							<input bind:group={effect} type="radio" value="allow" class="sr-only peer" checked />
							<span
								class="block text-center py-1.5 px-3 rounded text-xs font-semibold transition-colors duration-200 text-gray-300 peer-checked:bg-green-700 peer-checked:text-white"
								>ALLOW</span
							>
						</label>
						<label class="w-1/2">
							<input bind:group={effect} type="radio" value="deny" class="sr-only peer" />
							<span
								class="block text-center py-1.5 px-3 rounded text-xs font-semibold transition-colors duration-200 text-gray-300 peer-checked:bg-red-700 peer-checked:text-white"
								>DENY</span
							>
						</label>
					</div>
				</div>

				<div>
					<label for="action" class="block text-sm font-medium text-gray-300"
						>Action <span class="text-red-400">*</span></label
					>
					<input
						id="action"
						type="text"
						bind:value={action}
						required
						class="mt-1 block w-full p-2 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 text-sm"
						placeholder="e.g., deploy:prod, read:db"
					/>
				</div>

				<div>
					<label class="block text-sm font-medium text-gray-300 mb-2"
						>Conditions (Optional, all must be true)</label
					>
					{#each conditions as condition, i (i)}
						<div class="flex items-center gap-2 mb-2 p-2 bg-gray-700/50 rounded">
							<input
								type="text"
								bind:value={condition.key}
								placeholder="Key (e.g., branch)"
								class="flex-grow p-1.5 bg-[#111111] text-white rounded border border-[#333333] focus:ring-1 focus:ring-[#7C3AED] text-xs"
							/>
							<select
								bind:value={condition.operator}
								class="p-1.5 bg-[#111111] text-white rounded border border-[#333333] focus:ring-1 focus:ring-[#7C3AED] text-xs appearance-none"
							>
								{#each operators as op}
									<option value={op.id}>{op.name}</option>
								{/each}
							</select>
							<input
								type="text"
								bind:value={condition.value}
								placeholder="Value (e.g., main)"
								class="flex-grow p-1.5 bg-[#111111] text-white rounded border border-[#333333] focus:ring-1 focus:ring-[#7C3AED] text-xs"
							/>
							<button
								type="button"
								on:click={() => removeCondition(i)}
								class="text-red-400 hover:text-red-300 p-1"
								title="Remove condition"
							>
								<span class="material-symbols-outlined text-base leading-none">delete</span>
							</button>
						</div>
					{/each}
					<button
						type="button"
						on:click={addCondition}
						class="mt-1 text-xs text-[#7C3AED] hover:text-[#9a6aff] flex items-center gap-1"
					>
						<span class="material-symbols-outlined text-sm">add_circle</span> Add Condition
					</button>
				</div>
			</div>
		{/if}

		{#if ruleType === 'custom'}
			<div class="space-y-1">
				<label for="customJson" class="block text-sm font-medium text-gray-300"
					>Rule Logic (JSON) <span class="text-red-400">*</span></label
				>
				<textarea
					id="customJson"
					bind:value={customJson}
					rows="6"
					required
					class="mt-1 block w-full p-3 bg-[#111111] text-white rounded-lg border border-[#333333] focus:ring-2 focus:ring-[#7C3AED] focus:border-transparent placeholder-gray-500 font-mono text-xs resize-y"
					placeholder={`{ "effect": "allow", "action": "read", "context": { "resource": "/data/*" } }`}
				></textarea>
			</div>
		{/if}

		<div class="space-y-1">
			<label class="block text-xs font-medium text-gray-400">Live JSON Preview</label>
			<pre
				class="w-full p-3 bg-gray-800 text-gray-300 rounded-lg border font-mono text-xs overflow-x-auto whitespace-pre-wrap {errorMessage &&
				(ruleType === 'custom' || errorMessage.includes('Action'))
					? 'border-red-500'
					: 'border-gray-700'}"
			><code>{jsonPreview}</code></pre>
		</div>

		{#if errorMessage}
			<p class="text-sm text-red-400">{errorMessage}</p>
		{/if}

		<div class="flex justify-end pt-2 space-x-3">
			<button
				type="button"
				disabled={isLoading}
				on:click={closeModal}
				class="px-4 py-2 text-sm font-medium text-gray-300 bg-transparent rounded-lg hover:bg-white/10 disabled:opacity-50"
			>
				Cancel
			</button>
			<button
				type="submit"
				disabled={isLoading || errorMessage !== ''}
				class="px-4 py-2 text-sm font-medium text-white bg-[#7C3AED] hover:bg-[#6d28d9] rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
			>
				{#if isLoading} Saving... {:else} Save Rule {/if}
			</button>
		</div>
	</form>
</Modal>