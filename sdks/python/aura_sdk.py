import hashlib
import hmac
import json
import os
import time
from functools import wraps
from typing import Any, Callable, Dict, Optional

import requests

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
