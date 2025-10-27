import type { Load } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { API_BASE } from '$lib/api';

export const ssr = false;

export const load: Load = async ({ fetch }) => {
  let token: string | null = null;
  let orgId: string | null = null;
  if (browser) {
    token = localStorage.getItem('aura_token');
    orgId = localStorage.getItem('aura_org_id');
  }
  if (!token || !orgId) return { me: null, org: null, error: 'Not authenticated' };

  try {
    const [meRes, orgRes] = await Promise.all([
      fetch(`${API_BASE}/me`, { headers: { Authorization: `Bearer ${token}` } }),
      fetch(`${API_BASE}/organizations/${orgId}`, { headers: { Authorization: `Bearer ${token}` } })
    ]);
    const me = meRes.ok ? await meRes.json() : null;
    const org = orgRes.ok ? await orgRes.json() : null;
    return { me, org };
  } catch (e: any) {
    return { me: null, org: null, error: e?.message ?? 'Failed to load settings' };
  }
};
