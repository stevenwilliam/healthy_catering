-- 0004 — diet types, allergens, foods and their nutrition.
--
-- Nutrition lives on the FOOD. A meal's panel is the sum of its foods' panels
-- (docs/02 D-33), which is why every field is an integer: summing integers is
-- exact where summing decimals drifts.

CREATE TABLE diet_type (
  id             UUID PRIMARY KEY,
  name           TEXT NOT NULL UNIQUE,
  slug           TEXT NOT NULL UNIQUE,
  description    TEXT NOT NULL DEFAULT '',
  hero_image_key TEXT,
  seo_title      TEXT,
  seo_description TEXT,
  has_subtypes   BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order     INT NOT NULL DEFAULT 0,
  is_active      BOOLEAN NOT NULL DEFAULT TRUE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by     UUID REFERENCES app_user(id)
);

-- Descriptive only: the schedule and all four price tables key on diet_type
-- alone, or the price matrix multiplies by subtype (docs/03 Q-6).
CREATE TABLE diet_subtype (
  id           UUID PRIMARY KEY,
  diet_type_id UUID NOT NULL REFERENCES diet_type(id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  slug         TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  sort_order   INT NOT NULL DEFAULT 0,
  is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (diet_type_id, slug)
);

CREATE TABLE allergen (
  id         UUID PRIMARY KEY,
  code       TEXT NOT NULL UNIQUE,
  name_id    TEXT NOT NULL,
  name_en    TEXT NOT NULL,
  icon       TEXT,
  sort_order INT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE food (
  id           UUID PRIMARY KEY,
  name         TEXT NOT NULL,
  slug         TEXT NOT NULL UNIQUE,
  description  TEXT NOT NULL DEFAULT '',
  portion_size TEXT NOT NULL DEFAULT '',
  is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by   UUID REFERENCES app_user(id)
);
CREATE INDEX food_active_idx ON food (is_active, name);

CREATE TABLE food_photo (
  id          UUID PRIMARY KEY,
  food_id     UUID NOT NULL REFERENCES food(id) ON DELETE CASCADE,
  object_key  TEXT NOT NULL,
  alt_text    TEXT NOT NULL DEFAULT '',
  width       INT,
  height      INT,
  bytes       BIGINT,
  content_type TEXT,
  sort_order  INT NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX food_photo_food_idx ON food_photo (food_id, sort_order);

-- Integers only: kcal whole, everything else in milligrams (docs/02 D-24).
CREATE TABLE food_nutrition (
  food_id           UUID PRIMARY KEY REFERENCES food(id) ON DELETE CASCADE,
  calories_kcal     INT NOT NULL DEFAULT 0 CHECK (calories_kcal     >= 0),
  protein_mg        INT NOT NULL DEFAULT 0 CHECK (protein_mg        >= 0),
  fat_mg            INT NOT NULL DEFAULT 0 CHECK (fat_mg            >= 0),
  saturated_fat_mg  INT NOT NULL DEFAULT 0 CHECK (saturated_fat_mg  >= 0),
  carbohydrate_mg   INT NOT NULL DEFAULT 0 CHECK (carbohydrate_mg   >= 0),
  sugar_mg          INT NOT NULL DEFAULT 0 CHECK (sugar_mg          >= 0),
  fibre_mg          INT NOT NULL DEFAULT 0 CHECK (fibre_mg          >= 0),
  sodium_mg         INT NOT NULL DEFAULT 0 CHECK (sodium_mg         >= 0),
  cholesterol_mg    INT NOT NULL DEFAULT 0 CHECK (cholesterol_mg    >= 0),
  extras            JSONB NOT NULL DEFAULT '{}'::jsonb,
  -- A food with no panel must not silently under-report the meal it belongs to.
  is_complete       BOOLEAN NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by        UUID REFERENCES app_user(id)
);

CREATE TABLE food_diet_type (
  food_id      UUID NOT NULL REFERENCES food(id) ON DELETE CASCADE,
  diet_type_id UUID NOT NULL REFERENCES diet_type(id) ON DELETE CASCADE,
  PRIMARY KEY (food_id, diet_type_id)
);

CREATE TABLE food_allergen (
  food_id     UUID NOT NULL REFERENCES food(id) ON DELETE CASCADE,
  allergen_id UUID NOT NULL REFERENCES allergen(id) ON DELETE CASCADE,
  PRIMARY KEY (food_id, allergen_id)
);

CREATE TRIGGER diet_type_touch BEFORE UPDATE ON diet_type
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER diet_subtype_touch BEFORE UPDATE ON diet_subtype
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER food_touch BEFORE UPDATE ON food
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER food_nutrition_touch BEFORE UPDATE ON food_nutrition
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
