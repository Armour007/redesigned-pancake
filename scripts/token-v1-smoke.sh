#!/usr/bin/env bash
set -euo pipefail

API_BASE="${AURA_API_BASE_URL:-http://localhost:8081}"
echo "Using API base: ${API_BASE}"

ORG_ID="org_smoke"

# Optional insert of a local Ed25519 org key via psql if available
if command -v psql >/dev/null 2>&1; then
  echo "Attempting to insert local trust key via psql for ${ORG_ID} ..."
  # 32-byte seed
  SEED=$(head -c 32 /dev/urandom | base64 | tr '+/' '-_' | tr -d '=')
  KID="smk_$(head -c 4 /dev/urandom | hexdump -e '4/1 "%02x"')"
  SQL="\
INSERT INTO trust_keys(org_id, alg, ed25519_private_key_base64, kid, active, provider, key_ref, key_version, provider_config, jwk_pub, created_at) \
VALUES ('${ORG_ID}','EdDSA','${SEED}','${KID}',true,'local',NULL,NULL,'{}'::jsonb,'{}'::jsonb,NOW()) \
ON CONFLICT DO NOTHING;"
  PGPASSWORD="${DB_PASSWORD:-your_strong_password}" psql \
    -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" \
    -U "${DB_USER:-aura_user}" -d "${DB_NAME:-aura_db}" \
    -c "$SQL" || echo "psql insert failed (non-fatal)"
else
  echo "psql not found; skipping org trust key insert."
fi

# Issue v1 token
ISSUE_BODY=$(jq -nc --arg org "$ORG_ID" '{org_id:$org, sub:"user_smoke", aud:"svc", action:"read", resource:"doc:42", ttl_sec:300}')
RESP=$(curl -sS -X POST "$API_BASE/v1/token/issue" -H 'Content-Type: application/json' -d "$ISSUE_BODY")
ALG=$(echo "$RESP" | jq -r .alg)
TOKEN=$(echo "$RESP" | jq -r .token)
JTI=$(echo "$RESP" | jq -r .jti)
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then echo "Issue failed: $RESP"; exit 1; fi
echo "Issued token alg=$ALG jti=$JTI"

# Verify v1 token
VERIFY_BODY=$(jq -nc --arg t "$TOKEN" '{token:$t}')
VOUT=$(curl -sS -X POST "$API_BASE/v1/token/verify" -H 'Content-Type: application/json' -d "$VERIFY_BODY")
VALID=$(echo "$VOUT" | jq -r .valid)
REASON=$(echo "$VOUT" | jq -r .reason)
if [ "$VALID" != "true" ]; then echo "Verify failed: $VOUT"; exit 1; fi

echo "Verify: valid=$VALID reason=$REASON"

# Revoke token by JTI
EXP=$(($(date +%s)+300))
REVOKE_BODY=$(jq -nc --arg org "$ORG_ID" --arg j "$JTI" --argjson exp $EXP '{org_id:$org, jti:$j, exp:$exp}')
RRESP=$(curl -sS -X POST "$API_BASE/v1/token/revoke" -H 'Content-Type: application/json' -d "$REVOKE_BODY")
echo "Revoke: $RRESP"

# Introspect mark_used twice to assert replay detection
INT_BODY=$(jq -nc --arg t "$TOKEN" '{token:$t, mark_used:true}')
INT1=$(curl -sS -X POST "$API_BASE/v2/tokens/introspect" -H 'Content-Type: application/json' -d "$INT_BODY")
INT2=$(curl -sS -X POST "$API_BASE/v2/tokens/introspect" -H 'Content-Type: application/json' -d "$INT_BODY")
V1=$(echo "$INT1" | jq -r .valid)
V2=$(echo "$INT2" | jq -r .valid)
R2=$(echo "$INT2" | jq -r .reason)
echo "Introspect1: valid=$V1"
echo "Introspect2: valid=$V2 reason=$R2"
if [ "$V1" != "true" ] || [ "$V2" = "true" ]; then echo "Replay detection failed"; exit 1; fi

echo "Token v1 smoke: OK"
