-- The header menu becomes configurable (Steven, 2026-08-19): which items show,
-- and in what order.
--
-- A table rather than a JSON parameter, so it gets the same treatment as every
-- other master list — a searchable admin grid, per-row attribution, audit, and
-- a CSV export (99 §8).
--
-- What is NOT stored here is the LABEL. The wording stays in the message
-- catalogue keyed by label_key, because the site is trilingual and a label
-- typed into an admin box would exist in one language only. Renaming an item
-- is therefore a catalogue change; showing, hiding and reordering are data.
--
-- kind exists because one of these is not a link: CATEGORY renders the diet-type
-- dropdown. Without it the template would have to special-case a magic key,
-- which is the same thing with the rule hidden in the markup.
CREATE TABLE nav_item (
  id         UUID PRIMARY KEY,
  key        TEXT NOT NULL UNIQUE,
  kind       TEXT NOT NULL CHECK (kind IN ('LINK','CATEGORY')),
  -- Locale-free path, e.g. '/price-list'. The template runs it through the
  -- locale prefixer, so /en/price-list and /zh/price-list follow for free.
  path       TEXT NOT NULL DEFAULT '',
  label_key  TEXT NOT NULL,
  -- Which PageData.Active value marks this item as the current page.
  active_key TEXT NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  is_visible BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by UUID REFERENCES app_user(id),
  CONSTRAINT nav_item_link_has_path CHECK (kind <> 'LINK' OR length(btrim(path)) > 0)
);
CREATE INDEX nav_item_visible_idx ON nav_item (is_visible, sort_order);

CREATE TRIGGER nav_item_touch BEFORE UPDATE ON nav_item
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Seeded in the order the header currently renders, so switching the template
-- over to this table changes nothing until someone edits it.
INSERT INTO nav_item (id, key, kind, path, label_key, active_key, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000e01','category','CATEGORY','', 'nav.category', 'category', 1),
 ('00000000-0000-7000-8000-000000000e02','pricelist','LINK','/price-list','nav.pricelist','pricelist', 2),
 ('00000000-0000-7000-8000-000000000e03','benefits','LINK','/benefits','nav.benefits','benefits', 3),
 ('00000000-0000-7000-8000-000000000e04','contact','LINK','/contact','nav.contact','contact', 4),
 ('00000000-0000-7000-8000-000000000e05','about','LINK','/about','nav.about','about', 5),
 ('00000000-0000-7000-8000-000000000e06','career','LINK','/career','nav.career','career', 6);
