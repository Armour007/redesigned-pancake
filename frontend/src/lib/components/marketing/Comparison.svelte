<section class="bg-slate-950" aria-labelledby="comparison-heading">
  <div class="container mx-auto px-6 py-20">
    <h2 id="comparison-heading" class="text-3xl sm:text-4xl font-semibold text-white">Why AURA vs. homegrown rules</h2>
    <div class="mt-6 grid md:grid-cols-2 gap-6">
      <article class="rounded-2xl ring-1 ring-white/10 bg-white/5 p-6">
        <h3 class="text-white font-medium">AURA</h3>
        <ul class="mt-3 list-disc list-inside text-indigo-200 text-sm space-y-1">
          <li>Single verify() API with versioning and idempotency</li>
          <li>Rules updated instantly, no redeploys</li>
          <li>Observability: Prometheus, OTEL, Grafana, Tempo</li>
          <li>SDKs and webhooks for seamless integration</li>
          <li>RBAC, orgs/members, rate limiting built‑in</li>
        </ul>
      </article>
      <article class="rounded-2xl ring-1 ring-white/10 bg-white/5 p-6">
        <h3 class="text-white font-medium">Homegrown</h3>
        <ul class="mt-3 list-disc list-inside text-indigo-200 text-sm space-y-1">
          <li>Ad‑hoc checks scattered across codebases</li>
          <li>Risky redeploys for every logic tweak</li>
          <li>DIY metrics and tracing, limited visibility</li>
          <li>Glue code for keys, webhooks, and retries</li>
          <li>Hard to scale governance and permissions</li>
        </ul>
      </article>
    </div>
  </div>
</section>
