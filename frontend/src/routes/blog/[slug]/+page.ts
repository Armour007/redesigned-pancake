import { posts } from '$lib/blog/posts';
import { getAllBlogPosts } from '$lib/content/blog';

export const load = ({ params }) => {
  const content = getAllBlogPosts();
  const md = content.find((p) => p.slug === params.slug);
  if (md) {
    return { post: { slug: md.slug, title: md.title, date: md.date, html: '', component: md.component } };
  }
  const post = posts.find((p) => p.slug === params.slug) ?? null;
  return { post };
};
