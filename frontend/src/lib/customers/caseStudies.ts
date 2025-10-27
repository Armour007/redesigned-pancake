export type CaseStudy = {
  slug: string;
  name: string;
  logo: string; // path under /static
  headline: string;
  summary: string;
  metrics: { label: string; value: string }[];
  bodyHtml: string;
};

export const caseStudies: CaseStudy[] = [
  {
    slug: 'acme-checkout',
    name: 'Acme Checkout',
    logo: '/logos/logo1.svg',
    headline: '50% fewer false declines with rules that adapt in minutes',
    summary: 'Acme unified risk signals and rules in AURA and tuned them without redeploys.',
    metrics: [
      { label: 'False declines', value: '−50%' },
      { label: 'Time to change', value: 'minutes' },
      { label: 'Latency', value: '120ms P50' }
    ],
    bodyHtml: `<p>Acme moved from scattered checks to a single <code>verify()</code> call, streaming decisions to their risk data lake. With Grafana/Tempo, they traced outliers and tuned rules safely.</p>`
  },
  {
    slug: 'globex-platform',
    name: 'Globex Platform',
    logo: '/logos/logo2.svg',
    headline: 'Enterprise governance without slowing delivery',
    summary: 'RBAC and org controls gave Globex policy guardrails across teams.',
    metrics: [
      { label: 'Teams onboarded', value: '12' },
      { label: 'Incidents', value: '0 Sev‑1' },
      { label: 'Uptime', value: '99.9%' }
    ],
    bodyHtml: `<p>Globex standardized decisioning across services, replacing ad‑hoc patterns with auditability and idempotent APIs. Observability made tuning decisions measurable.</p>`
  }
];
