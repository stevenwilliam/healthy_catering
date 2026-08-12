-- 0008 — bank accounts, orders, order lines, payments.
--
-- Money is BIGINT whole rupiah throughout, and every price stored here is a
-- SNAPSHOT (PROMPT §5.6): editing a price later must never move a historical
-- order. The snapshot columns therefore carry no foreign key to the price row.

CREATE TABLE bank_account (
  id             UUID PRIMARY KEY,
  bank_name      TEXT NOT NULL,
  account_number TEXT NOT NULL,
  account_holder TEXT NOT NULL,
  branch         TEXT NOT NULL DEFAULT '',
  is_active      BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order     INT NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by     UUID REFERENCES app_user(id),
  UNIQUE (bank_name, account_number)
);

CREATE TABLE customer_order (
  id                 UUID PRIMARY KEY,
  order_code         TEXT NOT NULL UNIQUE,
  customer_id        UUID NOT NULL REFERENCES customer(id) ON DELETE RESTRICT,
  customer_type_id   UUID NOT NULL REFERENCES customer_type(id) ON DELETE RESTRICT,
  order_type         TEXT NOT NULL CHECK (order_type IN ('MEAL','PACKAGE')),
  status             TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN (
                       'DRAFT','AWAITING_PAYMENT','PAYMENT_SUBMITTED','PAID',
                       'COMPLETED','EXPIRED','CANCELLED','REFUNDED')),

  -- Money. Every figure tax-inclusive except the explicit base/tax split.
  subtotal_idr       BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_idr     >= 0),
  delivery_fee_idr   BIGINT NOT NULL DEFAULT 0 CHECK (delivery_fee_idr >= 0),
  discount_idr       BIGINT NOT NULL DEFAULT 0 CHECK (discount_idr     >= 0),
  total_idr          BIGINT NOT NULL DEFAULT 0 CHECK (total_idr        >= 0),
  -- Sum of the line splits, never re-derived from total_idr (docs/02 D-30).
  tax_base_idr       BIGINT NOT NULL DEFAULT 0 CHECK (tax_base_idr >= 0),
  tax_idr            BIGINT NOT NULL DEFAULT 0 CHECK (tax_idr      >= 0),
  tax_rate_bps       INT    NOT NULL DEFAULT 0 CHECK (tax_rate_bps BETWEEN 0 AND 10000),
  -- The Indonesian kode unik. payment_amount = total + rounding, and rounding
  -- is excluded from the tax base because a matching device is not
  -- consideration (docs/02 D-16, D-30).
  unique_code        INT CHECK (unique_code BETWEEN 1 AND 999),
  payment_rounding_idr BIGINT NOT NULL DEFAULT 0 CHECK (payment_rounding_idr >= 0),
  payment_amount_idr BIGINT NOT NULL DEFAULT 0 CHECK (payment_amount_idr >= 0),
  bank_account_id    UUID REFERENCES bank_account(id) ON DELETE SET NULL,

  payment_deadline_at TIMESTAMPTZ,
  placed_at          TIMESTAMPTZ,
  paid_at            TIMESTAMPTZ,
  completed_at       TIMESTAMPTZ,
  cancelled_at       TIMESTAMPTZ,
  cancel_reason      TEXT,
  notes              TEXT NOT NULL DEFAULT '',
  -- Why this customer paid this price, answerable without re-running the
  -- resolver (docs/01 §3.5).
  price_trace        JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT customer_order_total_consistent
    CHECK (total_idr = subtotal_idr + delivery_fee_idr - discount_idr),
  CONSTRAINT customer_order_payment_consistent
    CHECK (payment_amount_idr = total_idr + payment_rounding_idr),
  CONSTRAINT customer_order_tax_consistent
    CHECK (tax_base_idr + tax_idr = total_idr)
);
CREATE INDEX customer_order_customer_idx ON customer_order (customer_id, created_at DESC);
CREATE INDEX customer_order_status_idx   ON customer_order (status, payment_deadline_at);
CREATE INDEX customer_order_paid_idx     ON customer_order (paid_at DESC) WHERE status IN ('PAID','COMPLETED');

-- The suffix must be unique among orders currently awaiting money, or two
-- customers transfer the same amount and finance cannot tell them apart
-- (docs/02 D-16). Not globally unique — 999 suffixes would run out in a week.
CREATE UNIQUE INDEX customer_order_unique_code_open_idx
  ON customer_order (bank_account_id, payment_amount_idr)
  WHERE status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED');

