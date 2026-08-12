# 01 — Domain model

**Project:** Evermore — B2C healthy catering, Jakarta (`www.evermore.co.id`)
**Version:** 0.1 — planning. **Not approved. No code is built from this yet.**
**Date:** 2026-08-12
**Source:** Steven's master prompt (`docs/PROMPT.md` content, 2026-08-12).

This document is the entity model, the relationships, the state machines and the
ERD. Where the brief left a choice open it is marked **[D-n]** and resolved in
`02-decisions.md`; where the brief is silent or self-contradictory the gap is
raised in `03-open-questions.md` as **[Q-n]**. Nothing here invents a rule that
neither document states.

Conventions from `CLAUDE.md`, applied without restating them per table:

- Primary keys are **UUIDv7**. Human-facing codes (order number, kitchen code)
  are CSPRNG + Crockford base32 or a dated sequence, never the primary key.
- Money is **`BIGINT` whole rupiah**. IDR's minor unit (sen) is obsolete and not
  used in retail; `Rp 500.000` is stored as `500000`. No floating point on any
  path that touches money. Percentages are basis points.
- Timestamps are **`timestamptz`, UTC**. A *calendar* date that belongs to the
  business (delivery date, price validity, menu date) is a **`DATE`**, not an
  instant — a delivery on 2026-09-01 is that date in `Asia/Jakarta` regardless of
  the server. Any comparison of "now" against a business date converts through
  `Asia/Jakarta` explicitly.
- Every table carries `created_at`, `updated_at`; every table a human edits
  carries `created_by`, `updated_by`.
- Configurable values are rows in **`sys_parameters`**, not constants. The brief
  says "system settings"; that is this table.

---

## 1. Bounded areas

| # | Area | Owns |
|---|---|---|
| A | Identity & access | users, roles, permissions, 2FA, sessions, audit |
| B | Customer | customers, customer types, organisations, addresses |
| C | Catalogue | diet types, foods, nutrition, allergens, ingredients |
| D | Schedule | delivery time slots, food schedule (menu calendar) |
| E | Kitchen & geography | kitchens, service areas, capacity, routing |
| F | Pricing | tiers, four price tables, resolution, snapshots |
| G | Ordering | carts, orders, order lines, payments |
| H | Packages & credits | packages, customer packages, credit ledger |
| I | Fulfilment | deliveries, manifests, production sheets |
| J | Platform | sys_parameters, notifications, jobs, idempotency |

Layering follows `CLAUDE.md` §2 — the pricing resolver, the credit ledger rules
and the kitchen router are **pure domain code with no I/O**, which is what makes
the near-100% coverage the brief asks for (§14) achievable.

---

## 2. ERD

Split into four diagrams because one is unreadable. Foreign keys are shown; the
`platform` tables (`sys_parameters`, `audit_log`, `notification_log`,
`idempotency_key`, `job`) reference nothing and are omitted from the diagrams.

### 2.1 Identity, customer, address

```mermaid
erDiagram
    user ||--o{ user_role : has
    role ||--o{ user_role : grants
    role ||--o{ role_permission : contains
    permission ||--o{ role_permission : in
    user ||--o| customer : "profile (role=customer)"
    user ||--o| staff_profile : "profile (staff roles)"
    user ||--o{ user_totp : "2FA (staff, mandatory)"
    user ||--o{ refresh_token : holds
    organisation ||--o{ customer : employs
    customer_type ||--o{ customer : classifies
    customer_type ||--o| organisation : "corporate types link to"
    customer ||--o{ customer_address : owns
    kitchen ||--o{ staff_profile : "scopes (D-21)"

    user {
        uuid id PK
        citext email UK
        text password_hash "argon2id"
        text phone
        bool is_active
        timestamptz email_verified_at
        timestamptz last_login_at
    }
    customer {
        uuid id PK
        uuid user_id FK
        uuid customer_type_id FK
        uuid organisation_id FK "nullable"
        text full_name
        date  birth_date "nullable, for TDEE"
        text  gender "nullable"
        jsonb allergen_profile "D-23"
        text  preferred_locale "id-ID default"
        text  notify_channels "email|whatsapp"
    }
    organisation {
        uuid id PK
        text name
        text pic_name
        citext billing_email
        text po_number
        bool is_invoice_billing
        bool is_active
    }
    customer_type {
        uuid id PK
        text name UK
        text slug UK
        bool is_corporate
        bool is_active
        int  sort_order
    }
    customer_address {
        uuid id PK
        uuid customer_id FK
        text label
        text recipient_name
        text recipient_phone
        text address_line
        text district
        text city
        text province
        text postal_code
        numeric latitude "NOT NULL"
        numeric longitude "NOT NULL"
        geography geom "Point,4326 generated, GIST"
        text google_place_id
        text formatted_address
        text driver_note
        bool is_default "one per customer, partial unique"
        bool is_active "soft delete"
    }
```

