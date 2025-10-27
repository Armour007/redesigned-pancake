<script lang="ts">
  let name = '';
  let email = '';
  let company = '';
  let message = '';
  let sent = false;
  let error: string | null = null;

  async function submit(e: Event) {
    e.preventDefault();
    error = null; sent = false;
    try {
      const res = await fetch('/api/lead', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, email, company, message }) });
      if (!res.ok) throw new Error('Failed to submit');
      sent = true; name = email = company = message = '';
    } catch (e: any) {
      error = e?.message || 'Unexpected error';
    }
  }
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">Get a demo</h2>
      <p class="mt-3 text-indigo-200">Tell us about your use‑case and we’ll reach out.</p>
    </div>

    <form class="mt-6 max-w-2xl space-y-4" on:submit|preventDefault={submit}>
      {#if sent}
        <div class="rounded-lg bg-emerald-500/10 ring-1 ring-emerald-500/30 text-emerald-100 p-3">Thanks! We’ll be in touch.</div>
      {/if}
      {#if error}
        <div class="rounded-lg bg-rose-500/10 ring-1 ring-rose-500/30 text-rose-100 p-3">{error}</div>
      {/if}
      <div class="grid sm:grid-cols-2 gap-4">
        <label class="block text-sm">
          <span class="text-indigo-200">Name</span>
          <input class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" bind:value={name} required />
        </label>
        <label class="block text-sm">
          <span class="text-indigo-200">Email</span>
          <input type="email" class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" bind:value={email} required />
        </label>
      </div>
      <label class="block text-sm">
        <span class="text-indigo-200">Company</span>
        <input class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" bind:value={company} />
      </label>
      <label class="block text-sm">
        <span class="text-indigo-200">Message</span>
        <textarea class="mt-1 w-full rounded-md bg-black/50 text-white p-2 ring-1 ring-white/10 focus:ring-indigo-500/50" rows="4" bind:value={message} placeholder="What are you building?" />
      </label>
      <button class="inline-flex items-center rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white px-4 py-2 text-sm">Request demo</button>
    </form>
  </div>
</section>
