-- An "Order" entry in the header, pointing at the ordering app
-- (Steven, 2026-08-19).
--
-- Two new columns, because this item breaks two assumptions the table was
-- built on:
--
--   is_localised — every path so far was a public page, so the template ran it
--     through the locale prefixer and /en/price-list followed for free. The
--     app is NOT locale-prefixed: it lives at /app and carries its own
--     language preference. Prefixing it would produce /en/app/menu, which is a
--     404. A column rather than "if the path starts with /app" in the
--     template, because that is a rule and rules belong in data here.
--
--   icon — a cart is recognised faster than it is read, and this is the one
--     item where that matters. Named rather than a path, so the icon ships
--     with the CSS and cannot 404.
ALTER TABLE nav_item
  ADD COLUMN is_localised BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN icon         TEXT    NOT NULL DEFAULT '';

COMMENT ON COLUMN nav_item.is_localised IS
  'FALSE for links outside the locale-prefixed public site, e.g. /app.';

-- sort_order 10 so it sits at the end of the row, where a cart is looked for.
INSERT INTO nav_item
  (id, key, kind, path, label_key, active_key, sort_order, is_visible, is_localised, icon)
VALUES
 ('00000000-0000-7000-8000-000000000e09','order','LINK','/app/menu','nav.order','order',
  10, TRUE, FALSE, 'cart')
ON CONFLICT (key) DO NOTHING;
