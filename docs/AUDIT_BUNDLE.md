# Audit Bundle Export

This document explains the audit bundle export feature and how to consume its output.

## Endpoint

GET /organizations/{orgId}/regulator/audit-bundle

Query parameters:
- from (optional, RFC3339): start of time range. Defaults to 30 days before `to`.
- to (optional, RFC3339): end of time range. Defaults to now (UTC).
- as (optional): when set to `zip`, the response is a ZIP archive containing a single `bundle.json`.

## Response

- application/json
  {
    "from": "2025-09-30T00:00:00Z",
    "to":   "2025-10-30T12:34:56Z",
    "bundle": {
      "event_logs": [...],
      "audit_ledger": [...],
      "decision_traces": [...],
      "policy_versions": [...],
      "trust_keys": [...],
      "trust_token_revocations": [...]
    }
  }

- application/zip
  The ZIP contains a single file `bundle.json` with the same structure as above.

Each array contains JSON objects as stored in the database. Field sets may evolve over time.

## Usage notes

- Use narrower date windows for faster responses, e.g., `from` = 7 days ago.
- For archival or regulator handoff, prefer `?as=zip` and store immutably with a content hash.
- Combine with `GET /organizations/{orgId}/regulator/snapshot` for a quick high‑level summary.
- Audit anchors (root hashes) are available via `GET /v2/audit/anchor`.

## Example

Request:
GET /organizations/00000000-0000-0000-0000-000000000000/regulator/audit-bundle?from=2025-10-01T00:00:00Z&to=2025-10-15T00:00:00Z

Response (JSON):
{
  "from": "2025-10-01T00:00:00Z",
  "to": "2025-10-15T00:00:00Z",
  "bundle": {
    "event_logs": [{"id":"...","event":"policy_version_activated", ...}],
    "audit_ledger": [{"id":"...","event":"audit_anchor_set", ...}],
    "decision_traces": [],
    "policy_versions": [{"policy_id":"...","version":2, ...}],
    "trust_keys": [{"id":"...","active":true, ...}],
    "trust_token_revocations": []
  }
}
