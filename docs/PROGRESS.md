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
✅ **TOTP two-factor** — enrolment, confirmation, login challenge, recovery
codes. Mandatory for admin/finance/staff, refused as optional for those roles;
kitchen and courier are exempt (they sign in from shared phones on a service
floor). Secrets are AES-GCM encrypted at rest under `TOTP_ENCRYPTION_KEY`;
without that key the feature is OFF and the routes are absent, which the boot
log states. Verified end-to-end against the running service with codes
generated from the secret, including the case that matters: **the challenge
token issued after a correct password is refused as a session** (401
"Finish signing in first"). RFC 6238 vectors are pinned in a unit test.

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

**Trilingual, 2026-08-18 (Steven).** Indonesian, English and Simplified
Chinese, with a flag-and-name language selector on both surfaces. Public pages
are path-prefixed (`/`, `/en/`, `/zh/`) with hreflang alternates and a
per-locale sitemap; the app stores the preference. Copy lives in catalogues
(`internal/adapter/http/messages.go`, `web/src/lib/messages.ts`) and the
notification templates gained Chinese. Details and the limits in `docs/11`.
**Not done:** the app's inner screens (~140 strings) are still Indonesian, and
menu CONTENT is single-language by schema — raised as docs/03 Q-24.

**Brand pass, 2026-08-18 (Steven).** Page ground is now Nourish Green `#468973`;
the supplied wordmark replaces the text logo on both surfaces; the footer is the
masthead's fill, fixed to the bottom and thin; the WhatsApp float is larger and
on every page, public and app; the Erode/Inter pairing is bigger and bolder
throughout. Two accessibility consequences are recorded in `10` §2.7: nothing
reaches AA at reading size on the new ground, so body copy sits on beige sheets
and white cards, and the WhatsApp teal measures **1.00:1** against it and now
carries a beige ring. Re-verified by screenshot on home, menu, 404 and the app
at 390px and 1280px. `/images` was also never mounted, so every page's
`og:image` had been 404ing — fixed, and the card is generated.

**Then the two greens were swapped** (Steven): the page is deep `#1C3D34` and
the bars are mid `#468973`. Net accessibility gain — body copy on the page is
11.32 — at the cost of the bars, where beige is 3.93 and every string must be
19px/700 "large text". Favicon derived from the wordmark's leading `e`; a split
hero with a picture on the right, its source a `sys_parameters` row (migration
0015). `docs/10` §2.7 carries the measured tables.

## M14 — Security suite ✅
Injection, oversell, constraint-as-last-line, append-only, price overlap,
order reconciliation (8 orders), no negative balances, one REDEEM per delivery.
Runs against a real database; skips loudly without one.

## M15 — Deployment ✅ (development server)
systemd + nginx + hardened unit + first-run admin. Handbook at `14`.

## M16 — Documents ✅ / 🟡
✅ `12-security` (control map with the test that proves each) ·
✅ `14` deployment handbook · ✅ `15` user guide · ✅ `16` admin guide ·
✅ `11-i18n` (languages, locale negotiation, the selector, and what is *not*
translated) ·
⬜ `01-PRD`, `02-business-rules` with `BR-x.y` ids, `03-data-model`,
`04-api-specification`, `05`, `06`, `07`, `08`, `13a` — the planning
documents cover their content but the house numbering has not been folded in
(see `04-milestones.md` §2).

---

## Not built — be clear about it

| | Why it matters |
|---|---|
| **Turnstile CAPTCHA** | Chosen, needs a key. Registration and login are rate-limited meanwhile, which is weaker. |
| **UU PDP export/deletion** | Required before launch. Note the tension: tax records must be kept, so deletion should anonymise rather than delete. |
| **WhatsApp sending** | WAHA chosen and the queue is generic, but no sender number exists so the channel is off. |
| **Off-machine backup copy** | `scripts/backup.sh` runs and `scripts/restore-check.sh` has been run against a real dump, but the copy still lands on the SAME MACHINE as the database. The S3/rclone line is commented out awaiting a bucket. A backup that dies with the server is not a backup. |
| **CI has never run on a runner** | `.github/workflows/ci.yml` exists and every step in it passes locally, but no push has exercised it on GitHub Actions. Until it does, treat the pipeline as written-not-proven. |
| **§13 extras** | TDEE calculator, allergen warnings, vouchers, ratings, nutrition chart — all phase 2 per D-23. |

### Recently completed (previously in the table above)

Corrected because the list had gone stale and a stale "not built" is as
misleading as a false ✅:

| | Evidence |
|---|---|
| **Customer SPA + back-office screens** | Login (incl. 2FA step), Register, Menu/cart, Addresses, Orders, OrderDetail, Packages, Security, AdminPayments, AdminDeliveries, AdminSettings. Driven in a real browser at 390px and 1280px. |
| **Notification queue** | Postgres queue with `FOR UPDATE SKIP LOCKED` and backoff; real mail delivered end-to-end through the relay and logged `SENT`. |
| **Object storage** | Private MinIO bucket, server-generated keys, magic-byte type detection, presigned reads. Verified: presigned 200, bare URL 403. |
| **Delivery lifecycle transitions** | Domain state machine plus kitchen/courier endpoints, kitchen-scoped. |
| **TOTP two-factor** | See M4. |
| **Backup + restore drill** | `scripts/backup.sh` verifies the gzip stream and the dump's completion marker; `scripts/restore-check.sh` restores the newest dump into a scratch database and runs the security suite against the restored copy. Both have been run. |

## Blocked on Steven

Answered 2026-08-13 (second batch) — see `docs/02-decisions.md` Part 0c:

| # | Needed | Status |
|---|---|---|
| 1 | Real kitchens | ✅ five, geocoded from the real Maps key (migration 0014) |
| 2 | Bank account | ✅ Nobu 16830226665, PT Sunshine Food International |
| 3 | Google Maps keys | 🟡 one key supplied and working; **unrestricted and unsplit** — see below |
| 4 | SMTP relay + SPF/DKIM/DMARC | ⬜ **still open** — the key given under this heading was the Maps key |
| 5 | Domain + production host | 🟡 `dev.evermore.co.id` chosen; DNS does not resolve yet |
| 6 | Legal entity, NPWP | 🟡 name and address landed; **NPWP is a placeholder**, PKP status unanswered |
| 7 | Reversed-out logo | 🟡 deferred by Steven — text wordmark stays for now |
| 8 | WhatsApp sender | 🟡 wired to the shared WAHA; **session is FAILED, must be re-linked** |

### What is still genuinely blocking

| | Why it matters |
|---|---|
| **SMTP relay** | Item 4 was never answered — the key pasted there was the Google Maps key. Real email still goes nowhere but the local trap. |
| **PKP status** | If the company is not PKP, charging 11% PPN is not permitted. This is a legal answer, not a code change; the rate is already a setting. |
| **A real NPWP** | `123 123 123` is stored because it was given, but a real NPWP is 15 digits (16 since the NIK migration). It must not reach a faktur pajak. |
| **Maps key hygiene** | One key does both browser and server duty and allows `0.0.0.0/0`. It ships in page source, so anyone viewing source can spend against it. Split and cap before anything public. |
| **DNS for `dev.evermore.co.id`** | nginx and `APP_BASE_URL` are set; the name does not resolve, so the host answers on its IP today. |
| **WAHA session** | Bound to the right number (628176315568) but reporting `FAILED`. WhatsApp queues and retries; nothing sends until someone re-scans the QR. |
