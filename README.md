# healthy_catering

Brand: **Evermore**. Owner: stevenwilliam.

**The domain is not defined yet.** The repo is scaffolded to the house style and
the brand assets are in; the product brief arrives in a later prompt. Nothing
here assumes what the product does.

## Start here

1. `CLAUDE.md` — how this project is built. Read it first, every session.
2. `docs/00-README-and-decisions.md` — index, decision log, open questions.
3. `docs/99-steven-preference.md` — portable engineering DNA, project-agnostic.
4. `docs/10-design-system.md` — the Evermore brand as tokens, **with the
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
