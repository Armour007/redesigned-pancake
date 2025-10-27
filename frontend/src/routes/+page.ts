import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';

export const load = () => {
  // Landing page: no redirect. Provide minimal props if needed.
  return {};
};

