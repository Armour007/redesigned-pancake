<script lang="ts">
  import Logo from '$lib/components/Logo.svelte';
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <div class="flex items-center gap-3 aura-glow">
      <Logo href={null} wordmark={true} markPx={28} />
      <span class="text-indigo-300 text-sm">Product</span>
    </div>
    <h1 class="mt-2 text-3xl sm:text-4xl font-semibold text-white">Everything you need for trust decisions</h1>
    <p class="mt-2 text-indigo-200 max-w-2xl">AURA brings authorization, risk, and observability together so your team ships faster and safer. SDKs, webhooks, and dashboards included.</p>

    <div class="mt-10 grid md:grid-cols-2 gap-6">
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Rules & Policies</h2>
        <p class="mt-2 text-indigo-200 text-sm">Author decisions with structured policies—reference context like user, device, and risk signals. Test and roll out safely.</p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Signals & Risk</h2>
        <p class="mt-2 text-indigo-200 text-sm">Ingest device and behavioral signals. Use built‑in velocity, anomaly, and allow‑list checks.</p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Webhooks & DLQ</h2>
        <p class="mt-2 text-indigo-200 text-sm">Reliable delivery with signed webhooks and dead‑letter queue tools to reprocess safely.</p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Observability</h2>
        <p class="mt-2 text-indigo-200 text-sm">OpenAPI, request IDs, and first‑class Grafana/Tempo/Prometheus support. Trace a decision end‑to‑end.</p>
      </article>
    </div>

    <div class="mt-10 rounded-xl ring-1 ring-white/10 bg-white/5 p-6">
      <h2 class="text-white font-medium">SDKs</h2>
      <p class="mt-2 text-indigo-200 text-sm">Node, Python, and Go SDKs with typed responses. Try it via our quickstart templates.</p>
      <div class="mt-4 flex items-center gap-3">
        <a href="/docs" class="inline-flex items-center rounded-md a-gradient text-white px-3 py-1.5 text-sm">Read the docs</a>
        <a href="/register" class="inline-flex items-center rounded-md bg-white/10 ring-1 ring-white/20 hover:bg-white/20 text-white px-3 py-1.5 text-sm">Get started</a>
      </div>
    </div>
  </div>
</section>