### 2.2 Catalogue and schedule

```mermaid
erDiagram
    diet_type ||--o{ diet_subtype : "has (Special Diet only)"
    diet_type ||--o{ food_diet_type : allows
    food ||--o{ food_diet_type : "belongs to"
    food ||--|| food_nutrition : "has one per portion"
    food ||--o{ food_photo : has
    food ||--o{ food_allergen : declares
    allergen ||--o{ food_allergen : in
    food ||--o{ food_ingredient : "composed of (D-6, later)"
    ingredient ||--o{ food_ingredient : "used in (D-6, later)"
    food ||--o{ food_schedule : scheduled
    diet_type ||--o{ food_schedule : for
    delivery_time_slot ||--o{ food_schedule : "meal slot"

    diet_type {
        uuid id PK
        text name
        text slug UK
        text description
        text hero_image_key
        int  sort_order
        bool has_subtypes
        bool is_active
    }
    diet_subtype {
        uuid id PK
        uuid diet_type_id FK
        text name
        text slug
        bool is_active
    }
    food {
        uuid id PK
        text name
        text slug UK
        text description
        text portion_size
        bool is_active
    }
    food_nutrition {
        uuid id PK
        uuid food_id FK UK
        int calories_kcal
        int protein_mg
        int fat_mg
        int saturated_fat_mg
        int carbohydrate_mg
        int sugar_mg
        int fibre_mg
        int sodium_mg
        int cholesterol_mg
        jsonb extras
    }
    delivery_time_slot {
        uuid id PK
        time  slot_time "15-min grid"
        text  alias "customer-facing"
        int   sort_order
        bool  is_active
    }
    food_schedule {
        uuid id PK
        date  service_date
        uuid  diet_type_id FK
        uuid  slot_id FK
        uuid  food_id FK
        text  item_role "MAIN|SIDE|DESSERT|DRINK (D-7)"
        int   qty_capacity "nullable"
        text  status "DRAFT|PUBLISHED"
        timestamptz published_at
    }
```

### 2.3 Kitchens, geography, capacity

```mermaid
erDiagram
    kitchen ||--o{ kitchen_slot : serves
    delivery_time_slot ||--o{ kitchen_slot : "served by"
    kitchen ||--o{ kitchen_operating_day : opens
    kitchen ||--o{ kitchen_capacity : "capacity per date+slot"
    kitchen ||--o{ delivery : fulfils
    kitchen ||--o{ out_of_range_attempt : "nearest (nullable)"

    kitchen {
        uuid id PK
        text code UK
        text name
        text address_line
        numeric latitude "NOT NULL"
        numeric longitude "NOT NULL"
        geography geom "Point,4326, GIST"
        numeric service_radius_km
        geography service_area "Polygon,4326 nullable, overrides radius"
        text phone
        text pic_name
        int  default_daily_capacity
        int  priority "lower = preferred"
        bool is_active
        text notes
    }
    kitchen_capacity {
        uuid id PK
        uuid kitchen_id FK
        date service_date
        uuid slot_id FK
        int  max_portions
        int  reserved_portions "decremented atomically"
    }
    out_of_range_attempt {
        uuid id PK
        uuid customer_id FK "nullable, anonymous widget"
        numeric latitude
        numeric longitude
        text district
        uuid slot_id FK
        date service_date
        timestamptz occurred_at
        bool notify_requested
        citext notify_email
    }
```

### 2.4 Pricing, ordering, credits, fulfilment

