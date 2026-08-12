# healthy_catering — engineering & product DNA

This file is the contract for how this project is built. Read it first, every
session, before touching code or docs. Where it conflicts with a habit, this
file wins. Where it conflicts with `docs/02-business-rules.md` on *product*
logic, that document wins — this file governs *how* we build, not *what* the
product does.

It is generated from `docs/99-steven-preference.md`, which is Steven's portable
engineering DNA. **That file is the source; this one is the local application of
it.** When the two disagree, this file wins — it is the newer, more specific
decision.

---

## 1. What this is

**Codename:** healthy_catering
**Brand:** Evermore (see §7 and `docs/design_guideline/`)
**Owner:** stevenwilliam (itdept.sfg@gmail.com)
**Repo:** https://github.com/stevenwilliam/healthy_catering
**Status:** scaffolded. **The domain is not defined yet** — Steven supplies the
product brief in a later prompt. Nothing below assumes what the product does
beyond the name.

**Do not invent the domain.** Until the brief lands, do not create business
rules, a data model, or endpoints. Ask, or wait.

---

## 2. Architecture — non-negotiable

Hexagonal / clean layering. Dependencies point **inward only**:
`adapter → app → domain`, with `platform` available to all. `domain` imports no
framework, no driver, no `net/http`, no SQL.

```
cmd/api/main.go            # thin entrypoint: wire + run subcommands (serve, migrate, seed)
internal/
  domain/                  # pure business logic + types; exhaustively unit-tested; no I/O
  app/                     # use-cases / services; orchestrates domain + ports
  adapter/
    http/                  #   gin handlers, request/response mapping
    postgres/              #   gorm repositories (raw SQL on money paths)
    storage/               #   S3 / MinIO
    notify/                #   email / outbound
  platform/                # cross-cutting infra, business-agnostic, reusable across projects
    config/ logging/ metrics/ apierror/ id/ security/ ratelimit/ database/
db/
  migrations/NNNN_name.up.sql + NNNN_name.down.sql
  embed.go                 # go:embed migrations
web/                       # React SPA
```

`internal/platform/*` is meant to be **portable**. Carry it over from ruuma
(`/home/dev/projects/ruuma/internal/platform/`) and adapt rather than reinvent —
those shapes are proven.

---

## 3. Stack

Backend: **Go (latest)** · **`gin`** · **`gorm`** + `gorm.io/driver/postgres` ·
**PostgreSQL (latest major)** · `golang-jwt/jwt/v5` · `google/uuid` ·
S3/MinIO (`minio-go/v7`) · Prometheus (`client_golang`) · `golang.org/x/crypto`.

Frontend: **React 18** + **Vite** + **TypeScript** + **Tailwind**,
`web/src/{components,lib,pages}`. Pin React to 18, not 19. Node 20.

ORM is `gorm`. **Exception:** any code path touching money uses explicit
`gorm.Exec`/raw SQL with integer arithmetic — never the ORM for money math.

Not defaults, do not reach for them unprompted: automigrate as the source of
truth, GraphQL, microservices, Kubernetes, a NoSQL primary store, SSR
frameworks, CSS-in-JS.

---

## 4. Hard rules

- **Money is integers.** Store the minor unit as `BIGINT` and do all arithmetic
  in integers. Floating point is prohibited in any code path touching money.
  Percentages are basis points, rounded half-up:
  `floor((amount * bps + 5000) / 10000)`.
- **IDs are UUIDv7.** Human-facing codes use CSPRNG + Crockford base32.
- **The domain layer is pure and exhaustively unit-tested.** Adapters get
  integration tests.
- **Migrations are forward-only in production**, numbered, each with a matching
  `.down.sql`, embedded via `go:embed`. The migrations are the source of truth.
- **The database enforces the invariant**, not just the application — foreign
  keys, `NOT NULL`, `CHECK`, partial and unique indexes.
- **Concurrency is tested, not assumed.** Anything reserving a limited resource
  takes `SELECT … FOR UPDATE` inside one transaction and ships with a test that
  proves it cannot oversell.
- **Timestamps are `timestamptz` in UTC.** Business-day logic converts to the
  operating timezone explicitly — never server-local.
- **Errors are typed** through `platform/apierror`; one JSON error model. Never
  leak driver errors to clients.
- **Secrets only via config/env.** Nothing secret in git. `.env.example` is the
  documented surface; the real `.env` is git-ignored.

