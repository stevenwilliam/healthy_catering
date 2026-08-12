-- 0007 — tiers, packages and the four price tables.
--
-- Four separate tables with four separate admin forms: an explicit requirement
-- (PROMPT §5.2), not an accident of modelling.
--
-- Two things here are load-bearing and easy to get subtly wrong:
--
--  1. scope_key is a GENERATED column, not a hand-set string. An exclusion
--     constraint compares with `=`, and NULL = NULL is NULL, not true — so a
--     nullable customer_type_id would let two DEFAULT rows for the same date
--     both be accepted, and §5.3 would silently not hold. The text key is what
--     makes the constraint real (docs/02 D-25).
--
--  2. Every price is TAX-INCLUSIVE (docs/02 D-30, Steven 2026-08-12). The
--     base/tax split is NOT stored here: it is computed and snapshotted onto
--     the order, so changing the rate never rewrites the price tables and a
--     historical row never carries a rate it was not sold under.

CREATE TABLE meal_price_tier (
  id         UUID PRIMARY KEY,
  label      TEXT NOT NULL,
  min_qty    INT NOT NULL CHECK (min_qty >= 1),
  max_qty    INT CHECK (max_qty IS NULL OR max_qty >= min_qty),
  sort_order INT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by UUID REFERENCES app_user(id),
  -- Tier ranges must not overlap (PROMPT §5.4). int4range is half-open, so an
  -- open-ended tier is [min, infinity).
  EXCLUDE USING gist (
    int4range(min_qty, COALESCE(max_qty + 1, 2147483647)) WITH &&
  ) WHERE (is_active)
);
COMMENT ON TABLE meal_price_tier IS
  'Quantity tiers counted in MEALS (docs/02 D-32). Flat semantics: the whole order is priced at the rate of the tier its total lands in (D-10).';

CREATE TABLE package (
  id            UUID PRIMARY KEY,
  name          TEXT NOT NULL,
  slug          TEXT NOT NULL UNIQUE,
  description   TEXT NOT NULL DEFAULT '',
  meal_credits  INT NOT NULL CHECK (meal_credits > 0),
  validity_days INT NOT NULL CHECK (validity_days > 0),
  hero_image_key TEXT,
  sort_order    INT NOT NULL DEFAULT 0,
  is_active     BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by    UUID REFERENCES app_user(id)
);

-- No rows = any diet type (docs/02 D-12): both behaviours, no flag.
CREATE TABLE package_diet_type (
  package_id   UUID NOT NULL REFERENCES package(id) ON DELETE CASCADE,
  diet_type_id UUID NOT NULL REFERENCES diet_type(id) ON DELETE CASCADE,
  PRIMARY KEY (package_id, diet_type_id)
);

-- ── meal prices ──────────────────────────────────────────────────────────────

CREATE TABLE meal_price_normal (
  id               UUID PRIMARY KEY,
  customer_type_id UUID REFERENCES customer_type(id) ON DELETE CASCADE,
  scope_key        TEXT GENERATED ALWAYS AS
                   (COALESCE('CT:' || customer_type_id::text, 'DEFAULT')) STORED,
  diet_type_id     UUID NOT NULL REFERENCES diet_type(id) ON DELETE RESTRICT,
  tier_id          UUID NOT NULL REFERENCES meal_price_tier(id) ON DELETE RESTRICT,
  unit_price_idr   BIGINT NOT NULL CHECK (unit_price_idr >= 0),
  valid_from       DATE NOT NULL,
  valid_to         DATE,
  validity         DATERANGE GENERATED ALWAYS AS (daterange(valid_from, valid_to, '[)')) STORED,
  note             TEXT NOT NULL DEFAULT '',
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by       UUID REFERENCES app_user(id),
  updated_by       UUID REFERENCES app_user(id),
  CONSTRAINT meal_price_normal_dates CHECK (valid_to IS NULL OR valid_to > valid_from),
  CONSTRAINT meal_price_normal_no_overlap EXCLUDE USING gist (
    scope_key    WITH =,
    diet_type_id WITH =,
    tier_id      WITH =,
    validity     WITH &&
  ) WHERE (is_active)
);
CREATE INDEX meal_price_normal_lookup_idx
  ON meal_price_normal (scope_key, diet_type_id, tier_id) WHERE is_active;

