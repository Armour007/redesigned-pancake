# AURA — The Trust Protocol for Autonomous Systems

This repo contains:
- backend (Go + Gin + PostgreSQL) — core API, verification engine, JWT auth
- frontend (SvelteKit + Tailwind) — dashboard for agents, rules, API keys, and logs

Quick start (local):
1) Start Postgres
	- Edit docker-compose.yml DB_PASSWORD
	- Run: docker compose up -d

2) Backend env (.env at project root or in backend/):
	- DB_HOST=localhost
	- DB_PORT=5432
	- DB_USER=aura_user
	- DB_PASSWORD=your_strong_password
	- DB_NAME=aura_db
	- DB_SSLMODE=disable
	- JWT_SECRET=replace-with-a-strong-secret

3) Run backend
	- cd backend
	- go build ./...
	- go run ./cmd/server

4) Run frontend
	- cd frontend
	- npm install
	- npm run dev

Migrations: create tables under backend/db/migrations (use your preferred tool to apply).

Key API highlights:
- POST /auth/register → returns user_id and organization_id (frontend stores organization_id in localStorage)
- POST /auth/login → returns JWT token (frontend stores token)
- Agents CRUD under /organizations/:orgId/agents
- Rules under /organizations/:orgId/agents/:agentId/permissions
- API keys under /organizations/:orgId/apikeys
- Verify: POST /v1/verify — requires header X-API-Key: aura_sk_<secret>

Headers and versioning:
- AURA-Version: optional request header to pin API behavior (default: 2025-10-01). Every response includes X-AURA-Version.
- Idempotency-Key: for POST mutations under /organizations/:orgId (e.g., create agent/key). The first 2xx is cached (24h). Retries with the same key return the same result with header X-Idempotent-Replay: true.

Verify request body:
{
  "agent_id": "<uuid>",
  "request_context": { "action": "deploy:prod", "branch": "main" }
}

Decision response:
{ "decision": "ALLOWED" | "DENIED", "reason": "..." }

Rule JSON examples:
- Allow exact action with optional conditions
  { "effect": "allow", "action": "read:db", "context": { "table": {"eq": "users"} } }

- Deny within a time window (Mon–Fri, 09:00–18:00 IST)
  { "effect": "deny", "action": "deploy:prod", "time_window": { "days": ["Mon","Tue","Wed","Thu","Fri"], "start": "09:00", "end": "18:00", "tz": "Asia/Kolkata" } }

Minimal Python usage (no SDK yet):

import os, requests

VERIFY_URL = "http://localhost:8080/v1/verify"
api_key = os.getenv("AURA_API_KEY") # aura_sk_...
agent_id = os.getenv("AURA_AGENT_ID")

payload = {"agent_id": agent_id, "request_context": {"action": "deploy:prod"}}
resp = requests.post(VERIFY_URL, json=payload, headers={"X-API-Key": api_key})
print(resp.status_code, resp.json())

Notes:
- /v1/verify authenticates API keys; keys are hashed server-side and identified by an 8-char prefix.
- The engine supports effect allow/deny, simple AND conditions with eq/neq/gt/gte/lt/lte/contains, and optional time_window.

Try-it curl:

	Client guide
	See docs/CLIENT_GUIDE.md for end-to-end setup (SDK usage, webhooks, versioning/idempotency, and observability with Grafana + Tempo).

```powershell
# Verify decision (PowerShell)
curl -Method POST -Uri "http://localhost:8080/v1/verify" -Headers @{
	'Content-Type' = 'application/json';
	'X-API-Key'    = $env:AURA_API_KEY;
	'AURA-Version' = '2025-10-01'
} -Body (@{ agent_id = $env:AURA_AGENT_ID; request_context = @{ action = 'deploy:prod'; branch = 'main' } } | ConvertTo-Json)
	Observability:
	- Prometheus metrics at /metrics. HTTP durations include exemplars with trace_id when tracing is enabled.
	- Tracing enabled via AURA_OTEL_ENABLE=true. The included docker-compose also runs an OTEL collector and a Tempo tracing backend; Grafana is pre-wired to both Prometheus and Tempo for out-of-the-box metrics and traces.
	- Optional metric label: set AURA_METRICS_LABELS_ORG=true to add an org label to verify_decisions_total. Beware of high-cardinality.
```

