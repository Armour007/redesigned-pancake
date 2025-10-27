# AURA Product Roadmap (High-level)

Principles: simple, Stripe-like DX; universal, language-agnostic; verifiable actions for machines and autonomous systems.

## 0–2 weeks (Foundation DX)
- First-time Quick Start (done): redirect + drawer + masked key + multi-language snippets
- Multi-language SDK codegen scaffold (done); plan signature helpers rollout
- WooCommerce plugin skeleton (done): PoC for “like Stripe in WooCommerce”
- CLI quickstart script (pending): 1 command to create agent → rule → API key → print verify snippet
- VS Code launch (done): set frontend URL env consistently

## 2–4 weeks (Integrations and polish)
- Signature helpers in official SDKs (Node, Python, Go) and codegen languages
- Example integrations: LangChain/Autogen (agent verify-before-act), Airflow operator, Terraform provider
- Dashboard: keys/rules UI; auto-open Quick Start after creation
- Frontend polish: landing hero, features, dark mode; sections inspired by Stripe and GitHub Universe

## 1–3 months (Scale and reliability)
- Rate limiting + Redis by default, per-org quotas
- Webhook delivery retries, DLQ, signed events
- Metrics dashboards (Grafana), tracing defaults
- Packaging: Helm chart, one-line cloud deploy
- Registry publishing: Maven, NuGet, crates.io, PyPI, npm, etc.

## North Star
- AURA becomes the universal “policy and verification” layer for actions between services and agents—fast to adopt, hard to operate without.
