# Build progress

Live status. Legend: ✅ done & tested · 🟡 partial · ⬜ not started.

**A ✅ has to be re-earned by running the gate, never inherited from the last
time it passed.** That lesson cost real time on ruuma, where this file claimed a
green quality gate for days after it had started failing.

_Last updated: 2026-08-12, during the first build session._

**Everything marked ✅ below was run, not just written.** Where something was
written but not executed, it is 🟡 and says what is missing.

## M0 — Definition
- ✅ Repo created, `git init`, `main`, remote `git@github.com:stevenwilliam/healthy_catering.git`
- ✅ `CLAUDE.md` generated from the preference file, product and locale filled in
- ✅ `docs/99-steven-preference.md` carried over verbatim
- ✅ `.gitignore`, `.gitattributes`, `.env.example`, `README.md`
- ✅ Doc set index and decision log (D1–D10)
- ✅ Brand assets received, read and re-verified against the source PNGs
- ✅ Palette contrast calculated; dark-surface and border rules decided (`10` §2.4–2.6)
- ✅ **Product brief received** and stored verbatim at `docs/PROMPT.md`

## M0b — Planning documents
- ✅ `01-domain-model.md`, `02-decisions.md`, `03-open-questions.md`, `04-milestones.md`
- ✅ Steven's answers folded in: tax-inclusive pricing, no refunds, the meal as
      the unit of sale, nearest-kitchen routing (D-30…D-34)
- 🟡 **Not formally confirmed.** Steven said "code start" without walking the
      decision list, so `02-decisions.md` D-1…D-5 (the stack) were taken at
      their recommendation and recorded as *decided by default* (D10). If he
      disagrees on any of them, say so before M2 grows.

## M1 — Environment (this session)
- ✅ PostgreSQL 18.4: `healthy_catering` and `healthy_catering_test` created,
      owned by the `healthy_catering` role
- ✅ `postgresql-18-postgis-3` installed; `postgis` 3.6.4, `btree_gist`, `citext`
      created in both databases
- ✅ Redis 7 satellite container `redis-shared` running on 127.0.0.1:6379
- ✅ `.env` written (git-ignored, 0600) with generated JWT and TOTP keys
- ⬜ MinIO bucket for Evermore — ruuma's container is up but has no `evermore` bucket
- ⬜ nginx site, TLS, DNS for `evermore.co.id`

## M2 — Schema (milestone M0 of `04-milestones.md`)
- ✅ Migrations `0001`–`0011`, each with a matching `.down.sql`, `go:embed`ed
- ✅ **Up applied, fully rolled back, and re-applied** on a clean database
- ✅ Constraints proved by direct SQL, not assumed:
      overlapping price rows rejected · adjacent ranges accepted · cross-scope
      overlap allowed · overlapping tiers rejected · `qty_reserved > qty_capacity`
      rejected · a 15-minute-grid violation rejected · `credit_ledger` refuses
      `UPDATE` and `DELETE` · a positive `EXPIRE` rejected · an `ADJUSTMENT`
      with no reason rejected
- ✅ Reference data seeded: 6 roles, 28 permissions, 4 customer types, 5 diet
      types, 9 allergens, 2 active slots, 4 tiers, 3 packages, 28 parameters,
      2 **placeholder** kitchens

## M3 — Domain layer
- ✅ `money` (92% coverage) — integer rupiah, the D-30 tax split
- ✅ `pricing` (94%) — scope→DEFAULT fallback, promo within scope, flat tiers
- ✅ `credit` (97%) — 1 credit = 1 meal, append-only rules, expiry, extensions
- ✅ `routing` (95%) — nearest kitchen, polygon over radius, deterministic
- ✅ `nutrition` (95%) — meal panel aggregated from foods
- ✅ `schedule` (91%) — cut-off in Asia/Jakarta, deadline capped at cut-off
- ✅ `order` (94%) — state machines, totals
- ✅ `go test ./...` green; `gofmt` clean

## M4 — Service (milestone M3 of `04-milestones.md`, partial)
- ✅ `cmd/api`: `serve` · `migrate up|down|status` · `version`, graceful shutdown
- ✅ HTTP middleware: request id, structured logging (route template, never the
      path — PII), recovery, CSP without `unsafe-inline`, HSTS, CORS allowlist
- ✅ One error model; a driver error never reaches a client (verified: the
      failing candidate query returned a generic 500 with a request id)
- ✅ `platform/sysparam` — typed reads, 30s cache, invalidation, secret masking
- ✅ **`POST /api/v1/delivery-area/check` works against live data**: Kemang and
      Menteng resolve to their kitchens, Setiabudi picks the nearer at 4.8 km,
      Bogor is refused and logged with its nearest kitchen at 37 km
- ✅ `GET /api/v1/delivery-slots`, `GET /healthz`, `GET /metrics`
- ⬜ Everything else: auth, catalogue, calendar, prices, cart, orders, payments,
      packages, deliveries, reports, notifications, back office, public site

## M5 — Test & harden
- 🟡 Domain unit tests green. **No integration, security or E2E tests yet** —
      no IDOR suite, no negative-authz suite, no concurrency test against the
      real database, no Playwright.

## M6 — Handbooks
- ⬜ `14` deployment handbook, `15` user guide, `16` admin guide.

## Blocked, and blocking

| What | Blocks | Source |
|---|---|---|
| **The real kitchens** — addresses, pins, radii, capacities | Verifying routing; the homepage widget answers from placeholder kitchens today | `03` Q-4 |
| **PPN rate, PKP status, NPWP** | The first real invoice; the engine is built and reads a parameter seeded at 11% | `03` Q-1a |
| **Bank account details** | Payment instructions — the seeded account is `PT EVERMORE PLACEHOLDER` | `03` Q-19 |
| **Google Maps API keys** | The address form's pin picker and the coverage map | `03` Q-22 |
| **SMTP relay + SPF/DKIM/DMARC** | All transactional email | `03` Q-21 |
| **Domain and production host** | Deployment; `evermore.co.id` does not resolve here | `03` Q-24 |
| **Delivery fee bands** | Real fees — seeded figures are invented | `03` Q-3 |
| **Reversed-out logo, Erode licence** | The first dark header and any public page | `00` Q8, Q9 |

## Known gaps carried forward
- ⬜ No reversed-out logo; the mark vanishes on the primary green.
- ⬜ Erode licence not confirmed for web embedding.
- ⬜ Page 13 of the brand guidelines never supplied.
