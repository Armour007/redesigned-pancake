# WooCommerce Integration (Stripe‑like Ease)

Goal: make Aura as easy to integrate as Stripe in WooCommerce. The plugin is distributed separately; this repo only documents how to use it.

## What you install
- Aura Verify for WooCommerce (separate plugin repository/distribution)
  - Configure: API Key (from Aura), Agent ID, API Base (e.g., http://localhost:8081 for local)
  - Hooks into WooCommerce events (e.g., order status changes) and calls Aura `/v1/verify` with rich context.

## How it works
- On chosen Woo events, the plugin sends:
  - Headers: `X-API-Key`, `AURA-Version`
  - Body: `{ agent_id, request_context: { action: 'woocommerce.order.update', env: 'prod', order: {...} } }`
- The response decision (ALLOWED/DENIED) is recorded (e.g., as an order note). Policies can be set to block or warn.

## Why separate plugin?
- To keep Aura’s core clean and universal, WooCommerce is shipped as an independent plugin (like Stripe). This repo won’t contain Woo code.

## Related
- Webhook signature helpers: `sdks/node/src/webhook.ts`, `sdks/python/aura_webhook.py`
- Roadmap includes packaging and publishing the plugin to a dedicated repository/zip distribution.
