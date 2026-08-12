-- 0005 — delivery time slots and the menu calendar.
--
-- The MEAL is the unit of sale (docs/02 D-32, Steven 2026-08-12): several foods
-- compose a meal, the customer picks the meal, and one credit buys one meal
-- whatever it contains. So capacity, publication and the credit boundary all
-- attach to scheduled_meal — not to the individual food, as the brief's flat
-- food_schedule would have had it.

CREATE TABLE delivery_time_slot (
  id         UUID PRIMARY KEY,
  slot_time  TIME NOT NULL UNIQUE,
  alias      TEXT NOT NULL,
  -- The customer sees only the alias; the exact time is internal (PROMPT §8.1).
  sort_order INT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT FALSE,
  -- Per-slot cut-off overrides, so tuning dinner later is a settings change and
  -- not a migration (docs/03 Q-5). NULL falls back to the global parameters.
  cutoff_time      TIME,
  cutoff_lead_days INT CHECK (cutoff_lead_days >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by UUID REFERENCES app_user(id),
  -- A 15-minute grid (PROMPT §8.1).
  CONSTRAINT delivery_time_slot_grid CHECK (EXTRACT(MINUTE FROM slot_time)::int % 15 = 0
                                        AND EXTRACT(SECOND FROM slot_time) = 0)
);

CREATE TABLE scheduled_meal (
  id            UUID PRIMARY KEY,
  service_date  DATE NOT NULL,
  diet_type_id  UUID NOT NULL REFERENCES diet_type(id) ON DELETE RESTRICT,
  slot_id       UUID NOT NULL REFERENCES delivery_time_slot(id) ON DELETE RESTRICT,
  name          TEXT,
  description   TEXT NOT NULL DEFAULT '',
  hero_photo_key TEXT,
  -- NULL = unlimited. Counted in MEALS.
  qty_capacity  INT CHECK (qty_capacity IS NULL OR qty_capacity > 0),
  qty_reserved  INT NOT NULL DEFAULT 0 CHECK (qty_reserved >= 0),
  status        TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','PUBLISHED')),
  published_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by    UUID REFERENCES app_user(id),
  -- One meal per date + diet + slot.
  UNIQUE (service_date, diet_type_id, slot_id),
  -- The database refuses the oversell, not just the application (CLAUDE.md §4).
  CONSTRAINT scheduled_meal_no_oversell
    CHECK (qty_capacity IS NULL OR qty_reserved <= qty_capacity),
  CONSTRAINT scheduled_meal_published_has_time
    CHECK (status <> 'PUBLISHED' OR published_at IS NOT NULL)
);
CREATE INDEX scheduled_meal_calendar_idx
  ON scheduled_meal (service_date, slot_id, diet_type_id);
CREATE INDEX scheduled_meal_published_idx
  ON scheduled_meal (service_date, diet_type_id) WHERE status = 'PUBLISHED';

CREATE TABLE scheduled_meal_item (
  id                UUID PRIMARY KEY,
  scheduled_meal_id UUID NOT NULL REFERENCES scheduled_meal(id) ON DELETE CASCADE,
  food_id           UUID NOT NULL REFERENCES food(id) ON DELETE RESTRICT,
  item_role         TEXT NOT NULL DEFAULT 'MAIN'
                    CHECK (item_role IN ('MAIN','SIDE','DESSERT','DRINK')),
  sort_order        INT NOT NULL DEFAULT 0,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- The same dish cannot be scheduled twice in one sitting (PROMPT §4.4).
  UNIQUE (scheduled_meal_id, food_id)
);
CREATE INDEX scheduled_meal_item_meal_idx ON scheduled_meal_item (scheduled_meal_id, sort_order);
CREATE INDEX scheduled_meal_item_food_idx ON scheduled_meal_item (food_id);

CREATE TRIGGER delivery_time_slot_touch BEFORE UPDATE ON delivery_time_slot
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER scheduled_meal_touch BEFORE UPDATE ON scheduled_meal
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
