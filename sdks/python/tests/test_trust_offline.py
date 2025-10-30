import json
import time
import base64
import unittest
from unittest.mock import patch, Mock

from cryptography.hazmat.primitives.asymmetric import ed25519, ec, utils
from cryptography.hazmat.primitives import hashes, serialization

from aura_sdk import verify_trust_token_offline, TrustCaches


def b64u(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).rstrip(b"=").decode()


def make_jwt(header: dict, claims: dict, signer) -> str:
    hb = b64u(json.dumps(header).encode())
    pb = b64u(json.dumps(claims).encode())
    unsigned = (hb + "." + pb).encode()
    sig = signer(unsigned)
    return hb + "." + pb + "." + b64u(sig)


class TestOfflineVerify(unittest.TestCase):
    def test_ed25519_valid(self):
        priv = ed25519.Ed25519PrivateKey.generate()
        pub = priv.public_key()
        pub_bytes = pub.public_bytes(encoding=serialization.Encoding.Raw, format=serialization.PublicFormat.Raw)
        kid = "k-ed"
        jwks = {"keys": [{"kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "kid": kid, "x": b64u(pub_bytes)}]}

        def fake_get(url, *args, **kwargs):
            if "/.well-known/" in url:
                m = Mock()
                m.status_code = 200
                m.json = lambda: jwks
                return m
            raise AssertionError("unexpected url: " + url)

        exp = int(time.time()) + 300
        token = make_jwt({"alg": "EdDSA", "kid": kid}, {"exp": exp, "jti": "j1"}, lambda u: priv.sign(u))

        with patch("aura_sdk.requests.get", side_effect=fake_get):
            res = verify_trust_token_offline(token)
            self.assertTrue(res.get("valid"), res)

    def test_es256_valid(self):
        priv = ec.generate_private_key(ec.SECP256R1())
        pub = priv.public_key()
        numbers = pub.public_numbers()
        x = numbers.x.to_bytes(32, "big")
        y = numbers.y.to_bytes(32, "big")
        kid = "k-es"
        jwks = {"keys": [{"kty": "EC", "crv": "P-256", "alg": "ES256", "kid": kid, "x": b64u(x), "y": b64u(y)}]}

        def fake_get(url, *args, **kwargs):
            if "/.well-known/" in url:
                m = Mock()
                m.status_code = 200
                m.json = lambda: jwks
                return m
            raise AssertionError("unexpected url: " + url)

        exp = int(time.time()) + 300
        def signer(u: bytes) -> bytes:
            der = priv.sign(u, ec.ECDSA(hashes.SHA256()))
            r, s = utils.decode_dss_signature(der)
            rb = r.to_bytes(32, "big")
            sb = s.to_bytes(32, "big")
            return rb + sb

        token = make_jwt({"alg": "ES256", "kid": kid}, {"exp": exp, "jti": "j2"}, signer)
        with patch("aura_sdk.requests.get", side_effect=fake_get):
            res = verify_trust_token_offline(token)
            self.assertTrue(res.get("valid"), res)

    def test_revocations_etag(self):
        # JWKS
        jwks = {"keys": []}
        etag_value = 'W/"abc"'
        items = {"items": [{"jti": "r1"}]}

        def fake_get(url, headers=None, *args, **kwargs):
            m = Mock()
            if "/.well-known/" in url:
                m.status_code = 200
                m.json = lambda: jwks
                return m
            if "/revocations" in url:
                inm = (headers or {}).get("If-None-Match")
                if inm == etag_value:
                    m.status_code = 304
                    return m
                m.status_code = 200
                m.headers = {"ETag": etag_value}
                m.json = lambda: items
                return m
            raise AssertionError("unexpected url: " + url)

        cache = TrustCaches(jwks_ttl=1, rev_ttl=60)
        with patch("aura_sdk.requests.get", side_effect=fake_get):
            rev = cache.get_revocations("http://x", "org1")
            self.assertIn("r1", rev)
            # second call should use ETag and treat as not modified
            rev2 = cache.get_revocations("http://x", "org1")
            self.assertIn("r1", rev2)


if __name__ == "__main__":
    unittest.main()
