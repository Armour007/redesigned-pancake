import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { API_BASE } from '$lib/api';

export const ssr = false;

type EventLog = {
  id: string;
  created_at: string;
  event_type?: string;
  agent_id?: string | null;
  request_id?: string | null;
  request_details?: any;
};

export const load: Load = async ({ fetch }) => {
  let token: string | null = null;
  let orgId: string | null = null;
  if (browser) {
    token = localStorage.getItem('aura_token');
    orgId = localStorage.getItem('aura_org_id');
  }
  if (!token || !orgId) {
    return { logs: [], error: 'Not authenticated' };
  }
  try {
    const res = await fetch(`${API_BASE}/organizations/${orgId}/logs`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const logs: EventLog[] = await res.json();
    return { logs };
  } catch (e: any) {
    return { logs: [], error: e?.message ?? 'Failed to load logs' };
  }
};
 
