# healthy_catering — Document Set

**Version:** 0.2 (brief received, planning documents written)
**Date:** 12 August 2026
**Status:** the product brief landed on 2026-08-12 and is stored verbatim at
`PROMPT.md`. Evermore is a **B2C healthy-catering ordering website for Jakarta**
(`www.evermore.co.id`). The four planning documents it asks for — `01`
domain model, `02` decisions, `03` open questions, `04` milestones — are written
and **awaiting Steven's confirmation. No application code until he approves.**

---

## 1. What this document set is

The engineering and product spec for `healthy_catering`, built in the house
style. House style itself — how Steven works, and his stack, database and
security preferences — lives in `99-steven-preference.md` and is portable
between projects.

`02-business-rules.md` will be **normative**: where it conflicts with any other
document, it wins. Build and working conventions live in `../CLAUDE.md`.

| # | Document | Purpose | State |
|---|---|---|---|
| 00 | This file | Index, decision log, open questions | ✅ |
| 01 | `01-PRD.md` | Problem, personas, scope, requirements, metrics | ⬜ awaiting brief |
| 02 | `02-business-rules.md` | Normative business logic, `BR-x.y` | ⬜ awaiting brief |
| 03 | `03-data-model.md` | Schema, ERD, DDL, migration notes | ⬜ |
| 04 | `04-api-specification.md` | REST contract, error model, idempotency, auth | ⬜ |
| 05 | `05-architecture-and-nfr.md` | Service architecture, security, performance | ⬜ |
| 06 | `06-domain-operations.md` | Domain-specific operational logic, runbooks | ⬜ |
| 07 | `07-test-plan.md` | Test strategy, critical scenarios, QA checklist | ⬜ |
| 08 | `08-roadmap.md` | Phasing, release plan, sequencing | ⬜ |
| 09 | `09-deployment.md` | Production deployment | ⬜ |
| 10 | `10-design-system.md` | Palette, typography, components, a11y | 🟡 brand received |
| 11 | `11-local-dev-setup.md` | Local dev environment, everyday commands | ⬜ |
| 12 | `12-security.md` | OWASP ASVS L2 / Top-10 control map, abuse cases | ⬜ |
| 13a | `13a-development-server-preparation.md` | Dev-server handbook — Part A (server once) + Part B (onboard a project) | ⬜ |
| 14 | `14-production-deployment-handbook.md` | Empty-machine deployment, copy-paste | ⬜ |
| 15 | `15-user-guide.md` | Customer guide | ⬜ |
| 16 | `16-admin-guide.md` | Staff guide | ⬜ |
| 99 | `99-steven-preference.md` | Portable engineering DNA — **project-agnostic** | ✅ |

Plus `PROGRESS.md` (live build status) and `RUN-WHEN-BACK.md` (interactive steps).

### 1.1 Planning documents — temporary, and they occupy numbers the house set wants

The master brief (`PROMPT.md` §0) asks for four documents under names that
collide with the house set above. Nothing collides on disk today because none of
the house documents exist yet, and **nothing has been renamed unilaterally**.

| File | Purpose | State |
|---|---|---|
| `PROMPT.md` | Steven's master brief, verbatim — the source for all four | ✅ |
| `01-domain-model.md` | Entities, relationships, state machines, Mermaid ERD | ✅ awaiting confirmation |
| `02-decisions.md` | All 23 `[DECIDE]` items + 5 stack conflicts, with recommendations | ✅ awaiting confirmation |
| `03-open-questions.md` | 29 ambiguities, each with the answer needed and a default | ✅ awaiting confirmation |
| `04-milestones.md` | M0–M14, each slice demoable | ✅ awaiting confirmation |

`04-milestones.md` §2 proposes folding these four into the house numbering on
approval — `01` into `03-data-model` and `02-business-rules`, `02` and `03` into
this file, `04` into `08-roadmap` — so `02` is not left meaning two different
things. **Steven's call; see `03-open-questions.md`.**

---

## 2. Decision log

Record every decision that changes behaviour here, with a date, and reflect it
in the affected docs the same day.

