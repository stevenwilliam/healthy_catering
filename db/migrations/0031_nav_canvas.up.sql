-- The masthead navigation the "Evermore Mockups" canvas draws (artboard M1).
--
-- Steven, 2026-08-31: the canvas is the specification. Its header carries five
-- items — Menu minggu ini · Cara kerja · Paket · Untuk kantor · Wilayah antar
-- — which SUPERSEDES the reduction to "Beranda | Menu" made on 2026-08-19
-- (migrations 0027, 0028). The newer, more specific decision wins.
--
-- The nav stays DATA, as migration 0026 made it: labels come from the message
-- catalogue by key, so nothing here is typed wording that would exist in one
-- language on a trilingual site.
--
-- Two of the five have no page of their own, and this migration does NOT
-- invent one. "Untuk kantor" points at the contact page, which is where a
-- corporate enquiry already lands; "Wilayah antar" points at the home page's
-- coverage checker. Both are real destinations. What a dedicated corporate
-- page should SAY is a business question, and it is raised in
-- docs/03-open-questions.md rather than written here.

-- Menu keeps its path, gains the canvas's wording.
UPDATE nav_item SET label_key = 'nav.menu_week', sort_order = 1, is_visible = TRUE
 WHERE key = 'menu';

-- "Cara kerja" is the benefits page: it is what the page already explains.
UPDATE nav_item SET label_key = 'nav.how', sort_order = 2, is_visible = TRUE
 WHERE key = 'benefits';

-- "Paket" is the price list, which is where the packages are.
UPDATE nav_item SET label_key = 'nav.packages', sort_order = 3, is_visible = TRUE
 WHERE key = 'pricelist';

UPDATE nav_item SET label_key = 'nav.corporate', sort_order = 4, is_visible = TRUE
 WHERE key = 'contact';

INSERT INTO nav_item (id, key, kind, path, label_key, active_key, sort_order, is_visible)
VALUES (gen_random_uuid(), 'areas', 'LINK', '/#check', 'nav.areas', 'home', 5, TRUE)
ON CONFLICT (key) DO UPDATE
   SET path = EXCLUDED.path, label_key = EXCLUDED.label_key,
       sort_order = EXCLUDED.sort_order, is_visible = TRUE;

-- The home link and the "Pesan" CTA leave the nav row: M1 gives the wordmark
-- the home link and puts the order action in the masthead's button pair.
UPDATE nav_item SET is_visible = FALSE WHERE key IN ('home', 'order');
