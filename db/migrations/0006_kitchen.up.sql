-- 0006 — kitchens, service areas, capacity and the coverage log.
--
-- Every delivery is routed to exactly one kitchen from the address coordinates
-- (PROMPT §9). Steven, 2026-08-12: "auto assign location customer to nearest
-- kitchen" — so every kitchen seeds at the same priority and the ranking
-- collapses to nearest-first, with priority left as the manual override.

CREATE TABLE kitchen (
  id                UUID PRIMARY KEY,
  code              TEXT NOT NULL UNIQUE,
  name              TEXT NOT NULL,
  address_line      TEXT NOT NULL,
  district          TEXT NOT NULL DEFAULT '',
  city              TEXT NOT NULL DEFAULT '',
  latitude          NUMERIC(9,6) NOT NULL CHECK (latitude  BETWEEN -90  AND 90),
  longitude         NUMERIC(9,6) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  geom              GEOGRAPHY(Point,4326)
                    GENERATED ALWAYS AS (
                      ST_SetSRID(ST_MakePoint(longitude::float8, latitude::float8), 4326)::geography
                    ) STORED,
  -- Radius mode. Overridden by service_area when that is present (PROMPT §9.2).
  service_radius_km NUMERIC(6,2) NOT NULL DEFAULT 10 CHECK (service_radius_km > 0),
  service_area      GEOGRAPHY(Polygon,4326),
  phone             TEXT,
  pic_name          TEXT NOT NULL DEFAULT '',
  pic_phone         TEXT,
  opens_at          TIME NOT NULL DEFAULT '06:00',
  closes_at         TIME NOT NULL DEFAULT '21:00',
  default_daily_capacity INT CHECK (default_daily_capacity IS NULL OR default_daily_capacity > 0),
  default_slot_capacity  INT CHECK (default_slot_capacity  IS NULL OR default_slot_capacity  > 0),
  priority          INT NOT NULL DEFAULT 100,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  notes             TEXT NOT NULL DEFAULT '',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by        UUID REFERENCES app_user(id),
  CONSTRAINT kitchen_hours CHECK (closes_at > opens_at)
);
CREATE INDEX kitchen_geom_idx ON kitchen USING gist (geom);
CREATE INDEX kitchen_area_idx ON kitchen USING gist (service_area) WHERE service_area IS NOT NULL;
CREATE INDEX kitchen_active_idx ON kitchen (is_active, priority) WHERE is_active;

-- Deferred from 0002: staff can be scoped to one kitchen (docs/02 D-21).
ALTER TABLE staff_profile
  ADD CONSTRAINT staff_profile_kitchen_fk
  FOREIGN KEY (kitchen_id) REFERENCES kitchen(id) ON DELETE SET NULL;
CREATE INDEX staff_profile_kitchen_idx ON staff_profile (kitchen_id) WHERE kitchen_id IS NOT NULL;

CREATE TABLE kitchen_slot (
  kitchen_id UUID NOT NULL REFERENCES kitchen(id) ON DELETE CASCADE,
  slot_id    UUID NOT NULL REFERENCES delivery_time_slot(id) ON DELETE CASCADE,
  PRIMARY KEY (kitchen_id, slot_id)
);

-- ISO weekday: 1 = Monday … 7 = Sunday.
CREATE TABLE kitchen_operating_day (
  kitchen_id UUID NOT NULL REFERENCES kitchen(id) ON DELETE CASCADE,
  weekday    INT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
  PRIMARY KEY (kitchen_id, weekday)
);

-- Per date + slot, inheriting the kitchen default when absent.
CREATE TABLE kitchen_capacity (
  id            UUID PRIMARY KEY,
  kitchen_id    UUID NOT NULL REFERENCES kitchen(id) ON DELETE CASCADE,
  service_date  DATE NOT NULL,
  slot_id       UUID NOT NULL REFERENCES delivery_time_slot(id) ON DELETE RESTRICT,
  max_portions  INT NOT NULL CHECK (max_portions >= 0),
  reserved_portions INT NOT NULL DEFAULT 0 CHECK (reserved_portions >= 0),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by    UUID REFERENCES app_user(id),
  UNIQUE (kitchen_id, service_date, slot_id),
  -- Again: the database refuses the oversell even under a race.
  CONSTRAINT kitchen_capacity_no_oversell CHECK (reserved_portions <= max_portions)
);
CREATE INDEX kitchen_capacity_date_idx ON kitchen_capacity (service_date, slot_id);

-- Where demand exists that we cannot serve. This log is the map of where to
-- open the next kitchen (PROMPT §9.3.5).
CREATE TABLE out_of_range_attempt (
  id           UUID PRIMARY KEY,
  customer_id  UUID REFERENCES customer(id) ON DELETE SET NULL,
  latitude     NUMERIC(9,6) NOT NULL,
  longitude    NUMERIC(9,6) NOT NULL,
  geom         GEOGRAPHY(Point,4326)
               GENERATED ALWAYS AS (
                 ST_SetSRID(ST_MakePoint(longitude::float8, latitude::float8), 4326)::geography
               ) STORED,
  district     TEXT NOT NULL DEFAULT '',
  city         TEXT NOT NULL DEFAULT '',
  slot_id      UUID REFERENCES delivery_time_slot(id) ON DELETE SET NULL,
  service_date DATE,
  source       TEXT NOT NULL DEFAULT 'WIDGET'
               CHECK (source IN ('WIDGET','ADDRESS_FORM','CHECKOUT')),
  nearest_kitchen_id UUID REFERENCES kitchen(id) ON DELETE SET NULL,
  nearest_distance_m INT,
  notify_requested   BOOLEAN NOT NULL DEFAULT FALSE,
  notify_email       CITEXT,
  occurred_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX out_of_range_district_idx ON out_of_range_attempt (district, occurred_at DESC);
CREATE INDEX out_of_range_geom_idx ON out_of_range_attempt USING gist (geom);

-- Ready for the Distance Matrix upgrade (docs/02 D-18): the resolver swaps,
-- the schema does not change.
CREATE TABLE kitchen_travel_cache (
  kitchen_id  UUID NOT NULL REFERENCES kitchen(id) ON DELETE CASCADE,
  address_id  UUID NOT NULL REFERENCES customer_address(id) ON DELETE CASCADE,
  distance_m  INT NOT NULL,
  duration_s  INT NOT NULL,
  fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (kitchen_id, address_id)
);

CREATE TRIGGER kitchen_touch BEFORE UPDATE ON kitchen
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER kitchen_capacity_touch BEFORE UPDATE ON kitchen_capacity
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
