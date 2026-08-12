# Build progress

Live status. Legend: ✅ done & tested · 🟡 partial · ⬜ not started.

**A ✅ has to be re-earned by running the gate, never inherited from the last
time it passed.** That lesson cost real time on ruuma, where this file claimed a
green quality gate for days after it had started failing.

_Last updated: 2026-08-12._

## M0 — Definition
- ✅ Repo created, `git init`, `main`, remote `git@github.com:stevenwilliam/healthy_catering.git`
- ✅ `CLAUDE.md` generated from the preference file
- ✅ `docs/99-steven-preference.md` carried over verbatim
- ✅ `.gitignore`, `.gitattributes`, `.env.example`, `README.md`
- ✅ Doc set index and decision log started (D1–D4)
- ✅ Brand assets received and read: palette, typefaces, logo (`docs/design_guideline/`)
- ✅ Palette contrast calculated — 4 colours fail AA as text/fills, recorded in `10` §2.4
- ⬜ **Product brief — blocking everything below.** Domain, users, currency,
      timezone, languages, production domain. See `00` §3 Q1–Q6.

## M1 — Documents (step 3 of the workflow)
- 🟡 `10-design-system.md` — brand in, components pending a product
- ⬜ `01` PRD, `02` business rules, `03` data model, `04` API spec
- ⬜ `05` architecture/NFR, `06` domain operations, `07` test plan, `08` roadmap
- ⬜ `09` deployment, `11` local dev, `12` security

## M2 — Build (step 4)
- ⬜ Everything. Do not start before the brief is confirmed (`CLAUDE.md` §9.2).

## M3 — Test & harden (step 5)
- ⬜ Everything.

## M4 — Handbooks (step 6)
- ⬜ `14` deployment handbook, `15` user guide, `16` admin guide.

## Known gaps
- ⬜ **No reversed-out logo.** The supplied mark is dark ink only and vanishes
      on the primary green — needed before the first dark header (`00` Q8).
- ⬜ **Erode licence** not yet confirmed for web embedding (`00` Q9).
- ⬜ **Page 13 of the brand guidelines** ("Logo on Color Palette") was not
      supplied and is referenced by the colour page (`00` Q7).
