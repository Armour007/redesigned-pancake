import hashlib
import hmac
import json
import os
import time
from typing import Any, Dict, Optional

import requests

DEFAULT_BASE = os.environ.get("AURA_API_BASE", "http://localhost:8080")

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
