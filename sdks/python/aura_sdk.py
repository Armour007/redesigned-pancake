import base64
import hashlib
import hmac
import json
import os
import time
from functools import wraps
from typing import Any, Callable, Dict, Optional

import requests
from cryptography.hazmat.primitives.asymmetric import ed25519, ec, utils
from cryptography.hazmat.primitives import hashes

DEFAULT_BASE = (
    os.environ.get("AURA_API_BASE_URL")
    or os.environ.get("AURA_API_BASE")
    or "http://localhost:8081"
)

class AuraClient:
    def __init__(self, api_key: str, base_url: Optional[str] = None, version: Optional[str] = None):
        if not api_key:
            raise ValueError("api_key is required")
        self.api_key = api_key
        self.base_url = (base_url or DEFAULT_BASE).rstrip('/')
        self.version = version

    def verify(self, agent_id: str, request_context: Dict[str, Any]):
        url = f"{self.base_url}/v1/verify"
        headers = {
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
        }
        if self.version:
            headers["AURA-Version"] = self.version
        resp = requests.post(url, headers=headers, json={"agent_id": agent_id, "request_context": request_context})
        resp.raise_for_status()
        return resp.json()


def _default_context_builder(func: Callable, args: tuple, kwargs: dict) -> Dict[str, Any]:
    def safe(obj: Any) -> Any:
        try:
            json.dumps(obj)
            return obj
        except Exception:
            return str(obj)

    return {
        "function": getattr(func, "__name__", "unknown"),
        "args": [safe(a) for a in args],
        "kwargs": {k: safe(v) for k, v in kwargs.items()},
    }


def protect(
    agent_id: Optional[str] = None,
    *,
    client: Optional[AuraClient] = None,
    context_builder: Optional[Callable[[Callable, tuple, dict], Dict[str, Any]]] = None,
    on_deny: str | Callable[[str, Callable, tuple, dict], Any] = "raise",
):
    """
    Decorator that verifies with AURA before executing the wrapped function.

    - agent_id: UUID string; if omitted, uses env AURA_AGENT_ID.
    - client: optional AuraClient; if omitted, constructed from env AURA_API_KEY and AURA_API_BASE_URL.
    - context_builder: optional callable (func, args, kwargs) -> dict.
    - on_deny: 'raise' | 'return_none' | callable(reason, func, args, kwargs) -> Any
    """

    def decorator(func: Callable):
        local_client = client
        local_agent = agent_id or os.environ.get("AURA_AGENT_ID")

        @wraps(func)
        def wrapper(*args, **kwargs):
            nonlocal local_client, local_agent
            if local_client is None:
                api_key = os.environ.get("AURA_API_KEY")
                base = (
                    os.environ.get("AURA_API_BASE_URL")
                    or os.environ.get("AURA_API_BASE")
                    or DEFAULT_BASE
                )
                if not api_key:
                    raise RuntimeError("AURA_API_KEY is not set for aura.protect()")
                local_client = AuraClient(api_key=api_key, base_url=base)
            if not local_agent:
                raise RuntimeError("agent_id not provided and AURA_AGENT_ID is not set")

            builder = context_builder or _default_context_builder
            ctx = builder(func, args, kwargs)
            res = local_client.verify(local_agent, ctx)
            if not res or res.get("decision") != "ALLOWED":
                reason = (res or {}).get("reason", "access denied")
                if callable(on_deny):
                    return on_deny(reason, func, args, kwargs)
                if on_deny == "return_none":
                    return None
                raise PermissionError(f"AURA denied: {reason}")
            return func(*args, **kwargs)

        return wrapper

    return decorator


def compute_signature(secret: str, timestamp: int, payload: bytes) -> str:
    msg = f"{timestamp}.".encode() + payload
    return hmac.new(secret.encode(), msg, hashlib.sha256).hexdigest()


