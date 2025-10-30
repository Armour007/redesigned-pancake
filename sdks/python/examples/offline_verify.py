import os
import time
from aura_sdk import verify_trust_token_offline, TrustCaches

base = os.environ.get("AURA_API_BASE_URL", "http://localhost:8081")
org = os.environ.get("AURA_ORG_ID")
token = os.environ.get("AURA_TRUST_TOKEN")

if not token:
    print("Set AURA_TRUST_TOKEN to a token to verify")
    raise SystemExit(2)

cache = TrustCaches(jwks_ttl=300, rev_ttl=60)

res = verify_trust_token_offline(token, base_url=base, org_id=org, grace_seconds=10, cache=cache)
if not res.get("valid"):
    print("invalid:", res.get("reason"))
    raise SystemExit(1)
print("valid claims:", res.get("claims"))
