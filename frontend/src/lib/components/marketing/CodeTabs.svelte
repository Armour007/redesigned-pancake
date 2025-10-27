<script lang="ts">
  type Tab = 'node' | 'python' | 'go';
  let tab: Tab = 'node';
  const code: Record<Tab, string> = {
    node: `// npm i @aura/sdk\nimport Aura from '@aura/sdk'\nconst aura = new Aura({ apiKey: process.env.AURA_API_KEY })\nconst decision = await aura.verify({\n  agent_id: 'aura_agent_123',\n  request_context: { user_id: 'user_42', action: 'checkout.create' }\n})\nconsole.log(decision.decision)`,
    python: `# pip install aura-sdk\nfrom aura_sdk import Aura\naura = Aura(api_key=os.environ['AURA_API_KEY'])\nres = aura.verify(agent_id='aura_agent_123', request_context={ 'user_id': 'user_42', 'action': 'checkout.create' })\nprint(res['decision'])`,
    go: `// go get github.com/armour007/aura/sdks/go/aura\ncli := aura.NewClient(os.Getenv("AURA_API_KEY"))\nres, _ := cli.Verify(aura.VerifyRequest{\n  AgentID: "aura_agent_123",\n  RequestContext: map[string]any{"user_id":"user_42","action":"checkout.create"},\n})\nfmt.Println(res.Decision)`
  };
</script>

<section class="bg-slate-950">
  <div class="container mx-auto px-6 py-20">
    <div class="max-w-2xl">
      <h2 class="text-3xl sm:text-4xl font-semibold text-white">SDKs that feel native</h2>
      <p class="mt-3 text-indigo-200">Start with a copy‑paste and ship your first decision in minutes.</p>
    </div>

    <div class="mt-6">
      <div role="tablist" aria-label="SDK examples" class="inline-flex rounded-lg ring-1 ring-white/10 bg-white/5 p-1">
        <button role="tab" aria-selected={tab==='node'} class="px-3 py-1.5 text-sm rounded-md transition {tab==='node' ? 'bg-indigo-500 text-white' : 'text-indigo-200 hover:bg-white/10'}" on:click={() => tab='node'}>Node</button>
        <button role="tab" aria-selected={tab==='python'} class="px-3 py-1.5 text-sm rounded-md transition {tab==='python' ? 'bg-indigo-500 text-white' : 'text-indigo-200 hover:bg-white/10'}" on:click={() => tab='python'}>Python</button>
        <button role="tab" aria-selected={tab==='go'} class="px-3 py-1.5 text-sm rounded-md transition {tab==='go' ? 'bg-indigo-500 text-white' : 'text-indigo-200 hover:bg-white/10'}" on:click={() => tab='go'}>Go</button>
      </div>

      <div class="mt-4 rounded-xl bg-black/60 ring-1 ring-white/10 p-4 overflow-auto">
        <pre class="text-xs text-indigo-100 leading-relaxed"><code>{code[tab]}</code></pre>
      </div>
    </div>
  </div>
</section>
