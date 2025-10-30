# Security hardening

This document summarizes request signing, mTLS for agents, and browser protections added to Aura.

## HMAC request signing

Agents and services can optionally sign requests using headers:

- X-Aura-Timestamp: Unix seconds
- X-Aura-Nonce: unique identifier per request (random)
- X-Aura-Signature: hex(HMAC-SHA256(secret, canonical))

Canonical message:

```
<METHOD>\n<PATH>\n<TIMESTAMP>\n<NONCE>\n<BODY>
```

Environment:

- AURA_REQUEST_HMAC_SECRET: shared secret (enables verification middleware)
- AURA_REQUEST_HMAC_MAX_SKEW_SECONDS (default 300)
- AURA_REQUEST_HMAC_TTL_SECONDS (nonce TTL, default 300)
- AURA_REDIS_ADDR / AURA_REDIS_PASSWORD / AURA_REDIS_DB (optional Redis for nonce replay cache)

## Agent mTLS and CSR issuance (MVP)

- POST /v1/agents/:agentId/csr — Accepts a CSR (PEM) and optional TPM/TEE evidence (stored as audit) and returns a signed client certificate. Protected by attestation/API key middleware.
- Client certificate binding middleware extracts subject CN as `agentID` and attaches `agentCertFP` (SHA-256 fingerprint).

Environment for CA and TLS:

- AURA_AGENT_CA_CERT_FILE, AURA_AGENT_CA_KEY_FILE — CA certificate and private key used to sign agent CSRs
- AURA_AGENT_CERT_TTL_HOURS — Client certificate TTL (default 24h)
- AURA_TLS_CERT_FILE, AURA_TLS_KEY_FILE — Enable HTTPS server
- AURA_CLIENT_CA_FILE — Client CA bundle for verifying agent client certificates
- AURA_TLS_CLIENT_AUTH — `require` | `verify` | `off` (default off)

## Browser protections

- CSP: set via `AURA_CSP_POLICY`, default is strict and denies inline scripts.
- HSTS: enable with `AURA_HSTS_ENABLE=1`; optionally set `AURA_HSTS_MAX_AGE` and `AURA_HSTS_INCLUDE_SUBDOMAINS`.
- CSRF: enable with `AURA_CSRF_ENABLE=1`; dashboard routes (JWT protected) enforce a double-submit token (`X-CSRF-Token` matches `csrf_token` cookie).
- CORS: enable strict mode with `AURA_CORS_STRICT=1`, and specify `AURA_CORS_ORIGINS` or `AURA_FRONTEND_BASE_URL`.

## CI security scanning

- CodeQL SAST is configured for Go and JavaScript.
- OWASP ZAP baseline passive DAST can be triggered manually or via schedule; it targets the dev frontend (http://localhost:5173) started with docker-compose.dev.yml.

## Notes

- In production behind a reverse proxy/ingress, terminate TLS there or here; when using mTLS at the proxy, forward verified client identity via standard headers or use the TLS passthrough mode to this server.
- The CSR issuance endpoint performs minimal checks; integrate platform-specific attestation validation (TPM/TEE) in a future iteration.
