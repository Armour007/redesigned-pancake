<script lang="ts">
  import { page } from '$app/stores'; // To get the current path for active link styling
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import '../../app.css';// Import Tailwind
  import { API_BASE } from '$lib/api';
  import ToastHost from '$lib/components/ToastHost.svelte';

  // Simple check on mount if the token exists.
  // A more robust solution would use hooks or a dedicated auth store.
  let user: { full_name?: string; email?: string; avatarUrl?: string } = {};
  let menuOpen = false;
  let dlqTotal: number | null = null; // codegen DLQ
  let webhookDlqTotal: number | null = null;

  async function loadMe() {
    const token = localStorage.getItem('aura_token');
    if (!token) return;
    try {
      const res = await fetch(`${API_BASE}/me`, { headers: { Authorization: `Bearer ${token}` } });
      if (!res.ok) return;
      const me = await res.json();
      user.full_name = me.full_name;
      user.email = me.email;
    } catch {}
  }

  onMount(() => {
    const token = localStorage.getItem('aura_token');
    if (!token) {
      goto('/login'); // Redirect to login if no token
    }
    loadMe();
    // Fetch DLQ total for badge (best-effort)
  refreshDLQTotals();
  const id = setInterval(refreshDLQTotals, 30000);
  const handler = () => refreshDLQTotals();
    try { window.addEventListener('dlq:changed', handler); } catch {}
    return () => { clearInterval(id); try { window.removeEventListener('dlq:changed', handler); } catch {} };
  });

  async function refreshDLQTotals() {
    try {
      const token = localStorage.getItem('aura_token') || '';
      const [r1, r2] = await Promise.all([
        fetch(`${API_BASE.replace(/\/$/, '')}/admin/queue/dlq?count=1`, { headers: { Authorization: `Bearer ${token}` } }),
        fetch(`${API_BASE.replace(/\/$/, '')}/admin/webhooks/dlq?count=1`, { headers: { Authorization: `Bearer ${token}` } })
      ]);
      if (r1.ok) {
        const d1 = await r1.json();
        if (typeof d1.total === 'number') dlqTotal = d1.total;
      }
      if (r2.ok) {
        const d2 = await r2.json();
        if (typeof d2.total === 'number') webhookDlqTotal = d2.total;
      }
    } catch {}
  }

  // Sidebar navigation items
  const navItems = [
    { href: '/dashboard', label: 'Overview', icon: 'dashboard' }, // Material Symbols icon names
    { href: '/agents', label: 'Agents', icon: 'memory' },
    { href: '/logs', label: 'Logs', icon: 'article' },
    { href: '/settings', label: 'Settings', icon: 'settings' },
    { href: '/apikeys', label: 'API Keys', icon: 'key' },
    { href: '/admin/queue', label: 'Queue (DLQ)', icon: 'error' },
    { href: '/admin/webhooks/dlq', label: 'Webhooks DLQ', icon: 'notifications_active' },
  ];

  function handleLogout() {
      localStorage.removeItem('aura_token');
      goto('/login');
  }

</script>

<div class="flex min-h-screen bg-[#111111] text-white">
  <aside class="w-64 bg-[#111111] border-r border-white/10 flex flex-col fixed inset-y-0 left-0">
    <div class="p-6 flex items-center gap-3 h-16 border-b border-white/10">
      <div class="w-8 h-8 bg-[#7C3AED] rounded-full"></div>
      <h1 class="text-xl font-bold text-white">AURA</h1>
    </div>

    <nav class="flex-1 px-4 py-4 space-y-2">
      {#each navItems as item}
        <a
          href={item.href}
          class={`flex items-center gap-3 px-4 py-2.5 rounded-lg transition-colors duration-150 ${
            $page.url.pathname === item.href
              ? 'bg-[#7C3AED] text-white'
              : 'text-gray-400 hover:bg-white/10 hover:text-gray-200'
          }`}
        >
          <span class="material-symbols-outlined text-xl">{item.icon}</span>
          <span class="text-sm font-medium flex items-center gap-2">
            {item.label}
            {#if item.href === '/admin/queue' && dlqTotal && dlqTotal > 0}
              <span class="inline-flex items-center justify-center text-xs bg-red-600 text-white px-2 py-0.5 rounded-full">{dlqTotal}</span>
            {:else if item.href === '/admin/webhooks/dlq' && webhookDlqTotal && webhookDlqTotal > 0}
              <span class="inline-flex items-center justify-center text-xs bg-red-600 text-white px-2 py-0.5 rounded-full">{webhookDlqTotal}</span>
            {/if}
          </span>
        </a>
      {/each}
    </nav>

    <div class="p-4 mt-auto border-t border-white/10">
      <a href="/docs" class="flex items-center gap-3 px-4 py-2.5 rounded-lg text-gray-400 hover:bg-white/10 hover:text-gray-200 transition-colors duration-150">
        <span class="material-symbols-outlined text-xl">help_outline</span>
        <span class="text-sm font-medium">Help & Docs</span>
      </a>
       <button on:click={handleLogout} class="w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-gray-400 hover:bg-white/10 hover:text-gray-200 transition-colors duration-150">
        <span class="material-symbols-outlined text-xl">logout</span>
        <span class="text-sm font-medium">Logout</span>
      </button>
    </div>
  </aside>

  <div class="flex-1 flex flex-col ml-64"> <!-- Offset by sidebar width -->
    <header class="p-6 flex items-center justify-between border-b border-white/10">
      <div class="flex items-center gap-4">
        <button class="p-2 rounded-full hover:bg-white/10 text-gray-400 hover:text-gray-200" aria-label="Notifications">
          <span class="material-symbols-outlined text-xl" aria-hidden="true">notifications</span>
        </button>
      </div>

      <div class="relative">
        <button class="flex items-center gap-3 px-3 py-2 rounded hover:bg-white/10" on:click={() => (menuOpen = !menuOpen)}>
          <div class="w-8 h-8 rounded-full bg-gray-600 flex items-center justify-center text-sm font-medium">
            {(user.full_name || user.email || '?').toString().charAt(0).toUpperCase()}
          </div>
          <div class="text-left hidden sm:block">
            <div class="text-sm font-medium">{user.full_name || 'User'}</div>
            <div class="text-xs text-gray-400">{user.email || ''}</div>
          </div>
          <span class="material-symbols-outlined text-xl">expand_more</span>
        </button>
        {#if menuOpen}
          <div class="absolute right-0 mt-2 w-48 bg-[#151515] border border-white/10 rounded-lg shadow-xl z-20">
            <a href="/settings" class="block px-4 py-2 text-sm text-gray-200 hover:bg-white/10">Profile & Settings</a>
            <a href="/apikeys" class="block px-4 py-2 text-sm text-gray-200 hover:bg-white/10">API Keys</a>
            <button on:click={handleLogout} class="w-full text-left px-4 py-2 text-sm text-red-300 hover:bg-white/10">Logout</button>
          </div>
        {/if}
      </div>
    </header>

    <main class="flex-1 p-8">
      <slot /> <!-- Page content goes here -->
    </main>
    <ToastHost />
  </div>
</div>

<style>
  /* Ensure Material Symbols are loaded */
  .material-symbols-outlined {
    font-variation-settings:
    'FILL' 0,
    'wght' 400,
    'GRAD' 0,
    'opsz' 24
  }
</style>