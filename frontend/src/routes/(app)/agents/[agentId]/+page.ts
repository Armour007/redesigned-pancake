import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { error } from '@sveltejs/kit';

// --- Type Definitions ---
interface Agent {
	id: string;
	organization_id: string;
	name: string;
	description?: string | null;
	created_at: string;
}

interface Permission {
	id: string;
	agent_id: string;
	rule: any;
	is_active: boolean;
	created_at: string;
}

interface ApiKeyInfo {
	id: string;
	name: string;
	key_prefix: string;
	created_at: string;
	last_used_at?: string | null;
	expires_at?: string | null;
}

export interface AgentDetailPageLoadData {
	agent: Agent | null;
	rules: Permission[];
	apiKeys: ApiKeyInfo[];
	agentId: string;
	organizationId: string | null;
	error?: string;
	[key: string]: any; // Index signature
}

// --- Load Function ---
export const load: Load = async ({ params, fetch }) => {
	const agentId = params.agentId;
	let token: string | null = null;
	let organizationId: string | null = null; // Needs replacement

	if (browser) {
		token = localStorage.getItem('aura_token');
		// !!! CRITICAL: REPLACE THIS HARDCODED ORGANIZATION ID !!!
		organizationId = '2bc40ca7-7830-4e3a-8f17-daf017247bb9'; // <<<--- REPLACE THIS !!!
	}

	if (!token || !organizationId) {
		console.error('No token or OrgID found for loading agent details');
		throw error(401, 'Not authorized. Please log in.');
	}

	let agent: Agent | null = null;
	let rules: Permission[] = [];
	let apiKeys: ApiKeyInfo[] = [];
	let loadError: string | undefined = undefined;

	try {
		// --- Fetch data concurrently ---
		const results = await Promise.all([
			fetch(`http://localhost:8080/organizations/${organizationId}/agents/${agentId}`, {
				headers: { Authorization: `Bearer ${token}` }
			}),
			fetch(`http://localhost:8080/organizations/${organizationId}/agents/${agentId}/permissions`, {
				headers: { Authorization: `Bearer ${token}` }
			}),
			fetch(`http://localhost:8080/organizations/${organizationId}/apikeys`, {
				headers: { Authorization: `Bearer ${token}` }
			})
		]);

		const agentRes = results[0];
		const rulesRes = results[1];
		const keysRes = results[2];

		// --- Process Agent Details (FIXED: Handle ARRAY response) ---
		if (!agentRes.ok) {
			if (agentRes.status === 404) throw error(404, 'Agent not found');
			const agentErrorData = await agentRes.json().catch(() => ({ error: 'Failed to parse agent error' }));
			console.error(`Failed to fetch agent details: ${agentRes.status}`, agentErrorData);
			loadError = agentErrorData.error || `Failed to load agent (status: ${agentRes.status})`;
		} else {
			const agentArray: Agent[] = await agentRes.json(); // Parse as an array
			console.log("Parsed agent ARRAY from API:", agentArray); // Log the array
			if (agentArray && agentArray.length > 0) {
				 agent = agentArray[0]; // Take the FIRST element from the array
				 console.log("Extracted single agent:", agent); // Log the extracted object
			} else {
				 console.error("Agent API returned OK but array was empty or invalid.");
				 loadError = "Agent data received in unexpected format (empty array)."
			}
		}
		// --- End Agent Processing Fix ---

		// --- Process Rules ---
		if (!rulesRes.ok) {
			console.error(`Failed to fetch rules: ${rulesRes.status}`);
		} else {
			rules = await rulesRes.json();
		}

		// --- Process API Keys ---
		if (!keysRes.ok) {
			console.error(`Failed to fetch API keys: ${keysRes.status}`);
		} else {
			apiKeys = await keysRes.json();
		}

	} catch (err: any) {
		console.error('Critical error loading agent detail page data:', err);
		if (err.status) throw err;
		throw error(500, err.message || 'Unknown server error fetching agent detail data.');
	}

	// --- Return all data ---
	return {
		agent, // This should now be the single agent object or null
		rules,
		apiKeys,
		agentId,
		organizationId,
		error: loadError
	};
};

export const ssr = false;