```mermaid
erDiagram
    meal_price_tier ||--o{ meal_price_normal : "priced at"
    meal_price_tier ||--o{ meal_price_promo : "priced at"
    diet_type ||--o{ meal_price_normal : for
    diet_type ||--o{ meal_price_promo : for
    package ||--o{ package_price_normal : "priced at"
    package ||--o{ package_price_promo : "priced at"
    package ||--o{ package_diet_type : "restricted to (D-12)"
    package ||--o{ customer_package : sold as

    customer ||--o{ order : places
    order ||--o{ order_line : contains
    order ||--o{ payment : "paid by"
    payment ||--o{ payment_proof : evidences
    bank_account ||--o{ payment : "transferred to"
    order ||--o| customer_package : "issues (package order)"
    customer_package ||--o{ credit_ledger : "movements"
    order_line ||--o{ delivery : "produces (meal order)"
    customer_package ||--o{ delivery : "produces (credit redemption)"
    customer_address ||--o{ delivery : "shipped to (snapshotted)"
    delivery ||--o{ delivery_line : contains
    food_schedule ||--o{ delivery_line : "fulfils"

    meal_price_normal {
        uuid id PK
        text scope_key "CT:<uuid> | DEFAULT"
        uuid customer_type_id FK "nullable when DEFAULT"
        uuid diet_type_id FK "D-11"
        uuid tier_id FK
        bigint unit_price_idr
        daterange validity "[) EXCLUDE overlap"
        bool is_active
    }
    meal_price_promo {
        uuid id PK
        text scope_key
        uuid customer_type_id FK
        uuid diet_type_id FK
        uuid tier_id FK
        bigint unit_price_idr
        daterange validity "[) EXCLUDE overlap"
        text  promo_label
        bool is_active
    }
    order {
        uuid id PK
        text order_number UK
        uuid customer_id FK
        text order_type "MEAL|PACKAGE"
        text status
        bigint subtotal_idr
        bigint delivery_fee_idr
        bigint discount_idr
        bigint total_idr
        bigint payment_amount_idr "total + unique suffix (D-16)"
        timestamptz payment_deadline_at
        text idempotency_key
        jsonb price_resolution_trace "why this price"
    }
    order_line {
        uuid id PK
        uuid order_id FK
        uuid food_id FK "nullable for package lines"
        uuid package_id FK "nullable for meal lines"
        int  qty
        bigint unit_price_idr "snapshot"
        bigint normal_price_idr "snapshot"
        bool  is_promo
        uuid  price_row_id "snapshot, no FK - history"
        text  price_table "meal_normal|meal_promo|pkg_normal|pkg_promo"
        jsonb food_snapshot "name, portion, nutrition, allergens"
    }
    credit_ledger {
        uuid id PK
        uuid customer_id FK
        uuid customer_package_id FK
        text entry_type "PURCHASE|REDEEM|REFUND|EXPIRE|ADJUSTMENT"
        int  qty "signed"
        text reference_type
        uuid reference_id
        timestamptz occurred_at
        uuid created_by
        text note
    }
    delivery {
        uuid id PK
        uuid order_id FK "nullable"
        uuid customer_package_id FK "nullable"
        uuid customer_id FK
        date service_date
        uuid slot_id FK
        uuid diet_type_id FK
        uuid kitchen_id FK
        uuid address_id FK
        jsonb address_snapshot "text + lat/lng at confirmation"
        int  assigned_distance_m
        text assignment_mode "AUTO|MANUAL"
        text assignment_reason
        timestamptz assigned_at
        text status
        timestamptz delivered_at
        text failure_reason
    }
```

---

## 3. Entities in detail — the parts the diagram cannot carry

### 3.1 Customer types and organisations (§4.1)

Table-driven, never an enum. Seeded: `Customer Default`, `Siloam Customer`,
`Company A`, `Company B`. `Customer Default` carries `is_system = true` so it
cannot be deleted — every registration lands on it and a missing default breaks
signup.

`is_corporate` on the type drives the UI (a corporate customer must be attached
to an `organisation`), but the **link is on the customer**, not the type — two
companies can share a type and one company can have staff on different types.
Changing a customer's type is staff-only, audit-logged, and — critically —
**does not reprice existing orders**, because prices are snapshotted (§5.6).

### 3.2 Diet types (§4.2)

