# AURA Web (Next.js 14)

Production-ready Next.js 14 + Tailwind app that powers the VC-ready marketing and dashboard experience.

## Quick start

- Requirements: Node 18+ (LTS recommended), npm 9+
- Port: 3000

### Install

Windows PowerShell:

```
# From repo root
cd .\web
npm install
```

### Develop

```
npm run dev
```

Then open http://localhost:3000

### Build

```
npm run build
```

### Start (production)

```
npm run start
```

## Tech

- Next.js 14 (App Router)
- Tailwind CSS
- Framer Motion
- Three.js + @react-three/fiber (optional visuals)

## Project structure

- `app/` — App Router pages (`/`, `/dashboard`, `/onboard`)
- `components/` — UI components (Navbar, Hero, Footer, AgentCard)
- `public/` — Static assets (add `public/logo/*` here)

## Notes

- This package uses `"type": "module"`. Config files using CommonJS must use `.cjs` or ESM `export default`.
- Path alias `@/*` is configured in `tsconfig.json`.
- Tailwind scans `app/**/*` and `components/**/*` by default.