CREATE TABLE meal_price_promo (
  id               UUID PRIMARY KEY,
  customer_type_id UUID REFERENCES customer_type(id) ON DELETE CASCADE,
  scope_key        TEXT GENERATED ALWAYS AS
                   (COALESCE('CT:' || customer_type_id::text, 'DEFAULT')) STORED,
  diet_type_id     UUID NOT NULL REFERENCES diet_type(id) ON DELETE RESTRICT,
  tier_id          UUID NOT NULL REFERENCES meal_price_tier(id) ON DELETE RESTRICT,
  unit_price_idr   BIGINT NOT NULL CHECK (unit_price_idr >= 0),
  promo_label      TEXT NOT NULL DEFAULT '',
  valid_from       DATE NOT NULL,
  valid_to         DATE,
  validity         DATERANGE GENERATED ALWAYS AS (daterange(valid_from, valid_to, '[)')) STORED,
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by       UUID REFERENCES app_user(id),
  updated_by       UUID REFERENCES app_user(id),
  CONSTRAINT meal_price_promo_dates CHECK (valid_to IS NULL OR valid_to > valid_from),
  -- Scoped to itself: a promo IS allowed to overlap a normal price — that is
  -- the point of a promo (PROMPT §5.3).
  CONSTRAINT meal_price_promo_no_overlap EXCLUDE USING gist (
    scope_key    WITH =,
    diet_type_id WITH =,
    tier_id      WITH =,
    validity     WITH &&
  ) WHERE (is_active)
);
CREATE INDEX meal_price_promo_lookup_idx
  ON meal_price_promo (scope_key, diet_type_id, tier_id) WHERE is_active;

-- ── package prices ───────────────────────────────────────────────────────────

CREATE TABLE package_price_normal (
  id               UUID PRIMARY KEY,
  customer_type_id UUID REFERENCES customer_type(id) ON DELETE CASCADE,
  scope_key        TEXT GENERATED ALWAYS AS
                   (COALESCE('CT:' || customer_type_id::text, 'DEFAULT')) STORED,
  package_id       UUID NOT NULL REFERENCES package(id) ON DELETE RESTRICT,
  price_idr        BIGINT NOT NULL CHECK (price_idr >= 0),
  valid_from       DATE NOT NULL,
  valid_to         DATE,
  validity         DATERANGE GENERATED ALWAYS AS (daterange(valid_from, valid_to, '[)')) STORED,
  note             TEXT NOT NULL DEFAULT '',
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by       UUID REFERENCES app_user(id),
  updated_by       UUID REFERENCES app_user(id),
  CONSTRAINT package_price_normal_dates CHECK (valid_to IS NULL OR valid_to > valid_from),
  CONSTRAINT package_price_normal_no_overlap EXCLUDE USING gist (
    scope_key  WITH =,
    package_id WITH =,
    validity   WITH &&
  ) WHERE (is_active)
);
CREATE INDEX package_price_normal_lookup_idx
  ON package_price_normal (scope_key, package_id) WHERE is_active;

CREATE TABLE package_price_promo (
  id               UUID PRIMARY KEY,
  customer_type_id UUID REFERENCES customer_type(id) ON DELETE CASCADE,
  scope_key        TEXT GENERATED ALWAYS AS
                   (COALESCE('CT:' || customer_type_id::text, 'DEFAULT')) STORED,
  package_id       UUID NOT NULL REFERENCES package(id) ON DELETE RESTRICT,
  price_idr        BIGINT NOT NULL CHECK (price_idr >= 0),
  promo_label      TEXT NOT NULL DEFAULT '',
  valid_from       DATE NOT NULL,
  valid_to         DATE,
  validity         DATERANGE GENERATED ALWAYS AS (daterange(valid_from, valid_to, '[)')) STORED,
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by       UUID REFERENCES app_user(id),
  updated_by       UUID REFERENCES app_user(id),
  CONSTRAINT package_price_promo_dates CHECK (valid_to IS NULL OR valid_to > valid_from),
  CONSTRAINT package_price_promo_no_overlap EXCLUDE USING gist (
    scope_key  WITH =,
    package_id WITH =,
    validity   WITH &&
  ) WHERE (is_active)
);
CREATE INDEX package_price_promo_lookup_idx
  ON package_price_promo (scope_key, package_id) WHERE is_active;

CREATE TRIGGER meal_price_tier_touch BEFORE UPDATE ON meal_price_tier
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER package_touch BEFORE UPDATE ON package
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER meal_price_normal_touch BEFORE UPDATE ON meal_price_normal
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER meal_price_promo_touch BEFORE UPDATE ON meal_price_promo
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER package_price_normal_touch BEFORE UPDATE ON package_price_normal
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER package_price_promo_touch BEFORE UPDATE ON package_price_promo
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
