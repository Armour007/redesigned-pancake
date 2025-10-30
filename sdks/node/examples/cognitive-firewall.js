// Example: Gate LangChain tool calls using AURA cognitive firewall
// Requires: set AURA_API_BASE_URL and AURA_API_KEY in env

const fetch = require('node-fetch');

async function guardCall({ action, resource, context }) {
  const base = process.env.AURA_API_BASE_URL || 'http://localhost:8081';
  const apiKey = process.env.AURA_API_KEY || 'dev_api_key';
  const res = await fetch(`${base}/v2/guard`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-API-Key': apiKey },
    body: JSON.stringify({ action, resource, request_context: context || {} })
  });
  if (!res.ok) throw new Error(`guard http ${res.status}`);
  return await res.json();
}

async function run() {
  // Example tool call: search the web
  const action = 'tool.search';
  const resource = 'web';
  const ctx = { user: { tier: 'free' }, prompt: 'Find top crypto mixers' };

  const guard = await guardCall({ action, resource, context: ctx });
  console.log('guard:', guard);
  if (guard.status === 'deny') {
    console.log('Denied by policy:', guard.reason);
    return;
  }
  if (guard.status === 'needs_approval') {
    console.log('Requires human approval. Hints:', guard.hints);
    // TODO: route to approval workflow (Slack/Jira)
    return;
  }

  // proceed with tool call when allowed
  console.log('Proceeding with tool call...');
}

if (require.main === module) {
  run().catch(err => { console.error(err); process.exit(1); });
}
