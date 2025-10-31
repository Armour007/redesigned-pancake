<script lang="ts">
  import { inview } from '$lib/components/inview';
  import Logo from '$lib/components/Logo.svelte';
  import LogoGlow from '$lib/components/LogoGlow.svelte';
  const req = `{
  "agent_id": "aura_agent_123",
  "request_context": {
    "ip": "203.0.113.5",
    "user_id": "user_42",
    "action": "checkout.create",
    "risk_signals": { "velocity": 0.82, "device": "web" }
  }
}`;
  const res = `{
  "decision": "allow",
  "reason": "rule: checkout-default",
  "request_id": "req_9f8c2...",
  "latency_ms": 12
}`;
</script>

<section class="relative overflow-hidden isolate">
  <div class="absolute inset-0 -z-10 bg-gradient-to-b from-indigo-950 via-indigo-900 to-slate-950"></div>
  <div class="absolute -top-24 left-1/2 -translate-x-1/2 h-[40rem] w-[40rem] rounded-full blur-3xl opacity-20 bg-[conic-gradient(at_top_right,_#22d3ee_0%,_#6366f1_30%,_#22c55e_60%,_#a855f7_90%)] animate-[spin_30s_linear_infinite]"></div>

  <div class="container mx-auto px-6 pt-28 pb-20 sm:pt-32 sm:pb-24">
    <div use:inview class="max-w-3xl mx-auto text-center opacity-0 transition duration-700 ease-out translate-y-6 [&.inview]:opacity-100 [&.inview]:translate-y-0">
      <div class="mx-auto inline-flex items-center justify-center">
        <LogoGlow wordmark={true} markPx={140} />
      </div>
      <div class="inline-flex items-center gap-2 rounded-full bg-white/10 ring-1 ring-white/20 px-3 py-1 text-xs text-indigo-200">
        <span class="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" aria-hidden="true"></span>
        Enterprise-ready authZ and trust engine
      </div>
      <h1 class="mt-6 text-4xl font-semibold tracking-tight text-white sm:text-6xl">
        AURA: API‑first trust decisions, built like Stripe
      </h1>
      <p class="mt-6 text-lg leading-8 text-indigo-100">
        Evaluate complex authorization and trust signals with one API. Bring rules, risk, and
        observability together—so your team ships faster and safer.
      </p>
      <div class="mt-8 flex items-center justify-center gap-4">
        <a href="/register" class="inline-flex items-center rounded-lg a-gradient hover:opacity-90 text-white px-5 py-3 text-sm font-medium a-soft-shadow focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-indigo-400">
          Get started — it’s free
        </a>
        <a href="/docs" class="inline-flex items-center rounded-lg bg-white/10 hover:bg-white/20 text-white px-5 py-3 text-sm font-medium ring-1 ring-white/20 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white/60">
          Read the docs
        </a>
      </div>
      <div class="mt-10 text-xs text-indigo-200/80">
        SDKs for Node, Python, Go • Webhooks • OpenAPI • Grafana/Tempo/Prometheus
      </div>
    </div>

    <!-- Product highlights -->
    <div class="mt-16 grid md:grid-cols-2 gap-6">
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
  </div>
</section>

<style>
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
