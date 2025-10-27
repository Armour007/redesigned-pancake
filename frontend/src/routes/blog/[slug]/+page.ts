import { getAllBlogPosts } from '$lib/content/blog';

export const load = ({ params }) => {
  const content = getAllBlogPosts();
  const md = content.find((p) => p.slug === params.slug);
  const post = md ? { slug: md.slug, title: md.title, date: md.date, html: '', component: md.component } : null;
  return { post };
};
