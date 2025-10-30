# Cognitive Firewall

Inline policy guard for agent actions with mitigation hints, AI-assisted authoring, and preview against historical traces.

## What it is
- Inline guard endpoint to evaluate policies at tool/function-call time.
- Mitigation hints: policies can return `needs_approval` with guidance.
- NL → Policy: compile natural language intents into AuraJSON/Rego templates.
- Test runner: feed inputs and expected outcomes for quick validation.
- Preview: evaluate a candidate policy against recent decision traces to estimate impact.

## Endpoints
- POST `/v2/guard` — Evaluate inline or assigned policy
  - Request: `{ agent_id?, action?, resource?, request_context, policy? { engine: aurajson|rego, body } }`
  - Response: `{ status: allow|deny|needs_approval, reason?, hints?, trace_id? }`
- POST `/v2/policy/author/nl-compile` — NL → Rego/AuraJSON prototype
- POST `/v2/policy/tests/run` — Run table-driven tests
- POST `/v2/policy/preview` — Preview against recent decision traces

## AuraJSON policy DSL (extended)
Rules support three effects: `allow`, `deny`, and `require_approval`.

Example:
```
{
  "precedence": { "deny_overrides": true },
  "rules": [
    { "id": "deny_prod_shell", "effect": "deny", "when": { "env": { "eq": "prod" }, "action": { "eq": "tool.execute" }, "resource": { "eq": "shell" } } },
    { "id": "needs_review_web", "effect": "require_approval", "hint": "Sensitive web access", "when": { "resource": { "eq": "web" } } },
    { "id": "allow_default", "effect": "allow" }
  ]
}
```

## Integrations
- LangChain / tool calling: gate calls by POSTing to `/v2/guard` before executing the tool.
  - Node example: `sdks/node/examples/cognitive-firewall.js`
  - Python example: `sdks/python/examples/cognitive_firewall.py`

## Previewing changes
- Use `/v2/policy/preview` to compare a new policy against last N decision traces for your org.
- You’ll get a summary of `allow/deny/needs_approval` counts and a few sample diffs.

## UI (prototype)
- A lightweight builder page can call NL compile, run tests, and preview endpoints. Wire it to your dashboard with API key auth.

## Notes
- NL compiler is heuristic and generates a starter template. Review and refine before activating.
- For Rego usage, enable OPA via `AURA_POLICY_ENABLE_OPA=1`.
