# Try It visualization: tooltips and richer charts

This doc proposes lightweight options to add tooltips and richer visuals to `TryIt.svelte` while keeping the bundle small and fast.

## Current state
- Custom SVG sparkline with allow/deny markers
- Computed allow rate and average latency stats
- No tooltips or hover interactions

## Options

1) uPlot (recommended)
- Size: ~45 KB minified, ~10–15 KB gzip (core)
- Pros: Very fast, small, great for real-time and sparklines
- Cons: Lower-level config; tooltips need a small plugin or custom hover
- Svelte wrapper: `svelte-uplot` or integrate directly

2) ApexCharts
- Size: ~300 KB minified (core) before gzip
- Pros: Built-in tooltips, legends, annotations; great DX
- Cons: Heavier bundle
- Svelte wrapper: `svelte-apexcharts`

3) Chart.js
- Size: ~220 KB minified (core) before gzip; treeshake partials
- Pros: Mature, familiar; plugins available
- Cons: Still heavier than uPlot
- Svelte wrapper: `svelte-chartjs`

## Recommended implementation (uPlot)

1) Install
- devDependencies: `uplot`
- Optional wrapper: `svelte-uplot`

2) Data model
- Keep two aligned series: timestamp (x) and latency (y)
- Derive a second series for allow/deny (0/1) overlay markers or color-coded points

3) Component sketch
- Lazy-load the chart on mount (`onMount`) to avoid SSR mismatches
- Provide a minimal tooltip: floating div showing date, decision, latency
- Keyboard focus: ensure focus ring on chart canvas with aria-label and live region for updates

4) Accessibility
- Provide a textual summary below the chart (we already compute stats)
- Optionally expose a table view for screen readers

5) Performance
- Cap history length (we already do), and throttle updates

## Pseudocode

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  let el: HTMLDivElement;
  let chart: any;
  let tooltip = { visible: false, x: 0, y: 0, text: '' };

  onMount(async () => {
    const uPlot = (await import('uplot')).default;
    const data = [timestamps, latencies];
    chart = new uPlot({
      width: el.clientWidth,
      height: 140,
      series: [
        {},
        { stroke: 'url(#grad)', width: 2 }
      ],
      hooks: {
        setCursor: [u => {
          const idx = u.cursor.idx;
          if (idx != null) {
            tooltip = {
              visible: true,
              x: u.cursor.left,
              y: u.cursor.top,
              text: `${new Date(timestamps[idx]).toLocaleTimeString()} · ${latencies[idx]} ms`
            };
          }
        }]
      }
    }, data, el);
  });
</script>

<div bind:this={el} class="relative">
  {#if tooltip.visible}
    <div class="absolute bg-black/80 text-white text-xs px-2 py-1 rounded"
         style={`left:${tooltip.x + 8}px; top:${tooltip.y + 8}px`}>
      {tooltip.text}
    </div>
  {/if}
</div>
```

## Rollout
- Step 1: Add uPlot as a dep and a lazy-loaded chart path behind a feature flag `TRYIT_CHARTS`
- Step 2: Validate bundle size; fall back to SVG if disabled
- Step 3: Iterate on tooltip content and color mapping for allow/deny
