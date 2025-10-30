# Device Attestation and mTLS

This guide explains how AURA ties hardware/remote attestation to client certificate issuance so only compliant devices receive mTLS credentials.

## Flow overview

1. Device performs attestation (TPM/TEE) and sends it to AURA
   - POST /v2/attest { type: "tpm"|"aws_nitro"|"azure_snp"|"intel_tdx"|"gcp_cc", payload: {...} }
   - The backend verifies attestation, computes a stable device fingerprint, stores posture, and marks `posture_ok`.
2. Certificate issuance
   - POST /v2/certs/issue { device_id, subject_cn, days }
   - Requires `posture_ok` and a recent attestation (default 24h). Returns an Ed25519 client cert (PEM) signed by the org CA.
3. Client connects using mTLS
   - Use the issued certificate in TLS client auth. The server can validate client auth and map `serial` to device/subject.

## Endpoints

- POST /v2/attest
  - Verifies the provided attestation and stores a `device` with posture
  - Response: { device_id, posture_ok }
- POST /v2/certs/issue
  - Issues an X.509 client certificate if posture_ok and fresh
  - Response: { serial, cert_pem, not_before, not_after }
- GET /v2/certs
  - Lists issued client certificates for the org
- POST /v2/certs/{serial}/revoke
  - Revokes a client certificate
- GET /v2/certs/crl.pem (optional in dev)
  - Returns a CRL if a CA private key is available in dev mode

## Policy integration (posture gating)

You can use your existing policy to enforce posture. When issuing a cert, AURA evaluates a policy with an input context like:

```
{
  "action": "issue_cert",
  "device": {
    "id": "...",
    "posture": { ... },
    "last_attested_at": "..."
  }
}
```

- If the policy returns `allow`, the certificate is issued
- If `deny` or `require_approval`, issuance is blocked (approval-based issuance can be implemented on demand)

## Providers

- TPM (dev stub): accepts `ek_pub`, `ak_pub`, PCRs, quote; computes a fingerprint and marks posture_ok=true (for development only)
- AWS Nitro, Azure SNP/TDX, GCP Confidential: production verifiers validate signatures and measurements against vendor chains; the interface supports pluggable verifiers

## Operations

- Rotate per-org client CA via DB or admin tools. In development, the CA is created automatically on first issuance
- List and revoke client certs; CRL endpoint helps mTLS servers quickly reject revoked certs

## Notes

- In production, store CA private keys in KMS (key_ref) and do not persist key_pem
- Add monitoring for attestation freshness and revocation activity
- Tie device_id/serial into your identity/auditing pipeline for complete traceability
