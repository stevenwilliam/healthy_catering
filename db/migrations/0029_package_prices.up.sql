-- Package prices (Steven, 2026-08-19): 10 days = Rp 500.000,
-- 30 days = Rp 1.200.000. Paket 20 already had Rp 900.000.
--
-- Whole rupiah in BIGINT, as everything touching money is (CLAUDE.md §4).
-- DEFAULT scope — customer_type_id NULL — so these are the public prices; a
-- negotiated corporate rate is a separate row and never reaches the price page.
--
-- valid_from 2026-01-01 to match the existing Paket 20 row rather than
-- CURRENT_DATE, so all three prices share one effective date and the price
-- page cannot show two of them live and one pending. The exclusion constraint
-- on (scope_key, package_id, validity) is what stops a second overlapping row
-- for the same package, so this INSERT is safe to re-run only because the
-- WHERE NOT EXISTS below makes it a no-op the second time.
--
-- Ongoing repricing belongs on the admin pricing screens, not in migrations.
-- This is here to set the starting point on every environment.
INSERT INTO package_price_normal (id, customer_type_id, package_id, price_idr, valid_from)
SELECT '00000000-0000-7000-8000-000000000901', NULL,
       '00000000-0000-7000-8000-000000000801', 500000, DATE '2026-01-01'
 WHERE NOT EXISTS (
   SELECT 1 FROM package_price_normal
    WHERE package_id = '00000000-0000-7000-8000-000000000801'
      AND customer_type_id IS NULL AND is_active);

INSERT INTO package_price_normal (id, customer_type_id, package_id, price_idr, valid_from)
SELECT '00000000-0000-7000-8000-000000000903', NULL,
       '00000000-0000-7000-8000-000000000803', 1200000, DATE '2026-01-01'
 WHERE NOT EXISTS (
   SELECT 1 FROM package_price_normal
    WHERE package_id = '00000000-0000-7000-8000-000000000803'
      AND customer_type_id IS NULL AND is_active);
