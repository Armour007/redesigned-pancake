import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment'; // To check if running in browser
import { API_BASE, authHeaders } from '$lib/api';

// Define the shape of an Agent based on your backend response
interface Agent {
	id: string; // UUIDs are strings in JSON
	organization_id: string;
	name: string;
	description?: string | null;
	created_at: string; // Timestamps are strings in JSON
}

export const load: Load = async ({ fetch }) => {
	// --- IMPORTANT: Get Org ID and Token ---
	// This is a placeholder. In a real app, you need a secure way
	// to get the user's token and their associated organization ID.
	// We'll read the token from localStorage (only works in browser) for MVP.
	let token: string | null = null;
	let organizationId: string | null = null;

	if (browser) { // localStorage only exists in the browser
	token = localStorage.getItem('aura_token');
	organizationId = localStorage.getItem('aura_org_id');
	}

	if (!token || !organizationId) {
		// If no token or orgId, we can't load data.
        // The layout's onMount should handle redirection, but we can return error too.
        console.error('No token or OrgID found for loading agents');
		return {
            agents: [] as Agent[],
            error: 'Authentication token or Organization ID missing.',
            organizationId: null
        };
	}

	try {
	    const response = await fetch(`${API_BASE}/organizations/${organizationId}/agents`, {
			method: 'GET',
			headers: {
					...authHeaders(token), // Send the token
			},
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({})); // Try to get error message
			throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
		}

		const agents: Agent[] = await response.json();

		return {
			agents: agents,
            organizationId: organizationId // Pass OrgID to the page component
		};
	} catch (error: any) {
		console.error('Failed to load agents:', error);
		return {
			agents: [] as Agent[],
			error: error.message || 'Unknown error fetching agents.',
            organizationId: organizationId // Still pass OrgID even on error
		};
	}
};

// Enable SSR = false for this page ONLY for the MVP because localStorage
// is only available in the browser. A better V1.0 solution would handle
// auth server-side using hooks and cookies.
export const ssr = false;