CREATE TABLE order_line (
  id                UUID PRIMARY KEY,
  order_id          UUID NOT NULL REFERENCES customer_order(id) ON DELETE CASCADE,
  line_no           INT NOT NULL,
  line_type         TEXT NOT NULL CHECK (line_type IN ('MEAL','PACKAGE')),

  -- MEAL lines reference the scheduled meal; PACKAGE lines the package. Kept
  -- as references for reporting, but the money and the composition are
  -- snapshots and do not depend on them surviving.
  scheduled_meal_id UUID REFERENCES scheduled_meal(id) ON DELETE SET NULL,
  package_id        UUID REFERENCES package(id) ON DELETE SET NULL,
  service_date      DATE,
  slot_id           UUID REFERENCES delivery_time_slot(id) ON DELETE SET NULL,
  diet_type_id      UUID REFERENCES diet_type(id) ON DELETE SET NULL,

  qty               INT NOT NULL CHECK (qty > 0),
  unit_price_idr    BIGINT NOT NULL CHECK (unit_price_idr  >= 0),
  normal_price_idr  BIGINT NOT NULL CHECK (normal_price_idr >= 0),
  line_total_idr    BIGINT NOT NULL CHECK (line_total_idr  >= 0),
  line_tax_base_idr BIGINT NOT NULL DEFAULT 0 CHECK (line_tax_base_idr >= 0),
  line_tax_idr      BIGINT NOT NULL DEFAULT 0 CHECK (line_tax_idr      >= 0),
  is_promo          BOOLEAN NOT NULL DEFAULT FALSE,
  promo_label       TEXT NOT NULL DEFAULT '',
  price_row_id      UUID,
  price_table       TEXT CHECK (price_table IN
                      ('meal_normal','meal_promo','package_normal','package_promo')),
  tier_id           UUID REFERENCES meal_price_tier(id) ON DELETE SET NULL,
  -- Every food in the meal, with roles and nutrition, as at purchase.
  meal_snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (order_id, line_no),
  CONSTRAINT order_line_total_consistent CHECK (line_total_idr = unit_price_idr * qty),
  CONSTRAINT order_line_tax_consistent   CHECK (line_tax_base_idr + line_tax_idr = line_total_idr),
  CONSTRAINT order_line_kind_consistent  CHECK (
    (line_type = 'MEAL'    AND scheduled_meal_id IS NOT NULL AND package_id IS NULL) OR
    (line_type = 'PACKAGE' AND package_id        IS NOT NULL AND scheduled_meal_id IS NULL))
);
CREATE INDEX order_line_order_idx ON order_line (order_id, line_no);
CREATE INDEX order_line_meal_idx  ON order_line (scheduled_meal_id) WHERE scheduled_meal_id IS NOT NULL;

CREATE TABLE payment (
  id                 UUID PRIMARY KEY,
  order_id           UUID NOT NULL REFERENCES customer_order(id) ON DELETE CASCADE,
  provider           TEXT NOT NULL DEFAULT 'MANUAL_TRANSFER',
  bank_account_id    UUID REFERENCES bank_account(id) ON DELETE SET NULL,
  expected_amount_idr BIGINT NOT NULL CHECK (expected_amount_idr >= 0),
  paid_amount_idr    BIGINT CHECK (paid_amount_idr IS NULL OR paid_amount_idr >= 0),
  status             TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN
                       ('PENDING','SUBMITTED','VERIFIED','REJECTED','EXPIRED','REFUNDED')),
  submitted_at       TIMESTAMPTZ,
  verified_at        TIMESTAMPTZ,
  verified_by        UUID REFERENCES app_user(id),
  rejected_at        TIMESTAMPTZ,
  rejected_by        UUID REFERENCES app_user(id),
  rejection_reason   TEXT,
  refunded_at        TIMESTAMPTZ,
  refunded_by        UUID REFERENCES app_user(id),
  -- No refunds is the policy (docs/02 D-31); this exists only for an erroneous
  -- or duplicate transfer, and the reason is mandatory when it is used.
  refund_reason      TEXT,
  external_ref       TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT payment_rejection_has_reason
    CHECK (status <> 'REJECTED' OR (rejection_reason IS NOT NULL AND rejection_reason <> '')),
  CONSTRAINT payment_refund_has_reason
    CHECK (status <> 'REFUNDED' OR (refund_reason IS NOT NULL AND refund_reason <> ''))
);
CREATE INDEX payment_order_idx  ON payment (order_id);
CREATE INDEX payment_queue_idx  ON payment (status, submitted_at)
  WHERE status = 'SUBMITTED';

CREATE TABLE payment_proof (
  id           UUID PRIMARY KEY,
  payment_id   UUID NOT NULL REFERENCES payment(id) ON DELETE CASCADE,
  object_key   TEXT NOT NULL,
  content_type TEXT NOT NULL,
  bytes        BIGINT NOT NULL CHECK (bytes > 0),
  checksum     TEXT,
  uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  uploaded_by  UUID REFERENCES app_user(id)
);
CREATE INDEX payment_proof_payment_idx ON payment_proof (payment_id);

CREATE TRIGGER bank_account_touch BEFORE UPDATE ON bank_account
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER customer_order_touch BEFORE UPDATE ON customer_order
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER payment_touch BEFORE UPDATE ON payment
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
