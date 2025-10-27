# Content migration: blog and case studies to Markdown

This outlines a safe, incremental path to move the marketing content from TS arrays to Markdown with frontmatter, using mdsvex. Keeps URLs stable and allows non-dev edits.

## Goals
- Author posts and case studies in Markdown (.md or .svx)
- Keep /blog and /customers routes working with pagination and [slug]
- Preserve current fields: title, date, description, tags, hero image, metrics

## Approach A: mdsvex (Markdown in SvelteKit)

1) Install deps (add to package.json)
- devDependencies: mdsvex, gray-matter, remark-gfm, rehype-slug

2) Configure mdsvex
- Update `svelte.config.js`:
  - import { mdsvex } from 'mdsvex'
  - add extensions: ['.svelte', '.md', '.svx']
  - mdsvex({ remarkPlugins: [remarkGfm], rehypePlugins: [rehypeSlug] })

3) Create content folders
- `frontend/src/content/blog/*.md`
- `frontend/src/content/case-studies/*.md`

4) Authoring format
- Frontmatter example (blog):
```
---
slug: tracing-for-developers
title: Tracing for developers
date: 2025-10-15
description: Practical guide to distributed tracing with OTEL
tags: [observability, tracing]
hero: /images/tracing-hero.jpg
---

Your Markdown content with images and code blocks.
```
- Frontmatter example (case study):
```
---
slug: acme-fraud-reduction
title: Acme reduced fraud 38%
logo: /logos/acme.svg
industry: E-commerce
metrics:
  - label: Chargebacks
    value: -38%
  - label: Approval rate
    value: +7.2%
quote:
  text: Aura let us ship verification in days.
  author: Jane Doe, VP Risk @ Acme
---

Story content...
```

5) Content loaders
- Blog index and slug pages:
  - Use `import.meta.glob('/src/content/blog/*.md', { eager: true })`
  - Map each module's `metadata` and `default` (component) for SSR
  - Sort by date; generate derived `slug`
- Case studies index and slug pages similarly with `/src/content/case-studies/*.md`

6) Routes updates (minimal)
- Replace current TS arrays lookups with the globbed content collections
- Keep `load()` returning the same shape consumed by the Svelte pages

7) Images
- Place reusable assets in `static/images/*` or `static/logos/*`
- Link via absolute paths `/images/...` or `/logos/...`

8) Validation
- Add a tiny content check in `npm run build` (optional) that fails on missing required frontmatter fields

## Approach B: Headless CMS (Contentful/Sanity/Prismic)

- Pros: Non-technical authors, preview workflows, media management
- Cons: Adds infra, authentication, API calls, migration effort
- Sketch:
  - Define Post and CaseStudy types
  - Add SDK client and a `CONTENT_ENABLED` feature flag
  - Implement a fetch layer in +page.ts with caching (edge/CDN or server memory)
  - Map CMS records to the same view models used by pages

## Rollout plan
- Phase 1: Add mdsvex config; migrate one blog post and one case study; keep TS arrays as fallback
- Phase 2: Migrate all content and remove arrays
- Phase 3 (optional): Introduce CMS if needed

## Notes
- Keep slugs stable to preserve SEO
- Ensure OG meta and sitemap include new content
- Use a link checker (CI) to catch broken links
