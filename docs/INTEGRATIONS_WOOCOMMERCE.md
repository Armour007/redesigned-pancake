# WooCommerce Integration (Aura Verify)

Make WooCommerce actions safer by verifying them through Aura in real time.

## Install (Dev)
- Copy `integrations/woocommerce/` to your WordPress site under `wp-content/plugins/aura-verify/`.
- Activate the plugin. Open "Aura Verify" in admin.
- Configure:
  - API Key (from AURA)
  - Agent ID (you created in your org)
  - API Base: http://localhost:8081 (default)

## What it does
- Hooks on order status change to `processing` or `completed` and calls POST `/v1/verify` with:
  - headers: `X-API-Key`, `AURA-Version`
  - body: `{ agent_id, request_context: { action: 'woocommerce.order.update', env: 'prod', order: {...} } }`
- Adds an order note indicating ALLOWED/DENIED.

## Customize
- Change which events to verify (e.g., payment complete, refund, coupon changes)
- Make policy configurable: block vs warn on DENIED
- Add an admin page to map Woo events → Aura actions

## Next steps
- Add webhook verification helpers (see `sdks/node/src/webhook.ts`, `sdks/python/aura_webhook.py`)
- Package for distribution (.zip) and publish in a separate repo
