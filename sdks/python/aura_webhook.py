import hmac
import hashlib
import time
from typing import Optional


def _parse_signature(header: str) -> Optional[tuple[int, str]]:
    if not header:
        return None
    parts = [p.strip() for p in header.split(',')]
    t = None
    v1 = None
    for p in parts:
        kv = p.split('=', 1)
        if len(kv) != 2:
            continue
        k, v = kv
        if k == 't':
            try:
                t = int(v)
            except ValueError:
                return None
        elif k == 'v1':
            v1 = v
    if t is None or v1 is None:
        return None
    return t, v1


def verify_aura_signature(raw_body: bytes, signature_header: str, secret: str, tolerance_sec: int = 300) -> bool:
    parsed = _parse_signature(signature_header)
    if not parsed:
        return False
    ts, v1 = parsed
    payload = f"{ts}.{raw_body.decode('utf-8')}".encode('utf-8')
    mac = hmac.new(secret.encode('utf-8'), payload, hashlib.sha256).hexdigest()

    # constant-time compare
    try:
        if not hmac.compare_digest(bytes.fromhex(mac), bytes.fromhex(v1)):
            return False
    except ValueError:
        return False

    now = int(time.time())
    return abs(now - ts) <= tolerance_sec
