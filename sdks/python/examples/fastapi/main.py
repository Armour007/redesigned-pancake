from fastapi import FastAPI, Request, Header, HTTPException
from aura_sdk import AuraClient, verify_signature
import os
import json

app = FastAPI()

@app.post("/webhooks/aura")
async def webhook(request: Request, aura_signature: str = Header(None)):
    secret = os.environ.get("AURA_WEBHOOK_SECRET", "")
    body = await request.body()
    if not verify_signature(secret, aura_signature, body):
        raise HTTPException(status_code=401, detail="invalid signature")
    event = json.loads(body)
    print("AURA event", event.get("type"))
    return {"ok": True}

@app.get("/verify")
async def verify():
    client = AuraClient(api_key=os.environ.get("AURA_API_KEY"), base_url=os.environ.get("AURA_API_BASE"), version=os.environ.get("AURA_VERSION"))
    resp = client.verify(os.environ.get("AURA_AGENT_ID"), {"action": "deploy:prod", "branch": "main"})
    return resp
