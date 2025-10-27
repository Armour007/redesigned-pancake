import { getAllBlogPosts } from '$lib/content/blog';
import { posts as fallback } from '$lib/blog/posts';

export const load = async () => {
  const md = getAllBlogPosts();
  const posts = md.length
    ? md.map(p => ({ slug: p.slug, title: p.title, date: p.date, summary: (p as any).summary || (p as any).description }))
    : fallback;
  return { posts };
};
