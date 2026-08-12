# healthy_catering — Document Set

**Version:** 0.1 (scaffolded)
**Date:** 12 August 2026
**Status:** repo and conventions in place. **The domain is not defined yet** —
Steven supplies the product brief in a later prompt.

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
| 14 | `14-production-deployment-handbook.md` | Empty-machine deployment, copy-paste | ⬜ |
| 15 | `15-user-guide.md` | Customer guide | ⬜ |
| 16 | `16-admin-guide.md` | Staff guide | ⬜ |
| 99 | `99-steven-preference.md` | Portable engineering DNA — **project-agnostic** | ✅ |

Plus `PROGRESS.md` (live build status) and `RUN-WHEN-BACK.md` (interactive steps).

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

---

## 3. Open questions

Answers go here as they land; each one that changes behaviour becomes a decision.

| # | Question | Default if unanswered |
|---|---|---|
| Q1 | **What is the product?** Domain, users, the job it does. | — blocking, no default |
| Q2 | Is **Evermore** the customer-facing name, or an internal brand for `healthy_catering`? | Evermore is the customer-facing brand |
| Q3 | Currency, and is it a zero-decimal currency? | — blocking for any money path |
| Q4 | Business timezone | — blocking for any scheduling |
| Q5 | UI languages | ID + EN via message catalogues, as ruuma |
| Q6 | Production domain | — |
| Q7 | Can we have **page 13** of the brand guidelines ("Logo on Color Palette")? It is referenced by the colour page and was not supplied. | proceed without it |
| Q8 | Is there a **reversed-out logo** for dark fills? The supplied mark is dark ink only and disappears on Nourish Green. | derive one, as ruuma did |
| Q9 | **Erode** licence terms for web embedding | confirm before first use |
