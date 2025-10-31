<script lang="ts">
  import Logo from './Logo.svelte';
  export let size = 72;
  export let wordmark: boolean = false;
  export let markPx: number = size;
</script>

<div class="relative inline-flex items-center justify-center" style={`width:${markPx}px;height:${markPx}px`}>
  <Logo href={null} wordmark={wordmark} glow markPx={markPx} />
  <div class="absolute inset-0 rounded-full" style="filter: blur(18px); background: radial-gradient(60% 60% at 50% 50%, rgba(51,195,255,0.35), rgba(138,77,255,0.28), transparent 65%);"></div>
</div>