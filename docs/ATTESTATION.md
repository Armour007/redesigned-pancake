# Attestation and Adaptive Risk

This prototype adds:

- SPIFFE-based attestation tokens via `/auth/attest`
- Combined auth on verify endpoints (attestation or API key)
- Org setting to disable API keys
- TLS SPIFFE extraction + certificate fingerprint in decision traces
- Runtime risk signals (rate spike) injected into policy evaluation

## Attestation

- Enable endpoint:
  - `AURA_ATTEST_ENABLE=true`
- Token TTL:
  - `AURA_ATTEST_TTL_SECONDS=300` (default 300s)
- Signing key:
  - `AURA_ATTEST_SIGNING_KEY` (falls back to `JWT_SECRET` in dev)

Request:

- POST `/auth/attest`
- Headers: `X-SPIFFE-ID: spiffe://example.org/ns/default/sa/agent-123`
- Body:
  - `{ "org_id": "<org-uuid>", "agent_id": "<agent-uuid>" }`

Response:

- `{ "token": "<jwt>", "expires_at": 173... }`

Token claims include `kind:"attest"`, `sub` (SPIFFE ID), `org_id`, `agent_id`, `aud:"aura"`.

## Verify authentication

- `/v1` and `/v2` verification endpoints now accept either:
  - `Authorization: Bearer <attestation JWT>` (preferred), or
  - `X-API-Key: aura_sk_...` (legacy)
- Org can disable API keys:
  - Column `organizations.api_keys_disabled` (default `false`). When true, API key auth is rejected.
  - Admin toggle endpoint: `PUT /organizations/:orgId/settings {"api_keys_disabled": true}` (requires org admin auth)

## TLS SPIFFE and traces

When TLS client certs are used, the SPIFFE ID is read from URI SAN and the leaf certificate SHA-256 fingerprint is captured. Decision traces include:

```json
{"principal": {"org_id":"...","agent_id":"...","spiffe_id":"...","authn_kind":"spiffe","cert_fingerprint":"..."}}
```

## Adaptive risk signals

A simple sliding-window tracker computes a 0–100 `risk.score` from requests in the last 30s per org+agent. When requests exceed `AURA_RISK_SPIKE_THRESHOLD` (default 50), the `rate_spike` flag is set. The signal is injected into the policy input as:

```json
{"risk": {"score": 82, "flags": ["rate_spike"]}, ...}
```

Debug endpoints:

- `GET /v2/signals/risk?org_id=<>&agent_id=<>`
- Raise an external alert flag for demos: `POST /v2/signals/risk/alerts` with body `{ "flag": "external_alert", "ttl_seconds": 120, "agent_id": "<optional>" }`
- Clear an external alert flag: `DELETE /v2/signals/risk/alerts` with body `{ "flag": "external_alert", "agent_id": "<optional>" }`

## Decisions and policy insights

- Recent decisions (for dashboards): `GET /v2/decisions/search?limit=50&agent_id=&allow=`
- Policy versions (for graphs): `GET /v2/policies/:policyId/versions`
- Prototype recommendations from recent decisions: `GET /v2/policy/recommendations?org_id=`

## Spike simulation script

Use the provided PowerShell script to trigger a temporary rate spike and observe policy behavior:

```
scripts/simulate-risk-spike.ps1 -OrgId <org-uuid> -AgentId <agent-uuid> -Count 40 -Concurrency 5 -BaseUrl http://localhost:8081
```

Tip: set `AURA_RISK_SPIKE_THRESHOLD` low (e.g., 20) and use a policy that denies when `risk.score >= 80` to see DENY within ~30s.

## Example policy snippet (AuraJSON)

Deny when risk score is high:

```json
{
  "any": [
    {"gte": [{"var": "risk.score"}, 80], "effect": "deny", "reason": "high risk"},
    {"effect": "allow"}
  ]
}
```

## Acceptance test idea

1. Set `AURA_RISK_SPIKE_THRESHOLD=20`.
2. Apply a policy denying when `risk.score >= 80`.
3. Send > 20 verify calls in < 30s for an agent; verify subsequent calls are DENIED within 30s.

