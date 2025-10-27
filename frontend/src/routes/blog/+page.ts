import { getAllBlogPosts } from '$lib/content/blog';

export const load = async () => {
  const md = getAllBlogPosts();
  const posts = md.map(p => ({ slug: p.slug, title: p.title, date: p.date, summary: (p as any).summary || (p as any).description }));
  return { posts };
};
