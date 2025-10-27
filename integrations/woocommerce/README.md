# Aura Verify WooCommerce Plugin (Skeleton)

This WordPress plugin demonstrates how to gate WooCommerce actions via Aura.

## Install (Dev)
1. Copy this folder to your WordPress site under `wp-content/plugins/aura-verify/`.
2. Activate the plugin in the WordPress admin.
3. Open “Aura Verify” in the admin menu and set:
   - API Key (from AURA)
   - Agent ID (for your org/agent)
   - API Base (e.g., http://localhost:8081)

## What it does
- Hooks on order status changes to `processing` or `completed` and calls `/v1/verify`.
- Sends context like action name and order details.
- Adds an order note whether Aura allowed/denied the action.

## Notes
- This is a minimal skeleton for demonstration. For production:
  - Add granular policy controls (which events to verify, block vs warn)
  - Handle timeouts and retries
  - Add webhook signature verification for inbound events (if used)
  - Support multi-env (dev/staging/prod)
