export type BlogFrontmatter = {
  slug: string;
  title: string;
  date: string; // ISO
  description?: string;
};

export type BlogPost = BlogFrontmatter & {
  component: any;
};

export function getAllBlogPosts(): BlogPost[] {
  const modules = import.meta.glob('/src/content/blog/*.{md,svx}', { eager: true }) as Record<string, any>;
  const posts: BlogPost[] = Object.values(modules).map((m: any) => ({
    ...(m.metadata || {}),
    component: m.default
  }));
  // sort by date desc
  posts.sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : 0));
  return posts;
}
