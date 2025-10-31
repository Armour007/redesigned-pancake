<script lang="ts">
  export let size: 'xs' | 'sm' | 'md' | 'lg' = 'sm';
  export let wordmark: boolean = true;
  export let glow: boolean = false;
  export let className: string = '';
  // When set, overrides the size of the mark (in px)
  export let markPx: number | null = null;
  // If null, renders a non-link wrapper (div). Otherwise, renders anchor with href.
  export let href: string | null = '/';

  const sizeMap = {
    xs: { img: 'h-4 w-4', text: 'text-sm' },
    sm: { img: 'h-5 w-5', text: 'text-base' },
    md: { img: 'h-7 w-7', text: 'text-lg' },
    lg: { img: 'h-9 w-9', text: 'text-xl' }
  } as const;

  $: dims = sizeMap[size] ?? sizeMap.sm;
  $: styleAttr = markPx ? `width:${markPx}px;height:${markPx}px` : '';
</script>

{#if href}
  <a class={`inline-flex items-center gap-2 ${glow ? 'aura-glow' : ''} ${className}`} href={href} aria-label="AURA Home">
    <img src="/logos/aura.svg" alt="" class={`${markPx ? '' : dims.img}`} style={styleAttr} aria-hidden="true" />
    {#if wordmark}
      <span class={`font-semibold tracking-tight`}>AURA</span>
    {/if}
  </a>
{:else}
  <div class={`inline-flex items-center gap-2 ${glow ? 'aura-glow' : ''} ${className}`} aria-label="AURA">
    <img src="/logos/aura.svg" alt="" class={`${markPx ? '' : dims.img}`} style={styleAttr} aria-hidden="true" />
    {#if wordmark}
      <span class={`font-semibold tracking-tight`}>AURA</span>
    {/if}
  </div>
{/if}
