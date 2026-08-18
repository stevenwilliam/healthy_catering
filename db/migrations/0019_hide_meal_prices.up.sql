-- Whether the public price list shows per-portion prices.
--
-- Steven, 2026-08-18: hide them. A setting rather than deleted markup, because
-- "do we publish our per-meal rate" is a commercial decision that will be
-- revisited — and CLAUDE.md §7 puts anything that can change without a deploy
-- in this table. Package prices are unaffected and stay visible.
--
-- Turning it back on is a value change on the settings screen, not a release.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c31','public.show_meal_prices','false','bool','public','Show per-portion prices',
  'When off, the public price list hides the per-portion price table and invites the visitor to ask for a quote instead. Package prices are shown either way.',FALSE,FALSE,2);
