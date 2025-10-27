export type Post = {
  slug: string;
  title: string;
  date: string; // ISO date
  summary: string;
  html: string;
};

export const posts: Post[] = [
  {
    slug: 'introducing-aura',
    title: 'Introducing AURA: API-first trust decisions',
    date: '2025-10-27',
    summary: 'Why we built AURA and how it helps teams ship secure experiences faster.',
    html: `<p>AURA unifies rules, risk signals, and observability into a single <code>/v1/verify</code> API so teams can focus on shipping product. With SDKs, webhooks, OpenAPI, and first-class metrics and tracing, it\'s built like Stripe for trust decisions.</p>`
  },
  {
    slug: 'observability-for-trust',
    title: 'Observability for trust decisions: metrics and traces that matter',
    date: '2025-10-20',
    summary: 'How to use Prometheus and Tempo to understand and improve your decision flows.',
    html: `<p>Decisions without visibility are guesswork. We instrument every <code>verify()</code> call with latency histograms and decision counters, and emit spans you can explore in Grafana.</p>`
  },
  {
    slug: 'webhooks-that-scale',
    title: 'Webhooks that scale: signatures, retries, and idempotency',
    date: '2025-10-15',
    summary: 'Best practices for reliable webhook delivery and verification using AURA.',
    html: `<p>We sign events, expect idempotent endpoints, and expose delivery status. Use our SDK helpers to verify signatures and handle replays safely.</p>`
  }
];
