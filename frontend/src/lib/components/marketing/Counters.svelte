<script lang="ts">
  import { inview } from '$lib/components/inview';
  let a = 0, b = 0, c = 0;
  function animate(target: number, setter: (v:number)=>void, duration=1200) {
    const start = performance.now();
    const tick = (t:number) => {
      const p = Math.min(1, (t - start) / duration);
      setter(Math.floor(target * (0.2 + 0.8 * p)));
      if (p < 1) requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  }
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-10">
    <div use:inview={{ once: true }} class="grid sm:grid-cols-3 gap-6 opacity-0 translate-y-2 transition duration-700 [&.inview]:opacity-100 [&.inview]:translate-y-0" on:introstart={() => {}}>
      <div class="rounded-xl ring-1 ring-white/10 bg-white/5 p-6" on:introend={() => animate(120, v => a = v)}>
        <div class="text-3xl font-semibold text-white">{a}ms</div>
        <div class="text-sm text-indigo-200">Median decision latency</div>
      </div>
      <div class="rounded-xl ring-1 ring-white/10 bg-white/5 p-6" on:introend={() => animate(99, v => b = v)}>
        <div class="text-3xl font-semibold text-white">{b}.9%</div>
        <div class="text-sm text-indigo-200">Uptime last 90 days</div>
      </div>
      <div class="rounded-xl ring-1 ring-white/10 bg-white/5 p-6" on:introend={() => animate(500, v => c = v)}>
        <div class="text-3xl font-semibold text-white">{c}M+</div>
        <div class="text-sm text-indigo-200">Verify per month</div>
      </div>
    </div>
  </div>
</section>
