import base64
import json
import time
import unittest
from unittest.mock import patch, Mock

from cryptography.hazmat.primitives.asymmetric import ed25519
from cryptography.hazmat.primitives import serialization
try:
    from pyld import jsonld
    HAS_PYLD = True
except Exception:
    HAS_PYLD = False

from aura_sdk import verify_vc_ldp


def b64u(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).rstrip(b"=").decode()


@unittest.skipUnless(HAS_PYLD, "pyld not installed")
class TestVCLDP(unittest.TestCase):
    def test_vc_ldp_ed25519(self):
        # Keys and DID Document
        priv = ed25519.Ed25519PrivateKey.generate()
        pub = priv.public_key()
        pub_bytes = pub.public_bytes(encoding=serialization.Encoding.Raw, format=serialization.PublicFormat.Raw)
        did = "did:aura:org:00000000-0000-0000-0000-000000000001"
        vmid = did + "#key-1"
        did_doc = {
            "@context": ["https://www.w3.org/ns/did/v1"],
            "id": did,
            "verificationMethod": [
                {
                    "id": vmid,
                    "type": "JsonWebKey2020",
                    "controller": did,
                    "publicKeyJwk": {"kty": "OKP", "crv": "Ed25519", "x": b64u(pub_bytes)},
                }
            ],
            "assertionMethod": [vmid],
        }

        # Minimal VC
        vc = {
            "@context": [
                "https://www.w3.org/2018/credentials/v1",
            ],
            "type": ["VerifiableCredential"],
            "issuer": did,
            "issuanceDate": "2025-10-29T00:00:00Z",
            "credentialSubject": {"owner": "alice"},
        }

        # Normalize without proof (URDNA2015 N-Quads)
        nquads = jsonld.normalize({k: v for k, v in vc.items() if k != "proof"}, options={
            "algorithm": "URDNA2015",
            "format": "application/n-quads",
            "processingMode": "json-ld-1.1",
        })

        # Protected header: detached JWS with b64=false
        protected = {"alg": "EdDSA", "b64": False, "crit": ["b64"]}
        ph_b64 = b64u(json.dumps(protected).encode())
        signing_input = (ph_b64 + ".").encode() + nquads.encode()
        sig = priv.sign(signing_input)
        jws = ph_b64 + ".." + b64u(sig)

        vc_with_proof = dict(vc)
        vc_with_proof["proof"] = {
            "type": "JsonWebSignature2020",
            "created": "2025-10-29T00:00:00Z",
            "verificationMethod": vmid,
            "proofPurpose": "assertionMethod",
            "jws": jws,
        }

        def fake_get(url, *args, **kwargs):
            if "/resolve" in url:
                m = Mock()
                m.status_code = 200
                m.json = lambda: did_doc
                return m
            raise AssertionError("unexpected url: " + url)

        with patch("aura_sdk.requests.get", side_effect=fake_get):
            res = verify_vc_ldp(vc_with_proof, expected_org_id="00000000-0000-0000-0000-000000000001", expected_owner="alice")
            self.assertTrue(res.get("valid"), res)

        # negative owner mismatch
        with patch("aura_sdk.requests.get", side_effect=fake_get):
            res2 = verify_vc_ldp(vc_with_proof, expected_org_id="00000000-0000-0000-0000-000000000001", expected_owner="bob")
            self.assertFalse(res2.get("valid"))
            self.assertEqual(res2.get("reason"), "owner_mismatch")


if __name__ == "__main__":
    unittest.main()
