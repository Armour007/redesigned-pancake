# AURA First-Time Guide

AURA universal flow: Rule Creation → API Plug‑In → Real‑time Verification → Audit Logging.

This guide is shown for first‑time users only. You can revisit it anytime at /quickstart.

## 1) Create an Agent and Rule
- Open Swagger at http://localhost:8081/docs
- Register and Login, then click Authorize and paste your Bearer token
- Create an Agent: POST /organizations/{orgId}/agents
- Add an allow Rule to that agent: POST /organizations/{orgId}/agents/{agentId}/permissions
  Body:
  {
    "rule": {
      "action": "deploy",
      "effect": "allow",
      "context": { "env": "prod" }
    }
  }

## 2) Generate an API Key
- POST /organizations/{orgId}/apikeys with {"name":"dev key"}
- Copy the secret_key (you will only see it once). A masked preview appears in the UI.

## 3) Real‑time Verification
Send a verify request with the agent_id and your context.

- Headers:
  - X-API-Key: aura_sk_… (your secret)
  - AURA-Version: 2025-10-01
- Body:
  {
    "agent_id": "YOUR_AGENT_ID",
    "request_context": { "action": "deploy", "env": "prod" }
  }
- Expected: decision = ALLOWED

## 4) Audit Logging
- All verification events are recorded in event_logs.
- Retrieve via GET /organizations/{orgId}/logs or your database (admin access).

## SDK Quick Start
After you create a rule or API key, the UI shows a Quick Start drawer with prefilled snippets in multiple languages (Node, Python, Go, cURL, and more via codegen). You can also open http://localhost:5173/quickstart?agent_id=...&key_prefix=...

## Tips
- Keep secrets secure; rotate via /apikeys.
- Use Webhooks to stream decisions to your systems; verify signatures with HMAC-SHA256.
- Configure rate limits with env vars; set Redis to enable distributed throttling.
