import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { API_BASE } from '$lib/api';

export const ssr = false;

type ApiKey = {
  id: string;
  name?: string;
  key_prefix: string;
  created_at: string;
  last_used_at?: string | null;
  expires_at?: string | null;
  revoked_at?: string | null;
};

export const load: Load = async ({ fetch }) => {
  let token: string | null = null;
  let orgId: string | null = null;
  if (browser) {
    token = localStorage.getItem('aura_token');
    orgId = localStorage.getItem('aura_org_id');
  }
  if (!token || !orgId) {
    return { keys: [], orgId: orgId ?? undefined, error: 'Not authenticated' };
  }
  try {
    const res = await fetch(`${API_BASE}/organizations/${orgId}/apikeys`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const keys: ApiKey[] = await res.json();
    return { keys, orgId };
  } catch (e: any) {
    return { keys: [], orgId, error: e?.message ?? 'Failed to load API keys' };
  }
};
