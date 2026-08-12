# healthy_catering

Brand: **Evermore**. Owner: stevenwilliam.

A **B2C healthy-catering ordering website for Jakarta** — `www.evermore.co.id`.
Marketing and menu pages, customer accounts, à-la-carte meal orders and prepaid
meal-credit packages, manual bank transfer, automatic routing of every delivery
to one of several kitchens by address coordinates, and a full staff back office.
Phase 1 is web only; phase 2 is native mobile against the same `/api/v1`.

**Status: planning.** The brief landed 2026-08-12 and the four planning
documents are written and awaiting confirmation. **No application code yet** —
the stack itself is still contested (see `docs/02-decisions.md` D-1…D-5).

## Start here

1. `CLAUDE.md` — how this project is built. Read it first, every session.
2. `docs/PROMPT.md` — Steven's master brief, verbatim. The source of truth.
3. `docs/02-decisions.md` — 23 open decisions. **D-1…D-5 block everything.**
4. `docs/01-domain-model.md` — entities, ERDs, state machines.
5. `docs/03-open-questions.md` · `docs/04-milestones.md`
6. `docs/00-README-and-decisions.md` — index and decision log.
7. `docs/99-steven-preference.md` — portable engineering DNA, project-agnostic.
8. `docs/10-design-system.md` — the Evermore brand as tokens, **with the
   contrast maths done**. Read it before using a colour: four brand colours do
   not reach WCAG AA as text or button fills.

## Layout

```
cmd/api/          thin entrypoint — serve, migrate, seed
internal/
  domain/         pure business logic, no I/O
  app/            use-cases, orchestrating domain + ports
  adapter/        http, postgres, storage, notify
  platform/       cross-cutting infra, portable between projects
db/migrations/    numbered SQL, forward-only, each with a .down.sql
web/              React 18 + Vite + TypeScript + Tailwind
docs/             numbered document set
test/             integration, security, e2e
```

`internal/platform/*` should be carried over from
`/home/dev/projects/ruuma/internal/platform/` and adapted, not reinvented.
