import { readable } from 'svelte/store';

export const page = readable({
  params: { orgId: 'org_1' },
  url: new URL('http://localhost/')
});
