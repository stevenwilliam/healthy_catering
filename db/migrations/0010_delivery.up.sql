-- 0010 — deliveries and their lines.
--
-- The order is the commercial event; the delivery is the fulfilment event
-- (docs/02 D-15). One package order produces N deliveries over weeks, to
-- different addresses, through different kitchens — so routing runs per
-- delivery, never per order (PROMPT §9.3).

CREATE TABLE delivery (
  id                  UUID PRIMARY KEY,
  delivery_code       TEXT NOT NULL UNIQUE,
  customer_id         UUID NOT NULL REFERENCES customer(id) ON DELETE RESTRICT,
  order_id            UUID REFERENCES customer_order(id) ON DELETE SET NULL,
  customer_package_id UUID REFERENCES customer_package(id) ON DELETE SET NULL,

  service_date        DATE NOT NULL,
  slot_id             UUID NOT NULL REFERENCES delivery_time_slot(id) ON DELETE RESTRICT,

  address_id          UUID REFERENCES customer_address(id) ON DELETE SET NULL,
  -- Snapshotted at confirmation: editing a saved address later must not
  -- rewrite delivery history (PROMPT §8.3).
  address_snapshot    JSONB NOT NULL,
  latitude            NUMERIC(9,6) NOT NULL,
  longitude           NUMERIC(9,6) NOT NULL,
  geom                GEOGRAPHY(Point,4326)
                      GENERATED ALWAYS AS (
                        ST_SetSRID(ST_MakePoint(longitude::float8, latitude::float8), 4326)::geography
                      ) STORED,

  kitchen_id          UUID REFERENCES kitchen(id) ON DELETE SET NULL,
  assigned_distance_m INT CHECK (assigned_distance_m >= 0),
  assignment_mode     TEXT NOT NULL DEFAULT 'AUTO' CHECK (assignment_mode IN ('AUTO','MANUAL')),
  -- "nearest covering kitchen, 3.2 km" — staff will be asked why this went to
  -- Kitchen B and must be able to answer (PROMPT §9.3).
  assignment_reason   TEXT NOT NULL DEFAULT '',
  assigned_at         TIMESTAMPTZ,

  delivery_fee_idr    BIGINT NOT NULL DEFAULT 0 CHECK (delivery_fee_idr >= 0),
  status              TEXT NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN
                        ('SCHEDULED','PREPARING','OUT_FOR_DELIVERY','DELIVERED',
                         'FAILED','SKIPPED','CANCELLED')),
  prepared_at         TIMESTAMPTZ,
  dispatched_at       TIMESTAMPTZ,
  delivered_at        TIMESTAMPTZ,
  delivered_by        UUID REFERENCES app_user(id),
  failure_reason      TEXT,
  skip_reason         TEXT,
  driver_note         TEXT NOT NULL DEFAULT '',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT delivery_has_a_source CHECK (order_id IS NOT NULL OR customer_package_id IS NOT NULL),
  CONSTRAINT delivery_failure_has_reason
    CHECK (status <> 'FAILED' OR (failure_reason IS NOT NULL AND failure_reason <> ''))
);
CREATE INDEX delivery_manifest_idx ON delivery (kitchen_id, service_date, slot_id, status);
CREATE INDEX delivery_customer_idx ON delivery (customer_id, service_date DESC);
CREATE INDEX delivery_date_idx     ON delivery (service_date, slot_id);
CREATE INDEX delivery_package_idx  ON delivery (customer_package_id) WHERE customer_package_id IS NOT NULL;
CREATE INDEX delivery_geom_idx     ON delivery USING gist (geom);

CREATE TABLE delivery_line (
  id                UUID PRIMARY KEY,
  delivery_id       UUID NOT NULL REFERENCES delivery(id) ON DELETE CASCADE,
  -- One line = one MEAL (docs/02 D-32).
  scheduled_meal_id UUID REFERENCES scheduled_meal(id) ON DELETE SET NULL,
  order_line_id     UUID REFERENCES order_line(id) ON DELETE SET NULL,
  diet_type_id      UUID REFERENCES diet_type(id) ON DELETE SET NULL,
  qty               INT NOT NULL DEFAULT 1 CHECK (qty > 0),
  -- The foods, their roles and their nutrition as at confirmation. The packing
  -- label and the production sheet print from this, so a later menu
  -- substitution cannot silently change what was cooked.
  meal_snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX delivery_line_delivery_idx ON delivery_line (delivery_id);
CREATE INDEX delivery_line_meal_idx     ON delivery_line (scheduled_meal_id);

CREATE TRIGGER delivery_touch BEFORE UPDATE ON delivery
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