def verify_signature(secret: str, header: str, payload: bytes, tolerance: int = 300) -> bool:
    if not header:
        return False
    parts = dict(p.split("=") for p in header.split(","))
    try:
        ts = int(parts.get("t", "0"))
        sig = parts.get("v1")
    except Exception:
        return False
    if not ts or not sig:
        return False
    now = int(time.time())
    if abs(now - ts) > tolerance:
        return False
    expected = compute_signature(secret, ts, payload)
    try:
        return hmac.compare_digest(bytes.fromhex(expected), bytes.fromhex(sig))
    except Exception:
        return False


def build_request_signing_headers(secret: str, method: str, path: str, body: bytes | str = b"") -> Dict[str, str]:
    """
    Produce X-Aura-* headers for HMAC request signing.
    canonical = METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY
    """
    ts = str(int(time.time()))
    nonce = hashlib.sha256(os.urandom(16)).hexdigest()[:24]
    if isinstance(body, str):
        body = body.encode()
    unsigned = "\n".join([method.upper(), path, ts, nonce, body.decode()])
    sig = hmac.new(secret.encode(), unsigned.encode(), hashlib.sha256).hexdigest()
    return {
        "X-Aura-Timestamp": ts,
        "X-Aura-Nonce": nonce,
        "X-Aura-Signature": sig,
    }


# -----------------------------
# Trust tokens: offline verify
# -----------------------------

def _b64url_decode(data: str) -> bytes:
    # add padding if missing
    pad = '=' * (-len(data) % 4)
    return base64.urlsafe_b64decode(data + pad)


def fetch_jwks(base_url: Optional[str] = None, org_id: Optional[str] = None) -> dict:
    base = (base_url or DEFAULT_BASE).rstrip('/')
    if org_id:
        url = f"{base}/.well-known/aura/{org_id}/jwks.json"
    else:
        url = f"{base}/.well-known/aura-jwks.json"
    resp = requests.get(url, timeout=10)
    resp.raise_for_status()
    return resp.json()


def fetch_revocations(base_url: str, org_id: str, etag: Optional[str] = None):
    base = (base_url or DEFAULT_BASE).rstrip('/')
    url = f"{base}/organizations/{org_id}/trust-tokens/revocations"
    headers = {}
    if etag:
        headers["If-None-Match"] = etag
    resp = requests.get(url, headers=headers, timeout=10)
    if resp.status_code == 304:
        return [], etag, True
    resp.raise_for_status()
    new_etag = resp.headers.get("ETag")
    items = resp.json().get("items", [])
    return items, new_etag, False


