import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { API_BASE } from '$lib/api';

export const ssr = false;

function isToday(dateStr: string) {
  const d = new Date(dateStr);
  const now = new Date();
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate();
}

export const load: Load = async ({ fetch }) => {
  let token: string | null = null;
  let orgId: string | null = null;
  if (browser) {
    token = localStorage.getItem('aura_token');
    orgId = localStorage.getItem('aura_org_id');
  }
  if (!token || !orgId) {
    return {
      agentCount: 0,
      apiKeyCount: 0,
      verificationCountToday: 0,
      error: 'Not authenticated'
    };
  }

  try {
    const [agentsRes, keysRes, logsRes] = await Promise.all([
      fetch(`${API_BASE}/organizations/${orgId}/agents`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${API_BASE}/organizations/${orgId}/apikeys`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${API_BASE}/organizations/${orgId}/logs`, { headers: { Authorization: `Bearer ${token}` } })
    ]);

    const agentCount = agentsRes.ok ? (await agentsRes.json()).length : 0;
    const apiKeyCount = keysRes.ok ? (await keysRes.json()).length : 0;
    let verificationCountToday = 0;
    if (logsRes.ok) {
      const logs: any[] = await logsRes.json();
      verificationCountToday = logs.filter((l) => isToday(l.created_at) && (l.event_type === 'verify' || l.event_type === 'verification' || !l.event_type)).length;
    }

    return { agentCount, apiKeyCount, verificationCountToday };
  } catch (e: any) {
    return {
      agentCount: 0,
      apiKeyCount: 0,
      verificationCountToday: 0,
      error: e?.message ?? 'Failed to load dashboard'
    };
  }
};
 
