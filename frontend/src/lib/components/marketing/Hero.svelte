<script lang="ts">
  import { inview } from '$lib/components/inview';
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
        <a href="/register" class="inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-5 py-3 text-sm font-medium shadow-lg shadow-indigo-500/25 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-indigo-400">
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

    <!-- Showcase card -->
    <div use:inview class="mt-16 opacity-0 transition duration-700 ease-out translate-y-6 [&.inview]:opacity-100 [&.inview]:translate-y-0">
      <div class="mx-auto max-w-6xl rounded-2xl border border-white/10 bg-white/5 backdrop-blur p-4 shadow-2xl shadow-indigo-900/30">
        <div class="aspect-[16/9] w-full rounded-xl bg-gradient-to-br from-slate-900 to-slate-800 p-6">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6 h-full">
            <div class="rounded-lg bg-black/60 ring-1 ring-white/10 p-4 overflow-auto">
              <div class="text-indigo-300 text-xs font-mono">POST /v1/verify</div>
              <pre class="mt-3 text-xs leading-relaxed text-indigo-100 overflow-x-auto"><code>{req}</code></pre>
            </div>
            <div class="rounded-lg bg-black/60 ring-1 ring-white/10 p-4">
              <div class="text-emerald-300 text-xs font-mono">Response</div>
              <pre class="mt-3 text-xs leading-relaxed text-emerald-100 overflow-x-auto"><code>{res}</code></pre>
              <div class="mt-4 text-[10px] text-indigo-200/80">Fully instrumented with Prometheus and OpenTelemetry</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<style>
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
