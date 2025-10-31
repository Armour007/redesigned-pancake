<script lang="ts">
  import Logo from '$lib/components/Logo.svelte';
</script>

<section class="bg-slate-950 min-h-screen">
  <div class="container mx-auto px-6 py-16">
    <div class="flex items-center gap-3 aura-glow">
      <Logo href={null} wordmark={true} markPx={28} />
      <span class="text-indigo-300 text-sm">Docs</span>
    </div>
    <h1 class="mt-2 text-3xl sm:text-4xl font-semibold text-white">AURA Documentation</h1>
    <p class="mt-2 text-indigo-200 max-w-2xl">Explore guides, API reference, and integration tips for building with AURA. Everything you need to get started and go deep.</p>

    <div class="mt-10 grid md:grid-cols-2 gap-6">
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Quickstart</h2>
        <p class="mt-2 text-indigo-200 text-sm">Get up and running in minutes with our SDKs and sample apps. <br />
        <b>Node:</b> <code>npm install aura-sdk</code><br />
        <b>Python:</b> <code>pip install aura-sdk</code><br />
        <b>Go:</b> <code>go get github.com/Armour007/aura</code></p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">API Reference</h2>
        <p class="mt-2 text-indigo-200 text-sm">Full OpenAPI spec, endpoints, and request/response examples.<br />
        <b>Base URL:</b> <code>https://api.aura.dev/v1</code><br />
        <b>Auth:</b> Bearer JWT<br />
        <b>Endpoints:</b> <code>/verify</code>, <code>/logs</code>, <code>/apikeys</code>, <code>/agents</code></p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Integration Guides</h2>
        <p class="mt-2 text-indigo-200 text-sm">How to connect AURA to your stack:<br />
        <b>Webhooks:</b> Signed, retryable, with DLQ<br />
        <b>SDKs:</b> Node, Python, Go<br />
        <b>Observability:</b> Prometheus, Tempo, Grafana dashboards</p>
      </article>
      <article class="a-card a-ribbon">
        <h2 class="text-white font-medium">Observability</h2>
        <p class="mt-2 text-indigo-200 text-sm">Prometheus, Tempo, and Grafana dashboards for monitoring and tracing.<br />
        <b>Metrics:</b> <code>/metrics</code> endpoint<br />
        <b>Tracing:</b> OpenTelemetry support<br />
        <b>Dashboards:</b> Prebuilt Grafana JSON</p>
      </article>
    </div>

    <div class="mt-10 rounded-xl ring-1 ring-white/10 bg-white/5 p-6">
      <h2 class="text-white font-medium">Swagger & OpenAPI</h2>
      <p class="mt-2 text-indigo-200 text-sm">Browse the full API spec and try endpoints live in Swagger UI.</p>
      <a href="/docs/swagger" class="inline-flex items-center rounded-md a-gradient text-white px-3 py-1.5 text-sm mt-4">Open Swagger UI</a>
    </div>
  </div>
</section>
