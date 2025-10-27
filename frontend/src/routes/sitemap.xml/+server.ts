import type { RequestHandler } from '@sveltejs/kit';
import { getAllBlogPosts } from '$lib/content/blog';
import { getAllCaseStudies } from '$lib/content/caseStudies';
import { env as pub } from '$env/dynamic/public';

function url(base: string, path: string) {
  return base.replace(/\/$/, '') + path;
}

export const GET: RequestHandler = async () => {
  const site = (pub.PUBLIC_SITE_URL || 'http://localhost:3000').toString().replace(/\/$/, '');

  const staticPaths = [
    '/',
    '/customers',
    '/blog',
    '/changelog',
    '/trust',
    '/press',
    '/contact'
  ];

  const blog = getAllBlogPosts().map((p) => `/blog/${p.slug}`);
  const customers = getAllCaseStudies().map((c) => `/customers/${c.slug}`);

  const entries = [...staticPaths, ...blog, ...customers];

  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` +
    entries
      .map((p) => `<url><loc>${url(site, p)}</loc></url>`)
      .join('') +
    `</urlset>`;

  return new Response(xml, { headers: { 'content-type': 'application/xml' } });
};
