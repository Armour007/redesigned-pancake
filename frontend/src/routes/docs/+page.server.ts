import type { Load } from './$types';
import { redirect } from '@sveltejs/kit';
import { API_BASE } from '$lib/api';

export const load: Load = async () => {
  // Redirect to backend-hosted Swagger UI
  throw redirect(302, `${API_BASE.replace(/\/$/, '')}/docs`);
};
