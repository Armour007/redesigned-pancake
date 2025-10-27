import type { PageLoad } from './$types';
import { posts } from '$lib/blog/posts';

export const load: PageLoad = ({ params }) => {
  const post = posts.find((p) => p.slug === params.slug) ?? null;
  return { post };
};
