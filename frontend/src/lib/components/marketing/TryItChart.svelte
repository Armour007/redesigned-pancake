<script lang="ts">
  import { onMount } from 'svelte';
  export let history: Array<{ t: number; allow: boolean; latency: number }> = [];

  let el: HTMLDivElement;
  let chart: any = null;
  let uPlotMod: any = null;
  let ready = false;
  let tooltip = { visible: false, x: 0, y: 0, text: '' };

  // Helper to build series arrays from history
  function buildData() {
    const xs: number[] = history.map(h => h.t);
    const ys: number[] = history.map(h => h.latency);
    return [xs, ys];
  }

  onMount(async () => {
    try {
      const mod = await import('uplot');
      uPlotMod = mod.default ?? mod; // ESM/CJS interop
      await import('uplot/dist/uPlot.min.css');

      const data = buildData();
      const width = el?.clientWidth || 300;

      chart = new uPlotMod({
        width,
        height: 120,
        cursor: { focus: { prox: 16 } },
        scales: {
          x: { time: false },
          y: { auto: true }
        },
        axes: [
          { show: false },
          { show: false }
        ],
        series: [
          {},
          {
            label: 'Latency',
            stroke: '#818cf8',
            width: 2
          }
        ],
        hooks: {
          setCursor: [u => {
            const idx = u.cursor.idx;
            if (idx != null && idx >= 0 && idx < history.length) {
              const h = history[idx];
              tooltip = {
                visible: true,
                x: u.cursor.left,
                y: u.cursor.top,
                text: `${new Date(h.t).toLocaleTimeString()} · ${h.allow ? 'allow' : 'deny'} · ${h.latency} ms`
              };
            }
          }]
        }
      }, data, el);

      // Handle resize
      const ro = new ResizeObserver(() => {
        try {
          const w = el?.clientWidth || 300;
          chart.setSize({ width: w, height: 120 });
        } catch {}
      });
      ro.observe(el);

      ready = true;
    } catch (e) {
      // If uPlot fails to load, we will silently keep fallback in parent.
      console.error('[TryItChart] Failed to load uPlot', e);
    }
  });

  // Update on history change
  $: if (chart && ready) {
    try {
      chart.setData(buildData());
    } catch {}
  }
</script>

<div class="relative" bind:this={el} aria-label="Latency over recent decisions (interactive chart)">
  {#if tooltip.visible}
    <div class="pointer-events-none absolute bg-black/80 text-white text-[11px] px-2 py-1 rounded"
         style={`left:${tooltip.x + 8}px; top:${tooltip.y + 8}px`}>
      {tooltip.text}
    </div>
  {/if}
</div>

<style>
  :global(.uplot) {
    display: block;
  }
</style>