def verify_trust_token_offline(
    token: str,
    *,
    base_url: Optional[str] = None,
    org_id: Optional[str] = None,
    grace_seconds: int = 0,
    revoked: Optional[set[str]] = None,
    cache: Optional["TrustCaches"] = None,
) -> dict:
    """
    Verify a trust token offline using JWKS and optional revocation set.

    Returns: { valid: bool, reason?: str, claims?: dict }
    """
    try:
        header_b64, payload_b64, sig_b64 = token.split(".")
    except ValueError:
        return {"valid": False, "reason": "bad_format"}

    try:
        header = json.loads(_b64url_decode(header_b64))
        claims = json.loads(_b64url_decode(payload_b64))
    except Exception:
        return {"valid": False, "reason": "bad_json"}

    alg = header.get("alg")
    kid = header.get("kid")
    # exp with grace
    exp = claims.get("exp")
    if not isinstance(exp, (int, float)):
        return {"valid": False, "reason": "missing_exp"}
    now = int(time.time())
    if now > int(exp) + int(grace_seconds):
        return {"valid": False, "reason": "expired"}
    jti = claims.get("jti")
    if jti and revoked and jti in revoked:
        return {"valid": False, "reason": "revoked"}

    if alg == "HS256":
        return {"valid": False, "reason": "hs256_offline_unsupported"}

    if cache:
        jwks = cache.get_jwks(base_url, org_id)
    else:
        jwks = fetch_jwks(base_url, org_id)
    keys = jwks.get("keys", [])
    key = None
    for k in keys:
        if not kid or k.get("kid") == kid:
            key = k
            break
    if not key:
        return {"valid": False, "reason": "kid_not_found"}

    signing_input = (header_b64 + "." + payload_b64).encode()
    try:
        sig = _b64url_decode(sig_b64)
    except Exception:
        return {"valid": False, "reason": "bad_sig"}

    try:
        if alg == "EdDSA" and key.get("kty") == "OKP" and key.get("crv") == "Ed25519":
            x_b = _b64url_decode(key.get("x", ""))
            pub = ed25519.Ed25519PublicKey.from_public_bytes(x_b)
            pub.verify(sig, signing_input)
            return {"valid": True, "claims": claims}
        if alg == "ES256" and key.get("kty") == "EC" and key.get("crv") == "P-256":
            x_b = _b64url_decode(key.get("x", ""))
            y_b = _b64url_decode(key.get("y", ""))
            if len(sig) != 64:
                return {"valid": False, "reason": "bad_sig"}
            r = int.from_bytes(sig[:32], "big")
            s = int.from_bytes(sig[32:], "big")
            der = utils.encode_dss_signature(r, s)
            pub_numbers = ec.EllipticCurvePublicNumbers(
                int.from_bytes(x_b, "big"), int.from_bytes(y_b, "big"), ec.SECP256R1()
            )
            pub = pub_numbers.public_key()
            pub.verify(der, signing_input, ec.ECDSA(hashes.SHA256()))
            return {"valid": True, "claims": claims}
        return {"valid": False, "reason": "unsupported_alg"}
    except Exception:
        return {"valid": False, "reason": "bad_sig"}


class TrustCaches:
    """Lightweight in-memory caches for JWKS and revocations with TTL and ETag support."""

    def __init__(self, jwks_ttl: int = 300, rev_ttl: int = 60):
        self._jwks_ttl = jwks_ttl
        self._rev_ttl = rev_ttl
        self._jwks_map: dict[tuple[str, str | None], tuple[float, dict]] = {}
        self._rev_map: dict[tuple[str, str], dict] = {}
        self._rev_etag: dict[tuple[str, str], str] = {}
        self._rev_at: dict[tuple[str, str], float] = {}

    def _key(self, base_url: Optional[str], org_id: Optional[str]):
        return ((base_url or DEFAULT_BASE).rstrip('/'), org_id)

    def get_jwks(self, base_url: Optional[str], org_id: Optional[str]):
        key = self._key(base_url, org_id)
        now = time.time()
        ts, data = self._jwks_map.get(key, (0.0, {}))
        if data and (now - ts) < self._jwks_ttl:
            return data
        # refresh
        fresh = fetch_jwks(base_url, org_id)
        self._jwks_map[key] = (now, fresh)
        return fresh

    def get_revocations(self, base_url: str, org_id: str):
        base = (base_url or DEFAULT_BASE).rstrip('/')
        key = (base, org_id)
        now = time.time()
        etag = self._rev_etag.get(key)
        last = self._rev_at.get(key, 0.0)
        if (now - last) < self._rev_ttl:
            # try conditional refresh but ok to be a no-op
            try:
                items, new_etag, not_mod = fetch_revocations(base, org_id, etag)
                if not not_mod and items:
                    self._rev_map[key] = {it.get("jti"): True for it in items if it.get("jti")}
                    self._rev_etag[key] = new_etag
                    self._rev_at[key] = now
            except Exception:
                pass
            return self._rev_map.get(key, {})
        # hard refresh
        items, new_etag, not_mod = fetch_revocations(base, org_id, etag)
        if not_mod:
            self._rev_at[key] = now
            return self._rev_map.get(key, {})
        self._rev_map[key] = {it.get("jti"): True for it in items if it.get("jti")}
        self._rev_etag[key] = new_etag
        self._rev_at[key] = now
        return self._rev_map[key]


# -----------------------------
# VC (JSON-LD) LDP verification
# -----------------------------