`diet_type` → `diet_subtype` is nullable and only `Special Diet` seeds subtypes
(`Diabetic`, `Cholesterol`). The schedule, pricing and ordering all key on
`diet_type_id`; the subtype is descriptive metadata on the customer and on the
food, never a pricing or scheduling axis. That keeps the price matrix from
multiplying by subtype. **[Q-6]** confirms this is intended.

### 3.3 Food and nutrition (§4.3)

Nutrition is stored as **integers in milligrams** (calories in whole kcal), not
decimals — the same reason money is integers, and it makes the weekly-intake
chart (§13.8) exact. Display divides by 1000 at the edge. `extras JSONB` takes
anything the label needs that is not a column.

`ingredient` + `food_ingredient` (per-100g nutrition, auto-computing the panel)
is modelled but **not built in phase 1** — see **[D-6]**.

### 3.4 Food schedule (§4.4)

`food_schedule` is the menu calendar. One row per
(`service_date`, `diet_type`, `slot`, `food`), with `item_role` distinguishing
main / side / dessert / drink so a composed set menu is one date+diet+slot with
several rows — **[D-7]**, recommended.

Unique index on (`service_date`, `diet_type_id`, `slot_id`, `food_id`) stops the
same dish being scheduled twice in one sitting. `status` is `DRAFT` until
published; **customers only ever query `PUBLISHED`**, enforced in the repository,
not the handler.

The schedule is **global across kitchens** — **[D-8]**. `kitchen_id` is
deliberately absent; adding it later is an additive migration.

### 3.5 Pricing (§5)

Four tables, four admin forms, as required. All four share the same shape:

- `scope_key TEXT NOT NULL` — either `'DEFAULT'` or `'CT:<customer_type_uuid>'`.
  A generated column derived from `customer_type_id` so it cannot drift, and the
  thing the exclusion constraint keys on (a nullable `customer_type_id` cannot be
  compared with `=` in an exclusion constraint — `NULL` is not equal to `NULL`,
  so two `DEFAULT` rows would both be accepted. That is the reason the brief's
  `scope_key` exists, and it is load-bearing).
- `validity daterange` with `[)` bounds; open-ended is `[valid_from,)`.
- `EXCLUDE USING gist (...)` per §5.3, requiring `btree_gist`.

**Resolution order** (pure function, exhaustively unit-tested):

1. Resolve **scope**: a row whose `scope_key = 'CT:<customer's type>'` and whose
   `validity` contains the *order date*. If none, `scope_key = 'DEFAULT'`.
2. Within the resolved scope, resolve **promo over normal** — **[D-9]**.
3. No row in either scope → **block**, `PRICE_NOT_CONFIGURED`, never guess.

Every resolution writes a `price_resolution_trace` onto the order (which scope
matched, which table, which row id, which tier) so "why did this customer pay
that?" is answerable from the record without re-running the resolver.

**Tiers** (`meal_price_tier`): `min_qty`, `max_qty` nullable = ∞. Validated on
save to have no overlaps and no gaps from 1 up to the max order quantity from
`sys_parameters` (999). Semantics are **flat** — **[D-10]**.

### 3.6 Packages and credits (§5.5, §7)

`customer_package` is the purchased instance: `package_id`, `meal_credits`
(snapshot), `validity_days` (snapshot), `activated_at`, `expires_at` (a **DATE**
in `Asia/Jakarta`), `status`.

Balance is **never stored**. `remaining = SUM(credit_ledger.qty)` for the
package, and every movement is an append-only row. The table has no `UPDATE` or
`DELETE` path at all — that is stated in its migration, per `CLAUDE.md` §4.

Redemption is a transaction that takes `SELECT ... FOR UPDATE` on the
`customer_package` row, recomputes the balance inside the lock, and refuses at
zero. A `CHECK` cannot express `SUM(...) >= 0` across rows, so the lock is the
invariant's enforcement point and it ships with the concurrency test the house
rules demand.

### 3.7 Orders, deliveries and the two flows (§6)

An **order** is the commercial event. A **delivery** is the fulfilment event.
They are separate because one package order produces N deliveries over weeks, to
different addresses, through different kitchens.

- **Flow A (meals):** order → order lines → deliveries created at checkout, one
  per (date, slot, address) combination in the cart.
