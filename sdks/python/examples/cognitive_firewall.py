"""Gate function calls with AURA cognitive firewall.
Requires env: AURA_API_BASE_URL, AURA_API_KEY
"""
import os
import json
import urllib.request

BASE = os.environ.get("AURA_API_BASE_URL", "http://localhost:8081")
API_KEY = os.environ.get("AURA_API_KEY", "dev_api_key")


def guard_call(action: str, resource: str, context: dict) -> dict:
    data = json.dumps({
        "action": action,
        "resource": resource,
        "request_context": context or {}
    }).encode()
    req = urllib.request.Request(
        f"{BASE}/v2/guard",
        data=data,
        headers={"Content-Type": "application/json", "X-API-Key": API_KEY},
        method="POST",
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def main():
    action = "tool.execute"
    resource = "shell"
    context = {"user": {"tier": "free"}, "cmd": "curl http://example.com"}
    res = guard_call(action, resource, context)
    print("guard:", res)
    if res.get("status") == "allow":
        print("Proceeding...")
    elif res.get("status") == "needs_approval":
        print("Needs approval:", res.get("hints"))
    else:
        print("Denied:", res.get("reason"))


if __name__ == "__main__":
    main()