SDKs:
- Node: `sdks/node` — AuraClient.verify() and webhooks verify helper; example at `sdks/node/examples/express`.
- Python: `sdks/python` — AuraClient.verify() and webhooks verify helper; example at `sdks/python/examples/fastapi`.
- Go: `sdks/go/aura` — client and webhooks verify; example at `sdks/go/aura/examples/nethttp`.

Observability:
- Prometheus metrics at /metrics. HTTP durations include exemplars with trace_id when tracing is enabled.
- Enable OpenTelemetry tracing via env:
	- AURA_OTEL_ENABLE=true
	- OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
- Optional metric label: set AURA_METRICS_LABELS_ORG=true to add an org label to verify_decisions_total. Beware of high-cardinality.

## Deploy

Docker Compose (production-ish):
- File: `docker-compose.deploy.yml`
- Images: `ghcr.io/Armour007/aura-backend:latest` and `ghcr.io/Armour007/aura-frontend:latest`
- Frontend expects PUBLIC_API_BASE to be baked at build time. Set GitHub Actions variable `PUBLIC_API_BASE` to your backend URL (e.g., `https://api.example.com`) so CI builds the frontend with the correct base.
- Quick start:
	1) Set environment variables (PowerShell example):
		 - `$env:POSTGRES_PASSWORD='supersecret'`
		 - `$env:JWT_SECRET='please_change'`
		 - `$env:AURA_CORS_ORIGINS='http://localhost:3000'`
	2) Run: `docker compose -f docker-compose.deploy.yml up -d`

Kubernetes:
- Manifests under `deploy/k8s/`: namespace, postgres, redis (optional), backend/ frontend deployments+services, and ingress.
- Ensure your CI publishes the frontend image with `PUBLIC_API_BASE` set to your backend external URL. At runtime, PUBLIC_ envs are compiled into the frontend bundle.
- Apply:
	- `kubectl apply -f deploy/k8s/namespace.yaml`
	- `kubectl apply -f deploy/k8s/secret-backend.yaml`
	- `kubectl apply -f deploy/k8s/configmap-backend.yaml`
	- `kubectl apply -f deploy/k8s/postgres.yaml`
	- `kubectl apply -f deploy/k8s/redis.yaml`
	- `kubectl apply -f deploy/k8s/deployment-backend.yaml`
	- `kubectl apply -f deploy/k8s/deployment-frontend.yaml`
	- `kubectl apply -f deploy/k8s/ingress.yaml` (update hosts)

Backend hardening via env:
- `AURA_CORS_ORIGINS` (comma-separated)
- `AURA_TRUSTED_PROXIES` (comma-separated)
- Rate limiting: `AURA_V1_VERIFY_RPM` and optional Redis `AURA_REDIS_ADDR`

## Optional: SpiceDB for Trust Graph

You can run AURA's Trust Graph using the default local SQL backend or switch to SpiceDB.

- See `docs/SPICEDB.md` for a quick start (Docker run, schema apply, and env to enable).
- Build with SpiceDB support using Go build tags: `go build -tags=spicedb ./...`
- Enable via env:
	- `AURA_REL_BACKEND=spicedb`
	- `AURA_SPICEDB_ENDPOINT` (e.g., `localhost:50051`)
	- `AURA_SPICEDB_TOKEN` (e.g., `dev-secret`)
	- Optional caches: `AURA_REL_CACHE_TTL_MS` and `AURA_REL_NEG_CACHE_TTL_MS`