- **Flow B (packages):** order → payment verified → `customer_package` +
  `PURCHASE` ledger entry → customer picks slots later; **each picked portion is
  one `REDEEM` entry and one `delivery_line`**, grouped into a `delivery` per
  (date, slot, address).

Kitchen routing runs **per delivery** at the moment the delivery is created —
for packages that is slot-pick time, not purchase time. Kitchen capacity is
therefore consumed at slot-pick time too. **[Q-9]** raises what this means for a
customer who buys 20 credits and finds their kitchen full.

### 3.8 Addresses and delivery (§8)

`latitude`/`longitude` are `NOT NULL`. `geom` is a **generated** column
(`ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography`) so it cannot
disagree with the numbers, GIST-indexed. Server-side bounds validation rejects
anything outside a Jabodetabek envelope **[Q-11]** as well as the obvious
`(0,0)`.

`is_default` is enforced by a **partial unique index**
(`WHERE is_default AND is_active`), not application logic.

Address text **and coordinates are snapshotted onto the delivery** at
confirmation. Editing a saved address never rewrites history.

### 3.9 Payments (§10)

`payment` holds `expected_amount_idr` (order total) and
`payment_amount_idr` (total plus the unique suffix, **[D-16]**). These are
separate columns because the suffix means **the amount transferred is not the
sum of the lines**, and a report that adds line totals will disagree with the
bank statement by up to Rp 999 per order unless the difference has a home. The
suffix difference is posted as `payment_rounding_idr` on the order so the sales
report reconciles exactly.

`bank_account` is a table, not a setting blob — multiple accounts, each with
`is_active` and `sort_order`.

### 3.10 Platform tables

`sys_parameters` (key, value, type, group, description, `is_secret`,
`updated_by`) with full admin CRUD and search, per `CLAUDE.md` §7. Seeded with
every tunable this brief names: cut-off time and lead days, payment deadline,
max qty per line, delivery fee bands, free-delivery threshold, low-credit
threshold, expiry-warning days, bank accounts' display copy, WhatsApp CS number,
Google Maps daily quota alarm thresholds.

`audit_log` (actor, action, entity_type, entity_id, before JSONB, after JSONB,
ip, user_agent, occurred_at) — append-only, written for every staff action
touching money, prices, customer type, credits, package expiry, kitchen
assignment and service areas.

`notification_log`, `idempotency_key`, `job` complete the set.

---

## 4. State machines

### 4.1 Order — commercial lifecycle

The brief's §6.3 list mixes commercial states with fulfilment states.
`SCHEDULED → PREPARING → OUT_FOR_DELIVERY → DELIVERED` describes a *delivery*,
and an order with 20 deliveries cannot be in one of them. Recommendation
**[D-15]**: the order owns the commercial states, the delivery owns fulfilment,
and the brief's fulfilment names remain available on the API as a **derived**
read-only `fulfilment_status` so nothing in §6.3 is lost.

```mermaid
stateDiagram-v2
    [*] --> DRAFT : cart
    DRAFT --> AWAITING_PAYMENT : checkout
    AWAITING_PAYMENT --> PAYMENT_SUBMITTED : proof uploaded
    AWAITING_PAYMENT --> EXPIRED : deadline passed (job)
    AWAITING_PAYMENT --> CANCELLED : customer or staff
    PAYMENT_SUBMITTED --> PAID : finance verifies
    PAYMENT_SUBMITTED --> AWAITING_PAYMENT : finance rejects (reason)
    PAYMENT_SUBMITTED --> EXPIRED : deadline passed, unverified
    PAID --> COMPLETED : all deliveries DELIVERED / package expired
    PAID --> CANCELLED : staff, before cut-off
    PAID --> REFUNDED : finance
    CANCELLED --> REFUNDED : finance
    EXPIRED --> [*]
    COMPLETED --> [*]
    REFUNDED --> [*]
```

Illegal transitions are rejected by the domain layer, not the handler, and the
transition table is a unit test.

### 4.2 Delivery — fulfilment lifecycle

