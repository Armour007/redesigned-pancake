<script lang="ts">
  export let variant: 'info' | 'success' | 'warning' | 'error' = 'info';
  export let title: string | null = null;
  // Accessible defaults: polite announcements; role defaults to 'status'.
  export let role: 'status' | 'alert' | string | null = null;
  export let live: 'polite' | 'assertive' | null = null;
  export let ariaAtomic: boolean = true;
  // Allow passing extra classes
  export let className: string = '';

  // Map variant to styles and icon
  const styles: Record<string, { box: string; icon: string; iconName: string }> = {
    info: {
      box: 'bg-blue-900/30 border-blue-700 text-blue-200',
      icon: 'text-blue-300',
      iconName: 'info'
    },
    success: {
      box: 'bg-[#0f1a0f] border-[#2f5f2f] text-green-200',
      icon: 'text-green-300',
      iconName: 'check_circle'
    },
    warning: {
      box: 'bg-amber-900/30 border-amber-700 text-amber-200',
      icon: 'text-amber-300',
      iconName: 'warning'
    },
    error: {
      box: 'bg-red-900/30 border-red-700 text-red-200',
      icon: 'text-red-300',
      iconName: 'error'
    }
  };

  $: s = styles[variant] || styles.info;
  $: isError = variant === 'error';
  $: computedRole = role ?? (isError ? 'alert' : 'status');
  $: computedLive = live ?? (isError ? 'assertive' : 'polite');
</script>

<div
  role={computedRole}
  aria-live={computedLive}
  aria-atomic={ariaAtomic}
  class={`flex items-start gap-3 p-3 rounded border ${s.box} ${className}`}
>
  <span class={`material-symbols-outlined text-xl flex-shrink-0 ${s.icon}`} aria-hidden="true">{s.iconName}</span>
  <div class="min-w-0">
    {#if title}
      <div class="font-semibold">{title}</div>
    {/if}
    <div class="text-sm"><slot /></div>
  </div>
</div>

<style>
  .material-symbols-outlined {
    font-variation-settings:
      'FILL' 0,
      'wght' 400,
      'GRAD' 0,
      'opsz' 24
  }
</style>
