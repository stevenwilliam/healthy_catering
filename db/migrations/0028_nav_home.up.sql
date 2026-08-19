-- A Home entry in the header (Steven, 2026-08-19), first in the order.
--
-- sort_order 0 so it leads without renumbering anything else. The wordmark
-- already links home, but a named Home item is what people look for, and the
-- two are not the same affordance.
INSERT INTO nav_item (id, key, kind, path, label_key, active_key, sort_order, is_visible) VALUES
 ('00000000-0000-7000-8000-000000000e08','home','LINK','/','nav.home','home', 0, TRUE)
ON CONFLICT (key) DO NOTHING;
