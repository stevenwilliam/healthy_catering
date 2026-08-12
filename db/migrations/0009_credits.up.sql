-- 0009 — packages sold, and the credit ledger.
--
-- The balance is NEVER stored (PROMPT §7): remaining = SUM(credit_ledger.qty).
-- The ledger is append-only and the database enforces that, so a balance
-- history always reconciles and nobody can quietly correct a mistake by
-- editing yesterday.

CREATE TABLE customer_package (
  id              UUID PRIMARY KEY,
  customer_id     UUID NOT NULL REFERENCES customer(id) ON DELETE RESTRICT,
  package_id      UUID NOT NULL REFERENCES package(id) ON DELETE RESTRICT,
  order_id        UUID REFERENCES customer_order(id) ON DELETE SET NULL,
  -- Snapshots: editing the package later must not change what was sold.
  meal_credits    INT NOT NULL CHECK (meal_credits > 0),
  validity_days   INT NOT NULL CHECK (validity_days > 0),
  package_name    TEXT NOT NULL,
  price_paid_idr  BIGINT NOT NULL DEFAULT 0 CHECK (price_paid_idr >= 0),
  status          TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN
                    ('PENDING','ACTIVE','EXHAUSTED','EXPIRED','CANCELLED')),
  purchased_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- The active period starts on payment verification, not purchase
  -- (docs/02 D-14): manual transfer can lag by a weekend.
  activated_at    TIMESTAMPTZ,
  expires_at      DATE,
  expired_at      TIMESTAMPTZ,
  extension_count INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT customer_package_active_has_dates
    CHECK (status = 'PENDING' OR status = 'CANCELLED'
           OR (activated_at IS NOT NULL AND expires_at IS NOT NULL))
);
CREATE INDEX customer_package_customer_idx ON customer_package (customer_id, status);
CREATE INDEX customer_package_expiry_idx   ON customer_package (expires_at)
  WHERE status = 'ACTIVE';

CREATE TABLE credit_ledger (
  id                  UUID PRIMARY KEY,
  customer_id         UUID NOT NULL REFERENCES customer(id) ON DELETE RESTRICT,
  customer_package_id UUID NOT NULL REFERENCES customer_package(id) ON DELETE RESTRICT,
  entry_type          TEXT NOT NULL CHECK (entry_type IN
                        ('PURCHASE','REDEEM','REFUND','EXPIRE','ADJUSTMENT')),
  -- Signed: +20 on purchase, -1 per meal redeemed (docs/02 D-32).
  qty                 INT NOT NULL CHECK (qty <> 0),
  reference_type      TEXT,
  reference_id        UUID,
  note                TEXT NOT NULL DEFAULT '',
  occurred_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by          UUID REFERENCES app_user(id),
  -- Signs must match their meaning, so a typo cannot post a positive EXPIRE.
  CONSTRAINT credit_ledger_sign CHECK (
    (entry_type = 'PURCHASE' AND qty > 0) OR
    (entry_type = 'REDEEM'   AND qty < 0) OR
    (entry_type = 'REFUND'   AND qty > 0) OR
    (entry_type = 'EXPIRE'   AND qty < 0) OR
    (entry_type = 'ADJUSTMENT')),
  -- An adjustment without a reason is an unexplained change to someone's
  -- balance (PROMPT §6.2.4).
  CONSTRAINT credit_ledger_adjustment_has_note
    CHECK (entry_type <> 'ADJUSTMENT' OR note <> '')
);
CREATE INDEX credit_ledger_package_idx  ON credit_ledger (customer_package_id, occurred_at);
CREATE INDEX credit_ledger_customer_idx ON credit_ledger (customer_id, occurred_at DESC);
CREATE INDEX credit_ledger_ref_idx      ON credit_ledger (reference_type, reference_id);

-- One REDEEM per delivery line, so a retry cannot spend a second credit for the
-- same meal.
CREATE UNIQUE INDEX credit_ledger_redeem_once_idx
  ON credit_ledger (reference_type, reference_id)
  WHERE entry_type = 'REDEEM';

-- Append-only, enforced by the database (CLAUDE.md §4).
CREATE TRIGGER credit_ledger_append_only
  BEFORE UPDATE OR DELETE ON credit_ledger
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

CREATE TRIGGER customer_package_touch BEFORE UPDATE ON customer_package
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

COMMENT ON TABLE credit_ledger IS
  'Append-only. remaining = SUM(qty). No UPDATE or DELETE path exists: a mistake is corrected by posting a compensating ADJUSTMENT, never by rewriting history (docs/02 D-26, D-28).';
