import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';

export const load = () => {
  // Server-side: default to login (can't read localStorage)
  if (!browser) {
    throw redirect(302, '/login');
  }
  // Client-side: prefer dashboard if token is present
  const token = localStorage.getItem('aura_token');
  if (token) {
    throw redirect(302, '/dashboard');
  }
  throw redirect(302, '/login');
};