```mermaid
stateDiagram-v2
    [*] --> SCHEDULED : created + routed
    SCHEDULED --> PREPARING : kitchen starts (after cut-off)
    SCHEDULED --> SKIPPED : customer/staff, before cut-off (credit returned)
    SCHEDULED --> CANCELLED : order cancelled
    SCHEDULED --> SCHEDULED : reschedule / re-route, before cut-off only
    PREPARING --> OUT_FOR_DELIVERY : courier picks up
    OUT_FOR_DELIVERY --> DELIVERED : courier confirms
    OUT_FOR_DELIVERY --> FAILED : nobody home, wrong address
    FAILED --> SCHEDULED : staff reschedules (credit decision, Q-8)
    DELIVERED --> [*]
```

### 4.3 Customer package

```mermaid
stateDiagram-v2
    [*] --> PENDING : order placed, unpaid
    PENDING --> ACTIVE : payment verified (D-14 sets activated_at)
    PENDING --> CANCELLED : order expired or cancelled
    ACTIVE --> EXHAUSTED : balance reaches 0
    ACTIVE --> EXPIRED : expires_at passed (job posts EXPIRE entry)
    EXHAUSTED --> ACTIVE : REFUND entry returns a credit (skip before cut-off)
    ACTIVE --> ACTIVE : staff extends expiry (audit-logged, reason required)
    EXPIRED --> ACTIVE : staff extends expiry (audit-logged, reason required)
    EXHAUSTED --> [*]
    EXPIRED --> [*]
```

Reactivating an `EXPIRED` package by extending the expiry must **reverse the
`EXPIRE` ledger entry with a compensating `ADJUSTMENT`**, never delete it — the
ledger is append-only and the balance history has to reconcile.

### 4.4 Payment

```mermaid
stateDiagram-v2
    [*] --> PENDING : order awaiting payment
    PENDING --> SUBMITTED : proof uploaded
    SUBMITTED --> VERIFIED : finance
    SUBMITTED --> REJECTED : finance, reason required
    REJECTED --> SUBMITTED : customer re-uploads
    PENDING --> EXPIRED : deadline
    VERIFIED --> REFUNDED : finance
```

---

## 5. The three algorithms that get near-100% coverage

### 5.1 Price resolver — pure

```
resolve(customer_type, diet_type, tier_qty, order_date) -> ResolvedPrice | error
```

Test matrix: scope hit · scope miss falling back to DEFAULT · both missing
(blocks) · promo overriding normal in the same scope · **customer-type normal
beating a DEFAULT promo (D-9)** · tier boundary at min · at max · at max+1 · open
-ended `max_qty` · validity boundary on `valid_from` · on `valid_to` (exclusive)
· overlapping rows rejected by the constraint · qty above the configured max.

### 5.2 Credit ledger — pure rules + one locked transaction

Test matrix: purchase posts +N · redeem posts −1 · balance never below zero ·
two concurrent redemptions of the last credit, one wins (the concurrency test) ·
skip before cut-off refunds · skip after cut-off does not · expiry posts the
negative remainder · expiry racing a redemption · staff extension reverses an
expiry with a compensating entry · adjustment requires a reason.

### 5.3 Kitchen router — pure, given candidates

```
route(address_point, slot, date, kitchens[], capacity[]) -> Assignment | NotServiceable
```

Test matrix, exactly as the brief lists it: inside one radius · inside two
overlapping radii (priority decides, then distance) · inside a polygon but
outside that kitchen's radius (polygon wins) · inside a radius but outside the
polygon (polygon wins, not serviceable from that kitchen) · outside everything
(blocks, logs the attempt) · candidate at capacity is dropped · inactive kitchen
dropped · kitchen not serving that slot dropped · kitchen closed that weekday
dropped · **manual assignment is never overwritten by a re-route** · re-route
after cut-off refused.

---

## 6. What this model deliberately does not do yet

- No `kitchen_id` on `food_schedule` (**[D-8]**, global menu).
- No `ingredient` build (**[D-6]**), though the tables are designed.
- No voucher/referral tables — §13.5 says they live in a **discount layer applied
  after price resolution**, so they never touch the four price tables. Modelled
  in `04-milestones.md` as a later slice.
- No per-employee corporate credit allocation UI, but `organisation` and the
  ledger's `customer_package` ownership are shaped so it is additive (§13.6).
- No QRIS. `PaymentProvider` is an interface from day one with
  `ManualTransferProvider` as the only implementation (§10).
