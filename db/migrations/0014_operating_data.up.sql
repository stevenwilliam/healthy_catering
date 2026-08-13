-- 0014 — Steven's answers of 2026-08-13 (second batch): the real operating data.
--
-- This is the migration that stops the system telling convincing lies. Until
-- now routing answered from two PLACEHOLDER kitchens with real Jakarta
-- coordinates, and a customer in Bekasi was told "Dilayani oleh Evermore
-- Kitchen Selatan" with a straight face.
--
-- Every coordinate below came from the Google Geocoding API using the real
-- key, not from memory. The formatted address each one resolved to is recorded
-- beside it so a wrong pin is arguable from the file rather than only from a map.

-- ── 1. Kitchens ──────────────────────────────────────────────────────────────
--
-- The placeholders are DEACTIVATED, not deleted: nine deliveries already
-- reference them, and destroying history to tidy a seed row is the wrong trade.
-- They stop being routing candidates the moment is_active goes false.
UPDATE kitchen
   SET is_active = FALSE,
       notes = notes || ' Superseded by the real kitchens in migration 0014.',
       updated_at = now()
 WHERE code IN ('JKT-S','JKT-P');

-- Steven: "find and use location from Siloam Lippo Village, Siloam MRCCC,
-- Ruuma Cideng, Siloam TB Simatupang, Siloam Kebon Jeruk", radius 20 km for
-- all, no daily capacity, one shared phone, open every day with no closing time.
--
-- default_daily_capacity stays NULL deliberately. It is not the oversell
-- control — that is scheduled_meal.qty_capacity, per date + diet + slot, which
-- is the row a booking actually takes a lock on. A number here would look like
-- a limit while enforcing nothing.
--
-- opens_at/closes_at span the whole day because Steven asked for no closing
-- time. Nothing reads these columns yet; the 18:00 order cut-off is a
-- sys_parameter and is unaffected by them.
--
-- Equal priority (100) throughout, so ranking collapses to nearest-first (D-34).
INSERT INTO kitchen (id, code, name, address_line, district, city,
                     latitude, longitude, service_radius_km,
                     phone, pic_name, pic_phone,
                     opens_at, closes_at,
                     default_daily_capacity, default_slot_capacity,
                     priority, is_active, notes) VALUES
 ('00000000-0000-7000-8000-000000000c01','LPV','Evermore Lippo Village',
  'Jl. Jend. Sudirman No.6, Bencongan, Kelapa Dua','Kelapa Dua','Kabupaten Tangerang',
  -6.225232, 106.597961, 20,
  '08176315568','', '08176315568',
  '00:00','23:59', NULL, NULL, 100, TRUE,
  'Geocoded 2026-08-13: "Siloam Lippo Karawaci, Jl. Jend. Sudirman No.6, Bencongan, Kecamatan Kelapa Dua, Kabupaten Tangerang, Banten 15810".'),

 ('00000000-0000-7000-8000-000000000c02','MRCCC','Evermore MRCCC Semanggi',
  'Jl. Garnisun 1 No.2-3, Karet Semanggi, Setiabudi','Setiabudi','Jakarta Selatan',
  -6.219054, 106.817257, 20,
  '08176315568','', '08176315568',
  '00:00','23:59', NULL, NULL, 100, TRUE,
  'Geocoded 2026-08-13: "Jalan Garnisun 1 No.2-3 5, RT.5/RW.4, Karet Semanggi, Kecamatan Setiabudi, Kota Jakarta Selatan 12930".'),

 ('00000000-0000-7000-8000-000000000c03','CDG','Evermore Cideng',
  'Jl. Tanah Abang II No.70B, Cideng, Gambir','Gambir','Jakarta Pusat',
  -6.175857, 106.811705, 20,
  '08176315568','', '08176315568',
  '00:00','23:59', NULL, NULL, 100, TRUE,
  'Geocoded 2026-08-13: "Jl. Tanah Abang II.70B, RT.1/RW.1, Cideng, Kecamatan Gambir, Kota Jakarta Pusat 10150". Ruuma Cideng.'),

 ('00000000-0000-7000-8000-000000000c04','TBS','Evermore TB Simatupang',
  'Jl. R.A. Kartini No.8, Cilandak Barat, Cilandak','Cilandak','Jakarta Selatan',
  -6.292634, 106.784291, 20,
  '08176315568','', '08176315568',
  '00:00','23:59', NULL, NULL, 100, TRUE,
  'Geocoded 2026-08-13: "Jl. R.A. Kartini No.8, RT.10/RW.4, Cilandak Bar., Kec. Cilandak, Kota Jakarta Selatan 12430".'),

 ('00000000-0000-7000-8000-000000000c05','KJ','Evermore Kebon Jeruk',
  'Jl. Perjuangan Kav.8, Kebon Jeruk','Kebon Jeruk','Jakarta Barat',
  -6.190825, 106.763625, 20,
  '08176315568','', '08176315568',
  '00:00','23:59', NULL, NULL, 100, TRUE,
  'Geocoded 2026-08-13: "Jl. Perjuangan No.Kav.8, RT.14/RW.10, Kb. Jeruk, Kec. Kb. Jeruk, Kota Jakarta Barat 11530".');

