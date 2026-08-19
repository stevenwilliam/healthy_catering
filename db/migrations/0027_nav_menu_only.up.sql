-- The header is reduced to a single "Menu" entry (Steven, 2026-08-19).
--
-- Two instructions, in order: hide Price list, Benefits, Contact, About and
-- Career; then hide the Category dropdown too and show only Healthy, labelled
-- "Menu".
--
-- Recorded as a migration rather than typed into this one database, so any
-- other environment comes up looking the same. It is only DATA — every hidden
-- row is still there, and turning one back on is a toggle on the admin screen,
-- not a release.
--
-- The pages themselves are untouched: /price-list, /benefits, /contact,
-- /about and /career still exist, still render in three languages and are
-- still in the sitemap. Only the header stops linking to them.
INSERT INTO nav_item (id, key, kind, path, label_key, active_key, sort_order, is_visible) VALUES
 ('00000000-0000-7000-8000-000000000e07','menu','LINK','/menu/healthy','nav.menu','category', 1, TRUE)
ON CONFLICT (key) DO NOTHING;

UPDATE nav_item SET is_visible = FALSE
 WHERE key IN ('category','pricelist','benefits','contact','about','career');
