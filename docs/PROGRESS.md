# Build progress

Live status. Legend: ✅ done & tested · 🟡 partial · ⬜ not started.

**A ✅ has to be re-earned by running the gate, never inherited from the last
time it passed.** That lesson cost real time on ruuma, where this file claimed a
green quality gate for days after it had started failing.

_Last updated: 2026-08-12 (product brief received; planning documents written)._

## M0 — Definition
- ✅ Repo created, `git init`, `main`, remote `git@github.com:stevenwilliam/healthy_catering.git`
- ✅ `CLAUDE.md` generated from the preference file
- ✅ `docs/99-steven-preference.md` carried over verbatim
- ✅ `.gitignore`, `.gitattributes`, `.env.example`, `README.md`
- ✅ Doc set index and decision log started (D1–D4)
- ✅ Brand assets received and read: palette, typefaces, logo (`docs/design_guideline/`)
- ✅ Palette contrast calculated — 4 colours fail AA as text/fills, recorded in `10` §2.4
- ✅ Re-verified 2026-08-12 against the source PNGs: all 12 hexes match the
      palette page, every §2.4 ratio reproduces, logo ink sampled `#1C3D34`
      (7582×1989 RGBA), only the final `e` mirrored
- ✅ Dark-surface inks and non-text/border contrast decided (`10` §2.5–2.6, D5)
- ✅ **Product brief received** 2026-08-12, stored verbatim at `docs/PROMPT.md`.
      B2C healthy catering, Jakarta, `www.evermore.co.id`. Locale settled (D7).

## M0b — Planning documents (the brief's first deliverable)
- ✅ `PROMPT.md` — the brief, verbatim
- ✅ `01-domain-model.md` — entities, 4 Mermaid ERDs, 4 state machines, the three
      algorithms that get near-100% coverage
- ✅ `02-decisions.md` — 23 decisions: 18 `[DECIDE]` items from the brief, 5 stack
      conflicts with `CLAUDE.md`, plus 6 forced by the modelling
- ✅ `03-open-questions.md` — 29 questions, each with a default; 3 have none
- ✅ `04-milestones.md` — M0–M14, each demoable, ~11–15 weeks
- 🟡 **Awaiting Steven's confirmation.** Nothing below starts until he approves
      (`PROMPT.md` §0.2, `CLAUDE.md` §9.2).

## M1 — Documents (step 3 of the workflow)
- 🟡 `10-design-system.md` — brand in, components pending approval
- ⬜ `01` PRD, `02` business rules, `03` data model, `04` API spec
- ⬜ `05` architecture/NFR, `06` domain operations, `07` test plan, `08` roadmap
- ⬜ `09` deployment, `11` local dev, `12` security, `13a` dev-server prep

## M2 — Build (step 4) — M0–M12 of `04-milestones.md`
- ⬜ Everything. **Blocked on the stack decision** (`02-decisions.md` D-1…D-5)
      and on Q-1 (tax), Q-4 (real kitchens), Q-13 (refund policy).

## M3 — Test & harden (step 5) — M13
- ⬜ Everything.

## M4 — Handbooks (step 6) — M14
- ⬜ `14` deployment handbook, `15` user guide, `16` admin guide.

## Known gaps
- ⬜ **No reversed-out logo.** The supplied mark is dark ink only and vanishes
      on the primary green — needed before the first dark header (`00` Q8).
- ⬜ **Erode licence** not yet confirmed for web embedding (`00` Q9).
- ⬜ **Page 13 of the brand guidelines** ("Logo on Color Palette") was not
      supplied and is referenced by the colour page (`00` Q7).
