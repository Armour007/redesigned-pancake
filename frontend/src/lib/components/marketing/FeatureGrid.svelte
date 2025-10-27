<script lang="ts">
  const features = [
    {
      title: 'Single Verify API',
      desc: 'Unify rules, context, and risk signals in one call. Versioned and idempotent with rich audit logs.',
      icon: '🔐'
    },
    {
      title: 'Rules, not rewrites',
      desc: 'Declaratively shape trust decisions per agent and environment. Update instantly without deploys.',
      icon: '⚙️'
    },
    {
      title: 'API keys & Webhooks',
      desc: 'Rotate keys, sign webhooks, and consume decision events in your systems reliably.',
      icon: '🔑'
    },
    {
      title: 'Observability native',
      desc: 'Prometheus metrics, OpenTelemetry traces, Grafana dashboards, and Tempo for end‑to‑end visibility.',
      icon: '📈'
    },
    {
      title: 'SDKs you use',
      desc: 'First‑party Node, Python, and Go SDKs with simple, consistent ergonomics.',
      icon: '📦'
    },
    {
      title: 'Enterprise‑ready',
      desc: 'RBAC, orgs/members, rate limiting, CORS, trusted proxies, and OpenAPI.',
      icon: '🏢'
    }
  ];
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">Everything you need to ship trust</h2>
      <p class="mt-3 text-indigo-200">Built like Stripe: a crisp API, delightful docs, and SDKs that just work.</p>
    </div>

    <div class="mt-10 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
      {#each features as f}
        <article class="group rounded-2xl ring-1 ring-white/10 bg-white/5 p-6 transition hover:bg-white/10">
          <div class="text-2xl" aria-hidden="true">{f.icon}</div>
          <h3 class="mt-4 text-white font-medium">{f.title}</h3>
          <p class="mt-2 text-sm text-indigo-200">{f.desc}</p>
          <div class="mt-4 text-xs text-indigo-300 opacity-0 group-hover:opacity-100 transition">Learn more →</div>
        </article>
      {/each}
    </div>
  </div>
</section>
