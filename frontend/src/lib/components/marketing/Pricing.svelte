<script lang="ts">
  type Plan = { name: string; priceMonthly: string; priceAnnual: string; description: string; cta: string; features: string[] };
  let annual = true;
  const plans: Plan[] = [
    {
      name: 'Developer',
      priceMonthly: '$0',
      priceAnnual: '$0',
      description: 'Start building and test in dev environments.',
      cta: 'Start free',
      features: ['1 agent', '10k verify/mo', 'Community support']
    },
    {
      name: 'Team',
      priceMonthly: '$79',
      priceAnnual: '$59',
      description: 'Production‑ready with observability and webhooks.',
      cta: 'Start trial',
      features: ['5 agents', '1M verify/mo', 'Webhooks & SDKs', 'Email support']
    },
    {
      name: 'Enterprise',
      priceMonthly: 'Contact us',
      priceAnnual: 'Contact us',
      description: 'Advanced governance, SSO, custom SLAs.',
      cta: 'Contact sales',
      features: ['Unlimited agents', 'Custom limits', 'SSO/SAML', 'Dedicated support']
    }
  ];
</script>

<section class="bg-slate-950" aria-labelledby="pricing-heading">
  <div class="container mx-auto px-6 py-20">
    <div class="flex items-end justify-between gap-6 flex-wrap">
      <div class="max-w-2xl">
        <h2 id="pricing-heading" class="text-3xl sm:text-4xl font-semibold text-white">Pricing for developers</h2>
        <p class="mt-3 text-indigo-200">Simple tiers, generous free plan. Switch billing cadence below.</p>
      </div>
      <div class="inline-flex rounded-lg ring-1 ring-white/10 bg-white/5 p-1">
        <button class="px-3 py-1.5 text-sm rounded-md transition {annual ? 'text-indigo-200 hover:bg-white/10' : 'bg-indigo-500 text-white'}" on:click={() => annual = false} aria-pressed={!annual}>Monthly</button>
        <button class="px-3 py-1.5 text-sm rounded-md transition {annual ? 'bg-indigo-500 text-white' : 'text-indigo-200 hover:bg-white/10'}" on:click={() => annual = true} aria-pressed={annual}>Annual</button>
      </div>
    </div>

    <div class="mt-8 grid grid-cols-1 md:grid-cols-3 gap-6">
      {#each plans as p}
        <article class="rounded-2xl ring-1 ring-white/10 bg-white/5 p-6">
          <div class="text-indigo-300 text-sm">{p.name}</div>
          <div class="mt-2 text-white text-3xl font-semibold">{annual ? p.priceAnnual : p.priceMonthly}</div>
          <p class="mt-2 text-indigo-200 text-sm">{p.description}</p>
          <ul class="mt-4 space-y-1 text-sm text-indigo-200">
            {#each p.features as f}
              <li>• {f}</li>
            {/each}
          </ul>
          <a href={p.name === 'Enterprise' ? '/contact' : '/register'} class="mt-5 inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-4 py-2 text-sm">{p.cta}</a>
        </article>
      {/each}
    </div>
  </div>
</section>