def _normalize_vc_urdna(vc: dict) -> str:
    # work on a shallow copy without proof
    try:
        from pyld import jsonld  # lazy import to avoid hard dependency when not used
    except Exception as e:
        raise ImportError("pyld_not_installed") from e
    doc = {k: v for k, v in vc.items() if k != "proof"}
    opts = {
        "algorithm": "URDNA2015",
        "format": "application/n-quads",
        "processingMode": "json-ld-1.1",
    }
    return jsonld.normalize(doc, options=opts)


def _resolve_did(base_url: str, did: str) -> dict:
    base = (base_url or DEFAULT_BASE).rstrip('/')
    url = f"{base}/resolve?did={did}"
    r = requests.get(url, timeout=10)
    r.raise_for_status()
    return r.json()


def verify_vc_ldp(vc: dict, *, base_url: Optional[str] = None, expected_org_id: Optional[str] = None, expected_owner: Optional[str] = None) -> dict:
    """
    Verify a JSON-LD VC with JsonWebSignature2020 proof using detached JWS (b64=false) over URDNA2015 N-Quads.
    Returns: { valid: bool, reason?: str }
    """
    try:
        proof = vc.get("proof") or {}
        jws = proof.get("jws")
        vmid = proof.get("verificationMethod")
        if not jws or not vmid:
            return {"valid": False, "reason": "invalid_proof"}
        parts = jws.split(".")
        if len(parts) != 3 or parts[1] != "":
            return {"valid": False, "reason": "not_detached"}
        # protected header
        ph = json.loads(_b64url_decode(parts[0]))
        alg = ph.get("alg")
        if ph.get("b64") is not False:
            return {"valid": False, "reason": "b64_must_be_false"}
        # normalize
        nquads = _normalize_vc_urdna(vc)
        signing_input = f"{parts[0]}.".encode() + nquads.encode()
        sig = _b64url_decode(parts[2])
        # resolve did
        did = vmid.split("#", 1)[0]
        doc = _resolve_did(base_url or DEFAULT_BASE, did)
        vms = {vm.get("id"): vm for vm in doc.get("verificationMethod", [])}
        vm = vms.get(vmid)
        if not vm:
            return {"valid": False, "reason": "vm_not_found"}
        jwk = vm.get("publicKeyJwk", {})
        if alg == "EdDSA" and jwk.get("kty") == "OKP" and jwk.get("crv") == "Ed25519":
            pub = ed25519.Ed25519PublicKey.from_public_bytes(_b64url_decode(jwk.get("x", "")))
            pub.verify(sig, signing_input)
        elif alg == "ES256" and jwk.get("kty") == "EC" and jwk.get("crv") == "P-256":
            x_b = _b64url_decode(jwk.get("x", ""))
            y_b = _b64url_decode(jwk.get("y", ""))
            if len(sig) != 64:
                return {"valid": False, "reason": "bad_sig"}
            r = int.from_bytes(sig[:32], "big")
            s = int.from_bytes(sig[32:], "big")
            der = utils.encode_dss_signature(r, s)
            pub_numbers = ec.EllipticCurvePublicNumbers(
                int.from_bytes(x_b, "big"), int.from_bytes(y_b, "big"), ec.SECP256R1()
            )
            pub = pub_numbers.public_key()
            pub.verify(der, signing_input, ec.ECDSA(hashes.SHA256()))
        else:
            return {"valid": False, "reason": "unsupported_alg"}
        # invariants
        if expected_org_id:
            iss = vc.get("issuer")
            want = f"did:aura:org:{expected_org_id}"
            if iss != want:
                return {"valid": False, "reason": "issuer_mismatch"}
        if expected_owner:
            cs = vc.get("credentialSubject") or {}
            if cs.get("owner") != expected_owner:
                return {"valid": False, "reason": "owner_mismatch"}
        return {"valid": True}
    except ImportError:
        return {"valid": False, "reason": "pyld_missing"}
    except Exception:
        return {"valid": False, "reason": "verify_error"}

