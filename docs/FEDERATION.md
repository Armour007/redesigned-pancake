# AURA Federation Protocol (MVP)

This document describes the minimal gossip protocol for federating AURA nodes to exchange signed proofs of security-relevant state. The goal is eventual convergence with weak consensus using a per-topic hash chain, with optional daily anchoring.

## Topics

- revocation: trust token JTI revocations and related notices
- key_rotation: rotations of org trust keys (KMS or local)
- org_registry: org and agent registry facts (public facets)

## Message format

Each message is a compact JWS over canonical JSON with fields:

- org_id: UUID string – the sender’s org
- topic: one of revocation, key_rotation, org_registry
- ts: RFC3339Nano timestamp (UTC)
- nonce: random hex string (replay separation)
- parents: optional array of parent hashes (DAG merge). Absent for linear append; present when creating a merge of multiple heads.
- payload: JSON object – topic-specific content

The payload canon is stable (struct layout) and the JWS uses normal base64 payload encoding (b64=true). Header fields:

- alg: EdDSA (Ed25519) or ES256 (P-256)
- kid: key identifier

Nodes verify JWS using the sender org’s org-scoped JWKS at `/.well-known/aura/:orgId/jwks.json` (or from the DB if local/KMS keys are registered).

## Hash chain and DAG merges (weak consensus)

Per topic, the head is computed as:

- canonical = canonicalize({ org_id, topic, ts, nonce, parents?, payload })
- if parents is present and non-empty: hash = sha256_hex(sort(parents) || canonical)
- else: let prev_hash = head(hash) of the latest entry for topic (or empty), then hash = sha256_hex(prev_hash || canonical)

Each row stores parents[] (nullable), prev_hash (for backwards-compat), and hash. Verification prefers parents when present.

## Anchoring

Optionally, nodes can anchor the current head into the audit_anchors table once per day, keyed by `AURA_FEDERATION_ANCHOR_ORG_ID`. We set `external_ref = "federation:<topic>"` to differentiate topics.

Pull responses include the latest anchor reference when configured:

```
GET /v2/federation/gossip?topic=revocation
{
  "messages": [...],
  "head": "<hex>",
  "anchor": { "org_id": "...", "date": "YYYY-MM-DD", "root_hash": "<hex>", "external_ref": "federation:revocation" }
}
```

## Endpoints

- POST /v2/federation/gossip – ingest externally signed messages (batch). Validates JWS and appends to the chain.
- GET /v2/federation/gossip – pull recent messages; returns current head (and anchor if configured).
- POST /v2/federation/gossip/publish – sign and append a local message with the org’s active trust key.
- POST /v2/federation/gossip/merge – create a merge message that references multiple parents to converge branches.
- GET/POST/DELETE /v2/federation/peers – manage peer list (URLs).

## Peer-to-peer

An optional publisher can push recent local messages to configured peers and auto-merge multiple heads for faster convergence:

- Enable via `AURA_GOSSIP_PUBLISH_ENABLE=1`
- POST batches to peer `/v2/federation/gossip`
- Best-effort; network errors are ignored
- If `AURA_FEDERATION_ANCHOR_ORG_ID` is set and multiple heads exist for a topic, the node will attempt to create a signed merge message (parents = heads) using that org’s active trust key.

Recommended: pair with periodic pull to reconcile and prevent gaps. Future work includes CRDT-like set reconciliation and deduplication via unique constraints on `hash`.

## Robustness & Safety

- Rate limiting: per-topic/org ingest cap in the last minute (default 200 per minute). Configure `AURA_GOSSIP_INGEST_RATE_PER_MIN`.
- Timestamp bounds: reject messages with `ts > now+5m`.
- Deterministic canonicalization: stable struct layout and RFC3339Nano encoding.
- Signature algorithms: EdDSA (Ed25519) and ES256 (P-256). HS256 is not accepted.

### Uniqueness and deduplication

- We enforce uniqueness with a composite unique index on `(topic, hash)`. The hash already includes the topic in its canonical input, but the composite index makes scoping explicit and keeps the schema self-descriptive.
- Ingest uses `ON CONFLICT DO NOTHING`, so replays are safely ignored.

## Testing

- Unit: canonicalization stability and signature verification helpers.
- Integration (optional): enable with `AURA_TEST_LIVE_DB=1` and run `go test ./...`. The test inserts a local trust key for a temporary org, ingests two signed messages, and verifies the final chain head deterministically.

## Configuration summary

- `AURA_FEDERATION_ANCHOR_ORG_ID` – UUID of the org that owns audit anchors.
- `AURA_GOSSIP_PUBLISH_ENABLE` – when `1`, enable background publisher.
- `AURA_GOSSIP_INGEST_RATE_PER_MIN` – max messages per topic/org per minute (default 200).

## Future: CRDT/DAG superset

To strengthen reconciliation and allow multi-parent merges (when multiple branches form), we can extend the schema and hash definition:

- Add a `parents text[]` column to hold zero or more parent heads (superset of the current single `prev_hash`).
- Define `hash = sha256_hex(sort(parents) || canonical)` where `sort(parents)` is the lexicographically sorted concatenation of parent hashes (empty string when no parents).
- During publish, default `parents = [current_head]` for a linear chain; during reconciliation, a node can create a merge message with `parents = [headA, headB, ...]` to converge branches deterministically.
- Optional: include a compact causal clock (Lamport/vector) in the canonical payload for richer conflict semantics.

This is compatible with the current implementation (single-parent), and can be introduced additively with read-path updates that prefer `parents` when present and fall back to `prev_hash` otherwise.
