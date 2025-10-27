<script lang="ts">
  type Entry = { date: string; title: string; items: string[] };
  const entries: Entry[] = [
    { date: '2025-10-27', title: 'Launch of marketing site', items: [
      'Stripe‑inspired landing page with animated hero and SDK tabs',
      'Pricing and comparison sections',
      'Try it mock verify demo'
    ]},
    { date: '2025-10-26', title: 'Core platform hardening', items: [
      'OpenAPI + Swagger UI, Prometheus, optional OTEL tracing',
      'Rate limiting, idempotency, request IDs, audit logs'
    ]}
  ];
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <h1 class="text-3xl sm:text-4xl font-semibold text-white">Changelog</h1>
    <p class="mt-2 text-indigo-200">What’s new in AURA</p>

    <div class="mt-8 space-y-8">
      {#each entries as e}
        <article class="rounded-xl ring-1 ring-white/10 bg-white/5 p-6">
          <div class="text-sm text-indigo-300">{e.date}</div>
          <h2 class="mt-1 text-white font-medium">{e.title}</h2>
          <ul class="mt-3 list-disc list-inside text-indigo-200 text-sm">
            {#each e.items as item}
              <li>{item}</li>
            {/each}
          </ul>
        </article>
      {/each}
    </div>
  </div>
</section>