---

## 5. Docs discipline

- Docs live in `docs/`, numbered. `docs/02-business-rules.md` is **normative** —
  rules carry `BR-x.y` IDs and code/tests reference those IDs.
- **Keep all docs in sync on every decision**, in the same commit as the change.
  A decision that isn't in the docs didn't happen.
- `docs/PROGRESS.md` is live build status (✅ done & tested · 🟡 partial ·
  ⬜ not started). A ✅ has to be re-earned by running the gate, never inherited.
- `docs/RUN-WHEN-BACK.md` holds steps needing an interactive terminal.
- `docs/99-steven-preference.md` is portable and project-agnostic. Improvements
  that are not specific to this project belong there.

---

## 6. Working conventions

- **Owner is Steven, nickname "ven".** When he answers a quoted list of
  questions, a line beginning `ven:` is his answer to the question above it.
- **`coding stop` means change nothing** — no edits, no new files, no commits,
  no migrations, no deploys, no config changes — until he says `coding start`.
  It is a hard gate, not a preference to weigh against the task, and it **holds
  across turns** until lifted. A new request while the hold is on is a request
  to discuss and plan, not a licence to resume: say what you would do, then
  wait. Reading, searching, read-only commands, answering and planning stay
  fine; what stops is anything that writes — the filesystem, the database, a
  running service, or a remote. If unsure whether the hold is on, it is.
- **Ask everything at once, up front, with a default per question.** One batch
  before starting, not a drip of questions mid-build.
- **Never stop partway.** If the plan says "build all modules A–Z", build all of
  them in one push.
- **Update related documents on every interaction** — including talk-only turns
  that settle a decision.
- **Auto-commit + push after every completed change**, without asking. Small,
  focused commits, conventional-commit messages. `main` is the working branch.
- **Tell the truth about what was verified.** If a test did not run, say so and
  put the step in `RUN-WHEN-BACK.md`. Never report "done and tested" for
  something only written.
- **Verify visual work by looking at it.** Screenshot the rendered page; do not
  conclude from reading CSS. (Learned expensively on ruuma.)
- **Editor is `vi`** in every runbook and docs example — never `nano`.
- **OS/server guides use full absolute paths**, never relative ones.
- Prefer editing existing files and reusing `platform/*` over new scaffolding.

---

## 7. Product & UI conventions

- **Search box on every list.** Every screen rendering a list or table has a
  search box that filters it. No exceptions.
- **Configurable values live in `sys_parameters`.** Anything that could change
  without a code change — company phone, email, address, tax rate, feature
  toggles, thresholds — is a row in that table, not a constant, and ships with
  full CRUD behind an admin permission.
- **Brand is Evermore.** The supplied guidelines are in
  `docs/design_guideline/`; `docs/10-design-system.md` is the engineering
  reading of them. **Several brand colours fail WCAG AA as text or as button
  fills** — that is documented there and must be respected, not rediscovered.
- **Accessibility is AA minimum**, and contrast is *calculated*, not eyeballed.
- **Every public page ships the SEO baseline** from `99-steven-preference.md`
  §13: per-route title and description, Open Graph tags **static in the served
  HTML** (preview bots do not run JavaScript), `robots.txt` disallowing the
  transactional surface, `sitemap.xml`, one `<h1>` per page, JSON-LD.

---

## 8. Document control

Always update related documents in the same commit as the change — PRD,
business rules, data model, API spec, deployment/user/admin guides. A change
whose docs are stale is not done.

---

## 9. Delivery workflow

1. **Initial git setup** — repo, remotes, conventions, this file. ← done
2. **Steven — preparation.** He gives the PRD and business-rules brief, tuning,
   and final confirmation. **Nothing downstream starts until he confirms.**
3. **Claude — build all documents A→Z** from the confirmed brief.
4. **Claude — build all modules in one shot, A→Z.** Do not stop partway.
5. **Claude — test, debug and security-harden, A→Z.** Do not stop partway.
6. **Claude — production deployment handbook** (copy-paste, empty machine, full
   absolute paths), **then** the user guide, **then** the admin guide.

---

## 10. Locale / environment

**Not yet decided.** Currency, timezone, languages and the production domain
come with the product brief. Until then, do not assume ruuma's answers — that
project is IDR / Asia-Jakarta / ID+EN because it is an Indonesian restaurant,
and this one has not said what it is.
