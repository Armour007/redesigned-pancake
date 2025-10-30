# SSO & SCIM (Enterprise)

This document outlines how to integrate with enterprise identity providers and automate user lifecycle.

## SSO (OIDC) — Okta / Google / Azure AD

- Enable: set `AURA_SSO_ENABLE=1`.
- Configure OIDC for your provider and map callback to: `https://<backend>/sso/<provider>/callback`.
- Environment (example):
  - `OIDC_ISSUER` (e.g., Okta/Google/Azure issuer URL)
  - `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`
  - `OIDC_REDIRECT_BASE` (frontend URL for post-login handoff)
- Flow: client hits `/sso/<provider>/login` → redirect to IdP → callback validates ID token → server mints app JWT → redirect to frontend with token.
- Org mapping: based on configured domain-to-org map or IdP Group → Org mapping (to be configured per tenant).

Note: The current build includes stub endpoints to help wire your IdP. Full OIDC token exchange can be enabled in a subsequent patch.

## SCIM 2.0 — User provisioning

- Enable: set `AURA_SCIM_ENABLE=1` and issue a provisioning token in `AURA_SCIM_TOKEN`.
- Endpoints:
  - `GET /scim/v2/Users?orgId=<uuid>`
  - `POST /scim/v2/Users?orgId=<uuid>` — creates or updates a user and ensures org membership.
  - `GET /scim/v2/Groups` — placeholder.
- Auth: `Authorization: Bearer <AURA_SCIM_TOKEN>`
- Default role: `AURA_SCIM_DEFAULT_ROLE` (defaults to `member`).

## RBAC roles

- Roles: owner, admin, auditor, read-only, member
- Mutations require admin/owner; read-only, auditor, member are restricted to GET/HEAD.
- Apply fine-grained controls per endpoint as needed; default guard is applied to organization routes.