-- Every active delivery slot, at every new kitchen.
INSERT INTO kitchen_slot (kitchen_id, slot_id)
  SELECT k.id, s.id
    FROM kitchen k CROSS JOIN delivery_time_slot s
   WHERE k.code IN ('LPV','MRCCC','CDG','TBS','KJ') AND s.is_active
     AND NOT EXISTS (SELECT 1 FROM kitchen_slot ks
                      WHERE ks.kitchen_id = k.id AND ks.slot_id = s.id);

-- Open every day. ISO weekdays, 1 = Monday … 7 = Sunday.
INSERT INTO kitchen_operating_day (kitchen_id, weekday)
  SELECT k.id, d
    FROM kitchen k CROSS JOIN generate_series(1,7) AS d
   WHERE k.code IN ('LPV','MRCCC','CDG','TBS','KJ')
     AND NOT EXISTS (SELECT 1 FROM kitchen_operating_day o
                      WHERE o.kitchen_id = k.id AND o.weekday = d);

-- ── 2. Bank account ──────────────────────────────────────────────────────────
--
-- The dummy is deactivated rather than deleted for the same reason as the
-- kitchens: existing orders were quoted against it, and a payment instruction
-- that silently rewrites itself is worse than one that is visibly historical.
UPDATE bank_account
   SET is_active = FALSE, updated_at = now()
 WHERE account_number = '1234567890';

INSERT INTO bank_account (id, bank_name, account_number, account_holder, branch,
                          is_active, sort_order)
VALUES ('00000000-0000-7000-8000-000000000d01',
        'Nobu', '16830226665', 'PT Sunshine Food International',
        'Menara Matahari', TRUE, 1)
ON CONFLICT (bank_name, account_number) DO UPDATE
   SET account_holder = EXCLUDED.account_holder,
       branch         = EXCLUDED.branch,
       is_active      = TRUE,
       updated_at     = now();

-- ── 3. The company itself ────────────────────────────────────────────────────
--
-- Evermore is the customer-facing brand; PT Sunshine Food International is the
-- legal entity that issues the invoice and owns the bank account. Both appear
-- in the product, in different places, and conflating them is how a transfer
-- ends up queried at the bank.
UPDATE sys_parameters SET value = 'PT Sunshine Food International', updated_at = now()
 WHERE key = 'company.legal_name';

UPDATE sys_parameters SET value = 'Menara Matahari 2nd Floor
Jl Boulevard Palem Raya No.07
Lippo Karawaci, Kab.Tangerang
Banten 15810 - Indonesia', updated_at = now()
 WHERE key = 'company.address';

UPDATE sys_parameters SET value = '08176315568', updated_at = now()
 WHERE key IN ('company.phone', 'company.whatsapp');

-- ⚠️ 123 123 123 is a PLACEHOLDER. A real NPWP is 15 digits (16 since the NIK
-- migration), so this cannot be the registered number. It is stored because
-- Steven asked for it and it is editable from the back office, but it must not
-- reach a faktur pajak. See docs/03-open-questions.md Q-24.
UPDATE sys_parameters SET value = '123 123 123', updated_at = now()
 WHERE key = 'company.npwp';

-- ── 4. WhatsApp ──────────────────────────────────────────────────────────────
--
-- Steven: "use waha 08176315568". The WAHA container on this host is SHARED
-- with ruuma (ruuma D11) and its session is already bound to 628176315568 —
-- the same number — so the credentials point at the existing instance rather
-- than standing a second one up.
--
-- ⚠️ That session reported status=FAILED when this migration was written, which
-- means WhatsApp will NOT send until it is re-linked by scanning a QR code.
-- The channel is still switched on: the job queue retries with backoff, so
-- messages queue rather than vanish, and they flow the moment the session is
-- healthy. See docs/RUN-WHEN-BACK.md §8.
UPDATE sys_parameters SET value = 'http://127.0.0.1:3000', updated_at = now()
 WHERE key = 'whatsapp.waha_url';

UPDATE sys_parameters SET value = 'default', updated_at = now()
 WHERE key = 'whatsapp.waha_session';

UPDATE sys_parameters SET value = 'a69cce02eb36426ca214de1a17a4ba86', updated_at = now()
 WHERE key = 'whatsapp.waha_api_key';

UPDATE sys_parameters SET value = 'true', updated_at = now()
 WHERE key = 'notify.whatsapp_enabled';
