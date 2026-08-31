-- Back to the 2026-08-19 arrangement: Beranda | Menu, and "Pesan".
UPDATE nav_item SET label_key = 'nav.menu', sort_order = 1, is_visible = TRUE
 WHERE key = 'menu';
UPDATE nav_item SET label_key = 'nav.benefits', sort_order = 3, is_visible = FALSE
 WHERE key = 'benefits';
UPDATE nav_item SET label_key = 'nav.pricelist', sort_order = 2, is_visible = FALSE
 WHERE key = 'pricelist';
UPDATE nav_item SET label_key = 'nav.contact', sort_order = 4, is_visible = FALSE
 WHERE key = 'contact';
DELETE FROM nav_item WHERE key = 'areas';
UPDATE nav_item SET is_visible = TRUE WHERE key IN ('home', 'order');
