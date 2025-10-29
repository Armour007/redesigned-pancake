# AURA Enterprise Features

This document highlights governance, federation, and compliance features for enterprise deployments.

## Trust tokens

- Endpoint: POST `/v2/tokens/introspect`
  - Body: `{ "token": "<compact JWT>" }`
  - Returns: `{ valid: boolean, reason?: string, claims?: object }`
  - Accepts HS256 tokens issued by `/v2/verify` when `include_trust_token=true`.

## Audit anchoring

- Endpoints:
  - GET `/v2/audit/anchor?date=YYYY-MM-DD` – Get the latest or a specific date's anchor
  - POST `/v2/audit/anchor` – Set or update anchor for a date
    - Body: `{ root_hash: string, date?: YYYY-MM-DD, external_ref?: string }`
- Suggested daily workflow: `.github/workflows/aura-audit-anchor.yml`
  - Fetches latest audit root hash (from `/v2/audit/ledger?limit=1`)
  - Optionally publishes it as a GitHub Gist (external transparency reference)
  - Stores the `external_ref` and `root_hash` via `/v2/audit/anchor`
  - Configure secrets:
    - `AURA_BACKEND_BASE_URL` – e.g. `https://api.example.com`
    - `AURA_API_KEY` – org API key with audit read/write access

## Federation contracts and scope enforcement

- Contracts
  - Create/list under: `/v2/federation/contracts`
  - Contract `scope` is JSON with optional allow-lists:
    - `allowed_actions`: string[] – exact match, `*` wildcard, or prefix patterns ending with `*` (e.g. `repo:*`)
    - `allowed_resources`: string[] – same matching rules as above
- Runtime enforcement (in `/v2/verify`)
  - If `target_org_id` is set and different from caller org:
    - Requires an active contract between the orgs
    - Enforces `allowed_actions` and `allowed_resources` where provided
    - Injects `federation` context to policy evaluation input:
      ```json
      {
        "federation": {
          "counterparty_org_id": "<uuid>",
          "scope": {
            "allowed_actions": ["..."],
            "allowed_resources": ["..."]
          }
        }
      }
      ```

## Notes

- Trust token context hashing uses canonicalized JSON for reproducibility across encoders.
- Audit ledger integrity can be verified via GET `/v2/audit/verify`.

# AURA Enterprise: Governance & Compliance (Stage 5)

This document describes the governance and compliance features introduced in Stage 5 and how to operate them.

## Goals

- Full policy lifecycle with compliance-grade auditability
- SOC2-ready change controls
- Auditable replay of decisions and integrity proofs

## Policy Governance

- Versioning with multi-approver approvals
  - Each policy version accumulates approvals in `policy_version_approvals`.
  - Required approvers: `AURA_POLICY_APPROVALS_REQUIRED` (default `2`).
  - When the threshold is reached, the version status becomes `approved`.
- Change tickets
  - Optional `change_ticket` string can be set when creating a version (AddPolicyVersion payload).
- Rollbacks
  - Activate a prior version via `POST /organizations/:orgId/policies/:policyId/versions/:version/activate`.
- Staged/Canary (prototype)
  - `policy_rollouts` table is present for percent-based rollouts. Selection logic may be enabled in a subsequent iteration.

## Immutable Audit Ledger

- Table: `audit_ledger` implements a per-org hash-chained append-only log.
  - `this_hash = SHA256(prev_hash || canonical_json(payload))`
  - Critical events appended so far:
    - Policy approval threshold reached (or an approval recorded)
    - Policy activation
    - Policy assignment
    - Organization settings update (e.g., API keys disabled)
- API
  - `GET /v2/audit/ledger?limit=200`: list recent entries.
  - `GET /v2/audit/verify?limit=10000`: verifies the hash chain and returns `ok:true` when intact.

## Federated Multi-Tenancy (roadmap)

- Hard org isolation remains enforced by auth and scoping.
- Cross-org trust to be captured as explicit federation contracts with audit entries on boundary crossings (planned).

## Integrations (roadmap)

- CI/CD: GitHub Actions, Jenkins (policy version bumps and approvals)
- Cloud IAMs: import/export controls
- K8s admission controllers: use Verify v2 for admission decisions
- Agent frameworks (LangChain, AutoGen, CrewAI): embed attestation and Verify v2 hooks

## Acceptance Criteria

- Change management controls (multi-approver, change tickets) are enforced and logged.
- Auditors can verify the audit chain via `GET /v2/audit/verify` and correlate changes to policy versions and assignments.
- Decision traces remain queryable (`/v2/decisions/search`) and can be joined with audit events for replay.

## Notes

- Anchoring of daily root hashes to an external store is planned. Today, verification is local using the hash chain.
