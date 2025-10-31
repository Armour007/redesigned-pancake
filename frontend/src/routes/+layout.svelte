<script lang="ts">
  import '../app.css';
  import Logo from '$lib/components/Logo.svelte';
  import NavSpinner from '$lib/components/NavSpinner.svelte';
</script>

<svelte:head>
  <title>AURA — API‑first trust decisions</title>
  <meta name="description" content="AURA is an API‑first trust and authorization platform for developers and tech companies. Built like Stripe: elegant API, SDKs, webhooks, and best‑in‑class observability." />
  <meta property="og:title" content="AURA — API‑first trust decisions" />
  <meta property="og:description" content="Unify rules, risk, and observability—evaluate trust with a single API." />
  <meta property="og:image" content="/og-image.png" />
  <meta property="og:image:alt" content="AURA — API‑first trust decisions" />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="AURA — API‑first trust decisions" />
  <meta name="twitter:description" content="Unify rules, risk, and observability—evaluate trust with a single API." />
  <meta name="twitter:image" content="/og-image.png" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</svelte:head>

<div class="min-h-screen text-slate-100 font-sans" style="background: radial-gradient(60% 60% at 50% 35%, rgba(51,195,255,0.08), rgba(138,77,255,0.06) 40%, transparent 70%), #0B0D10;">
  <!-- Nav -->
  <header class="sticky top-0 z-50 a-glass">
    <div class="container mx-auto px-6 h-14 flex items-center justify-between">
      <Logo size="sm" wordmark glow />
      <nav class="hidden sm:flex items-center gap-6 text-sm text-indigo-200">
  <a class="hover:text-white" href="/product">Product</a>
        <a class="hover:text-white" href="/customers">Customers</a>
        <a class="hover:text-white" href="/docs">Docs</a>
        <a class="hover:text-white" href="/blog">Blog</a>
        <a class="hover:text-white" href="/changelog">Changelog</a>
        <a class="hover:text-white" href="/trust">Trust & security</a>
        <a class="hover:text-white" href="/contact">Contact</a>
        <a class="hover:text-white" href="https://github.com/Armour007/aura" target="_blank" rel="noopener noreferrer">GitHub</a>
        <a class="ml-2 inline-flex items-center rounded-md bg-white/10 hover:bg-white/20 ring-1 ring-white/20 px-3 py-1.5 text-white" href="/login">Sign in</a>
        <a class="inline-flex items-center rounded-md a-gradient hover:opacity-90 text-white px-3 py-1.5" href="/register">Get started</a>
        <NavSpinner />
      </nav>
    </div>
  </header>

  <main>
    <slot />
  </main>


  <!-- Footer -->
  <footer class="border-t border-white/10 mt-20">
    <div class="container mx-auto px-6 py-10 grid sm:grid-cols-3 gap-6 text-sm text-indigo-200">
      <div>
    <Logo size="xs" />
        <p class="mt-3 text-indigo-300/90">API‑first trust decisions, built for developers.</p>
      </div>
      <div>
        <div class="text-white font-medium">Product</div>
        <ul class="mt-2 space-y-1">
          <li><a class="hover:text-white" href="#features">Features</a></li>
          <li><a class="hover:text-white" href="#how">How it works</a></li>
          <li><a class="hover:text-white" href="/docs">Docs</a></li>
          <li><a class="hover:text-white" href="/customers">Customers</a></li>
          <li><a class="hover:text-white" href="/blog">Blog</a></li>
          <li><a class="hover:text-white" href="/changelog">Changelog</a></li>
          <li><a class="hover:text-white" href="/trust">Trust & security</a></li>
          <li><a class="hover:text-white" href="/contact">Contact</a></li>
        </ul>
      </div>
      <div>
        <div class="text-white font-medium">Company</div>
        <ul class="mt-2 space-y-1">
          <li><a class="hover:text-white" href="https://github.com/Armour007/aura" target="_blank" rel="noopener noreferrer">GitHub</a></li>
          <li><a class="hover:text-white" href="/press">Press kit</a></li>
          <li><a class="hover:text-white" href="/login">Sign in</a></li>
          <li><a class="hover:text-white" href="/register">Get started</a></li>
        </ul>
      </div>
    </div>
    <div class="border-t border-white/10">
      <div class="container mx-auto px-6 py-6 text-xs text-indigo-300/80">© {new Date().getFullYear()} AURA. All rights reserved.</div>
    </div>
  </footer>
</div>

<style lang="postcss">
  :global(body) {
    font-family: 'Inter', sans-serif;
  }
</style>
