<script lang="ts">
  import { inview } from '$lib/components/inview';
  type Step = { n: number; title: string; desc: string };
  const steps: Step[] = [
    { n: 1, title: 'Create an Agent', desc: 'Model a surface (e.g., checkout) and connect it to your org.' },
    { n: 2, title: 'Install an SDK', desc: 'npm, pip, or go get—choose your stack and drop in a verify() call.' },
    { n: 3, title: 'Write rules', desc: 'Compose allow/deny logic with context and risk signals. No redeploys needed.' },
    { n: 4, title: 'Ship with confidence', desc: 'Stream decisions to webhooks, monitor with Grafana, and iterate safely.' }
  ];
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">From zero to decision in minutes</h2>
      <p class="mt-3 text-indigo-200">A frictionless path from signup to your first verified request.</p>
    </div>

    <ol class="mt-10 space-y-6">
      {#each steps as s}
        <li use:inview class="relative rounded-xl ring-1 ring-white/10 bg-white/5 p-6 opacity-0 translate-y-3 transition duration-500 [&.inview]:opacity-100 [&.inview]:translate-y-0">
          <div class="flex items-start gap-4">
            <div class="h-8 w-8 shrink-0 rounded-full bg-indigo-500 text-white grid place-items-center font-semibold">{s.n}</div>
            <div>
              <h3 class="text-white font-medium">{s.title}</h3>
              <p class="mt-1 text-sm text-indigo-200">{s.desc}</p>
            </div>
          </div>
        </li>
      {/each}
    </ol>

    <div class="mt-10">
      <a href="/register" class="inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-5 py-3 text-sm font-medium shadow-lg shadow-indigo-500/25 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-indigo-400">Get started</a>
    </div>
  </div>
</section>
