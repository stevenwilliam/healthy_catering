-- 0003 — customer types, organisations, customers, addresses.
--
-- Customer type is table-driven, never an enum (PROMPT §4.1). Addresses carry a
-- mandatory pin: the coordinates are the source of truth for routing and the
-- text is for the driver (PROMPT §8.2).

CREATE TABLE customer_type (
  id           UUID PRIMARY KEY,
  name         TEXT NOT NULL UNIQUE,
  slug         TEXT NOT NULL UNIQUE,
  description  TEXT NOT NULL DEFAULT '',
  is_corporate BOOLEAN NOT NULL DEFAULT FALSE,
  -- The default type cannot be deleted: every registration lands on it and a
  -- missing default breaks signup.
  is_system    BOOLEAN NOT NULL DEFAULT FALSE,
  is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order   INT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by   UUID REFERENCES app_user(id)
);

CREATE TABLE organisation (
  id                UUID PRIMARY KEY,
  name              TEXT NOT NULL,
  slug              TEXT NOT NULL UNIQUE,
  pic_name          TEXT NOT NULL DEFAULT '',
  pic_phone         TEXT,
  billing_email     CITEXT,
  billing_address   TEXT NOT NULL DEFAULT '',
  npwp              TEXT,
  po_number         TEXT,
  is_invoice_billing BOOLEAN NOT NULL DEFAULT FALSE,
  invoice_day       INT CHECK (invoice_day BETWEEN 1 AND 28),
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  notes             TEXT NOT NULL DEFAULT '',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by        UUID REFERENCES app_user(id)
);

CREATE TABLE customer (
  id                UUID PRIMARY KEY,
  user_id           UUID NOT NULL UNIQUE REFERENCES app_user(id) ON DELETE CASCADE,
  customer_type_id  UUID NOT NULL REFERENCES customer_type(id) ON DELETE RESTRICT,
  organisation_id   UUID REFERENCES organisation(id) ON DELETE SET NULL,
  birth_date        DATE,
  gender            TEXT CHECK (gender IN ('M','F','X')),
  height_cm         INT CHECK (height_cm BETWEEN 50 AND 260),
  weight_kg         INT CHECK (weight_kg BETWEEN 20 AND 400),
  activity_level    TEXT CHECK (activity_level IN ('SEDENTARY','LIGHT','MODERATE','ACTIVE','VERY_ACTIVE')),
  -- Warn, never hide (docs/02 D-23 #2): hiding a dish makes the menu look thin.
  allergen_profile  JSONB NOT NULL DEFAULT '[]'::jsonb,
  dislike_profile   JSONB NOT NULL DEFAULT '[]'::jsonb,
  preferred_locale  TEXT NOT NULL DEFAULT 'id-ID' CHECK (preferred_locale IN ('id-ID','en')),
  notify_email      BOOLEAN NOT NULL DEFAULT TRUE,
  notify_whatsapp   BOOLEAN NOT NULL DEFAULT TRUE,
  marketing_consent BOOLEAN NOT NULL DEFAULT FALSE,
  pdp_consent_at    TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX customer_type_idx ON customer (customer_type_id);
CREATE INDEX customer_org_idx  ON customer (organisation_id) WHERE organisation_id IS NOT NULL;

CREATE TABLE customer_address (
  id                UUID PRIMARY KEY,
  customer_id       UUID NOT NULL REFERENCES customer(id) ON DELETE CASCADE,
  label             TEXT NOT NULL,
  recipient_name    TEXT NOT NULL,
  recipient_phone   TEXT NOT NULL,
  address_line      TEXT NOT NULL,
  district          TEXT NOT NULL DEFAULT '',
  city              TEXT NOT NULL DEFAULT '',
  province          TEXT NOT NULL DEFAULT '',
  postal_code       TEXT NOT NULL DEFAULT '',
  -- Mandatory pin (PROMPT §8.2). The bounds are a coarse sanity envelope; the
  -- operating envelope is a sys_parameter checked in the application, because
  -- expanding to another city must not need a migration.
  latitude          NUMERIC(9,6) NOT NULL CHECK (latitude  BETWEEN -90  AND 90),
  longitude         NUMERIC(9,6) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  -- Generated, so the geometry can never disagree with the numbers.
  geom              GEOGRAPHY(Point,4326)
                    GENERATED ALWAYS AS (
                      ST_SetSRID(ST_MakePoint(longitude::float8, latitude::float8), 4326)::geography
                    ) STORED,
  google_place_id   TEXT,
  formatted_address TEXT,
  driver_note       TEXT NOT NULL DEFAULT '',
  is_default        BOOLEAN NOT NULL DEFAULT FALSE,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX customer_address_customer_idx ON customer_address (customer_id) WHERE is_active;
CREATE INDEX customer_address_geom_idx ON customer_address USING gist (geom);
-- Exactly one default per customer, enforced by the database rather than by a
-- read-modify-write in the application.
CREATE UNIQUE INDEX customer_address_one_default_idx
  ON customer_address (customer_id) WHERE is_default AND is_active;

CREATE TRIGGER customer_type_touch BEFORE UPDATE ON customer_type
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER organisation_touch BEFORE UPDATE ON organisation
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER customer_touch BEFORE UPDATE ON customer
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER customer_address_touch BEFORE UPDATE ON customer_address
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
