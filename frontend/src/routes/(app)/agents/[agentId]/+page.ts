import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { error } from '@sveltejs/kit';
import { API_BASE, authHeaders } from '$lib/api';

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

interface EventLog {
	id: number;
	timestamp: string;
	event_type: string;
	decision?: string;
	decision_reason?: string | null;
	request_details: any;
}

export interface AgentDetailPageLoadData {
	agent: Agent | null;
	rules: Permission[];
	apiKeys: ApiKeyInfo[];
	logs: EventLog[];
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
		organizationId = localStorage.getItem('aura_org_id');
	}

	if (!token || !organizationId) {
		console.error('No token or OrgID found for loading agent details');
		throw error(401, 'Not authorized. Please log in.');
	}

	let agent: Agent | null = null;
	let rules: Permission[] = [];
	let apiKeys: ApiKeyInfo[] = [];
	let logs: EventLog[] = [];
	let loadError: string | undefined = undefined;

	try {
		// --- Fetch data concurrently ---
		const results = await Promise.all([
				fetch(`${API_BASE}/organizations/${organizationId}/agents/${agentId}`, {
					headers: authHeaders(token)
				}),
				fetch(`${API_BASE}/organizations/${organizationId}/agents/${agentId}/permissions`, {
					headers: authHeaders(token)
				}),
				fetch(`${API_BASE}/organizations/${organizationId}/apikeys`, {
					headers: authHeaders(token)
				}),
				fetch(`${API_BASE}/organizations/${organizationId}/logs?agentId=${agentId}`, {
					headers: authHeaders(token)
				})
			]);

		const agentRes = results[0];
		const rulesRes = results[1];
	const keysRes = results[2];
	const logsRes = results[3];

		// --- Process Agent Details ---
		if (!agentRes.ok) {
			if (agentRes.status === 404) throw error(404, 'Agent not found');
			const agentErrorData = await agentRes.json().catch(() => ({ error: 'Failed to parse agent error' }));
			console.error(`Failed to fetch agent details: ${agentRes.status}`, agentErrorData);
			loadError = agentErrorData.error || `Failed to load agent (status: ${agentRes.status})`;
		} else {
			agent = await agentRes.json();
		}
		// --- End Agent Processing ---

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

		if (!logsRes.ok) {
			console.error(`Failed to fetch logs: ${logsRes.status}`);
		} else {
			logs = await logsRes.json();
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
	logs,
		agentId,
		organizationId,
		error: loadError
	};
};

export const ssr = false;