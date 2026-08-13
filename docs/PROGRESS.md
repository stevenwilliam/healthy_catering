# Build progress

Live status. Legend: ✅ done & tested · 🟡 partial · ⬜ not started.

**A ✅ has to be re-earned by running the gate, never inherited.**
Everything marked ✅ below was **run**, not just written. Where something was
written but not executed, it is 🟡 and says what is missing.

_Last updated: 2026-08-13, end of the first build session._

---

## Deployed right now

| | |
|---|---|
| URL | **http://192.168.88.101:8090** (nginx → 127.0.0.1:8081) |
| Service | `systemd` unit `evermore`, enabled, survives restart |
| API bind | `127.0.0.1` only — **verified unreachable from the LAN** |
| Secrets | `/etc/evermore/evermore.env`, `root:dev`, mode 640 |
| Database | PostgreSQL 18.4 + PostGIS 3.6, 13 migrations applied |
| ruuma | untouched — still owns `:80 default_server` |

## M0 — Definition ✅
Repo, `CLAUDE.md`, brand read and contrast calculated, brief received and
stored at `PROMPT.md`, locale settled.

## M0b — Planning documents ✅
`01` domain model · `02` decisions (34) · `03` open questions · `04` milestones.
All of Steven's answers folded in (D-30…D-34, D13–D16).

## M1 — Environment ✅
PostgreSQL 18 + PostGIS + btree_gist + citext, Redis satellite, `.env`,
self-hosted Erode and Inter with their licences.

## M2 — Schema ✅
Migrations 0001–0013, up **and** down, applied and re-applied on a clean
database. Constraints proved by direct SQL, not assumed: price overlap,
capacity oversell, append-only ledger and audit, sign checks, the 15-minute
grid.

## M3 — Domain layer ✅
`money` 92% · `pricing` 94% · `credit` 97% · `routing` 95% · `nutrition` 95% ·
`schedule` 91% · `order` 94% · `sanitize` · all pure, no I/O.

## M4 — Identity, RBAC, audit ✅
Registration, login, refresh rotation, deny-by-default permissions resolved per
request from the database, audit log, `api create-staff` first-run flow.
🟡 **TOTP enrolment not built** (schema and per-role requirement exist).

## M5 — Master data & settings ✅
Customer types, diet types, allergens, slots, organisations, `sys_parameters`
CRUD with type checking, secret masking and full audit. Search on every list.

## M6 — Catalogue & menu calendar ✅
Foods with nutrition, meals composed of dishes, aggregation verified
(320+80+60 = 460 kcal), publish/unpublish, copy-week, publish horizon.

## M7 — Pricing ✅
Four tables, four URLs, exclusion constraints reporting the conflicting row,
scope→DEFAULT→promo resolution, flat tiers, the D-9 corporate warning.

## M8 — Ordering ✅
Cart, cut-off, capacity on both counters, routing per delivery, snapshots,
idempotency. **Oversell proved impossible**: 12 concurrent orders on 4
portions → 4 succeed, 8 refused, no oversell.

## M9 — Payments ✅
Verification queue oldest-first, locked verify, rejection with reason, proof
upload validation, auto-expiry returning both counters.
🟡 **Object storage not wired** — proofs are recorded by key, not yet uploaded.

## M10 — Packages & credits ✅
Purchase, activation on verification, append-only ledger, booking.
**Double-spend proved impossible**: 8 concurrent bookings on 1 credit → 1
succeeds, balance 0, never negative.

## M11 — Reports ✅
All eight, kitchen-scoped, CSV-safe against spreadsheet formula injection.

## M12 — Notifications 🟡
Nine templates in both languages, SMTP sender, escaping and header-injection
tested. **The queue and the scheduled sends are not wired** — nothing is
actually emailed yet; the verification link is logged with a warning.

## M13 — Public site ✅
Home and per-diet menu pages, server-rendered, OG/Twitter tags static in the
HTML (verified with `curl`), robots.txt, sitemap.xml, JSON-LD.
**Verified by looking at it**: screenshots at 360px and 1280px in
`docs/screenshots/`, fonts confirmed resolved from the DOM.

## M14 — Security suite ✅
Injection, oversell, constraint-as-last-line, append-only, price overlap,
order reconciliation (8 orders), no negative balances, one REDEEM per delivery.
Runs against a real database; skips loudly without one.

## M15 — Deployment ✅ (development server)
systemd + nginx + hardened unit + first-run admin. Handbook at `14`.

## M16 — Documents ✅ / 🟡
✅ `12-security` (control map with the test that proves each) ·
✅ `14` deployment handbook · ✅ `15` user guide · ✅ `16` admin guide ·
⬜ `01-PRD`, `02-business-rules` with `BR-x.y` ids, `03-data-model`,
`04-api-specification`, `05`, `06`, `07`, `08`, `11`, `13a` — the planning
documents cover their content but the house numbering has not been folded in
(see `04-milestones.md` §2).

---

## Not built — be clear about it

| | Why it matters |
|---|---|
| **Back-office React UI** | The API is complete; there is no admin screen yet. Staff would use a REST client today. |
| **Customer SPA** | Same: ordering works over the API, there is no cart UI. |
| **Notification queue** | Templates exist; nothing is sent. |
| **Object storage** | Payment proofs are recorded, not stored. |
| **TOTP enrolment** | Required for admin/finance/staff before launch. |
| **Turnstile CAPTCHA** | Chosen, needs a key. |
| **UU PDP export/deletion** | Required before launch. Note the tension: tax records must be kept, so deletion should anonymise rather than delete. |
| **CI, govulncheck, gosec** | No pipeline exists. |
| **Backups** | Script sketched in `14` §10, never run. An untested restore is not a backup. |
| **Delivery lifecycle transitions** | The states and reports exist; kitchen/courier "mark prepared/delivered" endpoints are not built. |
| **§13 extras** | TDEE calculator, allergen warnings, vouchers, ratings, nutrition chart — all phase 2 per D-23. |

## Blocked on Steven

| # | Needed | Blocks |
|---|---|---|
| 1 | **Real kitchens** — pins, radii, capacities | Routing is answering from two placeholders |
| 2 | **Bank account** | Nothing can go public: instructions say DUMMY |
| 3 | **Google Maps keys** | The address pin picker cannot be built or exercised |
| 4 | **SMTP relay + SPF/DKIM/DMARC** | Real email |
| 5 | **Domain + production host** | TLS, public deployment |
| 6 | **PKP status, NPWP, legal entity** | The first real invoice |
| 7 | **Reversed-out logo** | The green header uses a text wordmark today |
| 8 | **WhatsApp sender number** | The WAHA channel stays off |
