# AURA Runbooks

This document summarizes common alerts, symptoms, quick queries, and tuning tips.

## Alerts

- HTTP5xxRateHigh
  - Symptom: 5xx error rate > 1 req/s for 5m
  - Quick queries:
    - `sum(rate(aura_http_requests_total{status=~"5.."}[5m]))`
    - Breakdown by path: `sum(rate(aura_http_requests_total{status=~"5.."}[5m])) by (path)`
  - Checklist: recent deploy? DB connectivity? auth middleware errors? application logs
  - Tuning: Raise threshold or extend `for:` window if expected bursty errors

- VerifyP95LatencyHigh
  - Symptom: p95 latency > 1s for 10m
  - Quick queries:
    - `histogram_quantile(0.95, sum(rate(aura_http_request_duration_seconds_bucket[5m])) by (le, path))`
  - Checklist: DB slow queries, CPU saturation, GC pauses, external integrations
  - Tuning: Adjust CPU/memory, HPA targets, DB indexes; raise threshold if workloads permit

- ExternalErrorRateHigh
  - Symptom: External op error ratio > 20% for 10m (e.g., SMTP, codegen)
  - Quick queries:
    - `sum(rate(aura_external_op_total{outcome="error"}[5m])) by (op)`
    - Compare with successes: `sum(rate(aura_external_op_total[5m])) by (op)`
  - Checklist: external service health, timeouts/retries, breaker state
  - Tuning: Backoff knobs, timeouts, breaker thresholds

- CircuitBreakerOpen
  - Symptom: Breaker open > 0 for 5m
  - Quick queries: `aura_circuit_breaker_open`
  - Checklist: upstream outage, configuration drift; verify breaker config env

- DLQDepthHigh (codegen)
  - Symptom: DLQ depth > 50 for 10m
  - Quick queries: `aura_dlq_depth{stream="codegen"}`; top reasons panel
  - Checklist: failing jobs; check recent worker deploys and inputs

- QueuePendingHigh (codegen)
  - Symptom: consumer lag > 50 for 10m
  - Quick queries: `aura_queue_pending{stream="codegen"}`
  - Checklist: worker downscaled? Redis issue? hot partitions?
  - Tuning: increase workers (`AURA_QUEUE_WORKERS`), relax rate limits

## Operational tips

- Readiness gates
  - `/readyz` requires DB ping; when queue is enabled, also requires Redis ping.
- Drain mode
  - `POST /admin/queue/drain {"enable": true}` to stop new reads; reclaimer finishes pending.
  - `GET /admin/queue/drain/status` for snapshot (pending, DLQ size).
  - `GET /admin/queue/drain/complete` returns `{drained: true}` when consecutive empty-pending ticks threshold is met; safe to terminate workers.
- Webhook failures
  - Webhook delivery retries 3x with exponential backoff; failed deliveries are pushed to `aura:webhooks:dlq`.
  - List entries: `GET /admin/webhooks/dlq`.
  - Retry selected/all: `POST /admin/webhooks/dlq/requeue` with `{ids?: string[], all?: bool, count?: number}`.
  - Delete selected/all: `POST /admin/webhooks/dlq/delete` with `{ids?: string[], all?: bool, count?: number}`.
- API key rotation
  - `POST /organizations/:orgId/apikeys/:keyId/rotate` returns a new one-time secret; hash is stored server-side.
