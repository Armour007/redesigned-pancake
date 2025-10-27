import type { RequestHandler } from '@sveltejs/kit';
import { env as pub } from '$env/dynamic/public';

export const GET: RequestHandler = () => {
  const site = (pub.PUBLIC_SITE_URL || 'http://localhost:3000').toString().replace(/\/$/, '');
  const body = `User-agent: *\nDisallow:\n\nSitemap: ${site}/sitemap.xml\n`;
  return new Response(body, { headers: { 'content-type': 'text/plain; charset=utf-8' } });
};