| ID | Date | Decision | Rationale | Docs touched |
|----|------|----------|-----------|--------------|
| D1 | 2026-08-12 | **Adopt Steven's house style verbatim** — `99-steven-preference.md` copied unchanged from ruuma, and `CLAUDE.md` generated from its sections 3–9. Hexagonal Go (`adapter → app → domain`), gin + gorm + PostgreSQL, money as integers with raw SQL on money paths, UUIDv7, numbered forward-only migrations, search box on every list, configurable values in `sys_parameters`, docs updated in the same commit, auto-commit and push to `main`. | Proven on ruuma; the preference file is written to be project-agnostic precisely so it can be copied. | CLAUDE.md, 99 |
| D2 | 2026-08-12 | **`coding stop` / `coding start` is a hard gate.** `coding stop` means change nothing — no edits, files, commits, migrations, deploys or config — until `coding start`. It holds across turns, and a new request while it is on is a request to plan, not a licence to resume. | Steven's directive, 2026-08-12. | CLAUDE.md §6, 99 §1 |
| D3 | 2026-08-12 | **Brand is Evermore**, from the guidelines supplied at `/home/aidev/asset/` and copied to `docs/design_guideline/`. Primary Nourish Green `#1C3D34`, secondary Restore Beige `#FFFAE0`, four tertiary accents; display **Erode**, body **Inter**, both self-hosted. **Four brand colours fail WCAG AA as text or button fills** — `#468973`, `#E0782D`, `#A36E50`, `#CC6883` — so `#1C3D34` carries green text and green fills, and orange is a highlight rather than a label. Calculated, not eyeballed, before any component exists. | Supplied brand; contrast checked on receipt so the build is not designed into a corner. | 10, design_guideline |
| D4 | 2026-08-12 | **Assets were copied, not moved.** Steven asked for a move; the source is another user's home (`aidev`) and a move there is not reversible by this project. The originals remain at `/home/aidev/asset/`. | Reversible beats literal when the difference is destroying someone else's copy. Flagged for Steven to confirm. | 00, design_guideline |
| D5 | 2026-08-12 | **Dark-surface and non-text contrast decided up front.** On `#1C3D34`: beige 11.32, blue light 8.15, orange light 7.27, beige deep 6.47 all pass; `#468973` (2.88) and `#CC6883` (3.33) are never text on it. `#CCBDAA` on beige is **1.75**, so it is *not* an input border, focus ring or any meaningful boundary — those use `#1C3D34`. Energize Orange as a button needs near-black ink at large size (`#1C3D34` on it = 3.90, `#000000` = 6.89), never white. Logo re-verified by decoding the PNG: 7582×1989, ink `rgba(28,61,52)`, **only the final `e` is mirrored** — both `r`s are normal. | The header is going to be Nourish Green and the canvas is going to be beige; deciding the legal inks and borders now costs one calculation, and rediscovering them in review costs a redesign. Closes the "verify before shipping" TODO left on the orange button. | 10 §1, §2.4–2.6 |
| D6 | 2026-08-12 | **Product brief received and stored verbatim at `PROMPT.md`.** Evermore is a B2C healthy-catering ordering website for Jakarta, `www.evermore.co.id`: marketing + menu pages, customer accounts, à-la-carte meal orders and prepaid credit packages, manual bank transfer, automatic per-delivery routing to one of several kitchens by coordinates, full back office. Phase 1 web only, no PWA; phase 2 mobile against the same `/api/v1`. Planning documents `01`–`04` written; **no code until Steven confirms them**. | Step 2 of the delivery workflow (`CLAUDE.md` §9) is complete; step 3 is gated on his confirmation. | CLAUDE.md §1 §10, 00, 01–04, PROGRESS |
| D7 | 2026-08-12 | **Locale settled: IDR as `BIGINT` whole rupiah, `Asia/Jakarta`, `id-ID` + `en`, `www.evermore.co.id`, Jakarta hosting for UU PDP residency. Evermore is the customer-facing brand**; `healthy_catering` is the repo codename. Answers `00` §3 Q2–Q6. | Stated in the brief. Sen is obsolete in retail so the rupiah is the minor unit; business dates are `DATE` and convert through `Asia/Jakarta` explicitly, never server-local. | CLAUDE.md §10, 00 §3, 01 |
| D8 | 2026-08-12 | **The stack is contested and unresolved.** `CLAUDE.md` §3 pins Go + gin + gorm + React 18/Vite; the brief §2 proposes TypeScript + NestJS + Prisma + Next.js and marks it `[DECIDE]`. Five sub-decisions (`02-decisions.md` D-1…D-5) are open and **block M0**. Recommendation: Go + gin + gorm, numbered SQL migrations, Redis as a satellite, and Go templates for the eight public routes beside a React 18 + Vite SPA. | Prisma has no first-class support for `daterange`, exclusion constraints or PostGIS — the three things this product's hardest parts need — so the ORM's benefit disappears exactly where it would be paid for. Recorded as contested rather than silently resolved either way. | 02, CLAUDE.md §3 (pending) |

---

## 3. Open questions

Answers go here as they land; each one that changes behaviour becomes a decision.

**Q1–Q6 are answered** by the brief of 2026-08-12 (D6, D7). The product questions
it raised in turn now live in **`03-open-questions.md`** — 29 of them, each with
the concrete answer needed and a default — and the choices it left open live in
**`02-decisions.md`**. This table keeps only what is still open here.

| # | Question | Status / default if unanswered |
|---|---|---|
| Q1 | **What is the product?** | ✅ answered — B2C healthy catering, Jakarta (D6) |
| Q2 | Is **Evermore** the customer-facing name? | ✅ yes; `healthy_catering` is the repo codename (D7) |
| Q3 | Currency | ✅ IDR, `BIGINT` whole rupiah (D7) |
| Q4 | Business timezone | ✅ `Asia/Jakarta` (D7) |
| Q5 | UI languages | ✅ `id-ID` default + `en` (D7) |
| Q6 | Production domain | ✅ `www.evermore.co.id` (D7) |
| Q7 | Can we have **page 13** of the brand guidelines ("Logo on Color Palette")? It is referenced by the colour page and was not supplied. | still open — proceed without it |
| Q8 | Is there a **reversed-out logo** for dark fills? The supplied mark is dark ink only and disappears on Nourish Green. | still open — derive one, as ruuma did. Now urgent: the brand's green header is on every page of M12 |
| Q9 | **Erode** licence terms for web embedding | still open — confirm before first use |
| Q10 | **Which stack?** `CLAUDE.md` §3 (Go) vs the brief §2 (NestJS/Next.js) | — blocking M0, no default. See `02-decisions.md` D-1…D-5 |
| Q11 | Do the four planning documents fold into the house numbering on approval, or keep their names? | fold, per `04-milestones.md` §2 |
