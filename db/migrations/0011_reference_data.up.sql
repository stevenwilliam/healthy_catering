-- 0011 — reference data and parameters.
--
-- Seeded, not hard-coded: every value here is a row an admin can change
-- (CLAUDE.md §7). Where a value is a placeholder awaiting Steven, it says so in
-- the description so nobody mistakes an invented number for a decided one.
--
-- UUIDs are fixed rather than generated so seeding is idempotent and a
-- reference in another migration or a test stays valid.

-- ── permissions ──────────────────────────────────────────────────────────────
INSERT INTO permission (id, code, description) VALUES
 ('00000000-0000-7000-8000-000000000101','customer.read','View customers'),
 ('00000000-0000-7000-8000-000000000102','customer.write','Create and edit customers'),
 ('00000000-0000-7000-8000-000000000103','customer.type.change','Change a customer type (audit-logged)'),
 ('00000000-0000-7000-8000-000000000104','organisation.manage','Manage organisations'),
 ('00000000-0000-7000-8000-000000000105','catalogue.read','View diet types and foods'),
 ('00000000-0000-7000-8000-000000000106','catalogue.write','Manage diet types, foods and nutrition'),
 ('00000000-0000-7000-8000-000000000107','schedule.read','View the menu calendar'),
 ('00000000-0000-7000-8000-000000000108','schedule.write','Build and publish the menu calendar'),
 ('00000000-0000-7000-8000-000000000109','price.read','View prices'),
 ('00000000-0000-7000-8000-00000000010a','price.write','Manage the four price tables'),
 ('00000000-0000-7000-8000-00000000010b','package.manage','Manage packages'),
 ('00000000-0000-7000-8000-00000000010c','order.read','View orders'),
 ('00000000-0000-7000-8000-00000000010d','order.write','Create and amend orders on behalf of customers'),
 ('00000000-0000-7000-8000-00000000010e','order.cancel','Cancel an order'),
 ('00000000-0000-7000-8000-00000000010f','payment.verify','Verify or reject a payment'),
 ('00000000-0000-7000-8000-000000000110','payment.refund','Return an erroneous transfer (admin only, D-31)'),
 ('00000000-0000-7000-8000-000000000111','credit.adjust','Post a credit adjustment or extend an expiry'),
 ('00000000-0000-7000-8000-000000000112','delivery.read','View deliveries and manifests'),
 ('00000000-0000-7000-8000-000000000113','delivery.fulfil','Mark prepared, dispatched or delivered'),
 ('00000000-0000-7000-8000-000000000114','delivery.reassign','Reassign a delivery to another kitchen'),
 ('00000000-0000-7000-8000-000000000115','kitchen.read','View kitchens'),
 ('00000000-0000-7000-8000-000000000116','kitchen.write','Manage kitchens, service areas and capacity'),
 ('00000000-0000-7000-8000-000000000117','report.read','Run reports'),
 ('00000000-0000-7000-8000-000000000118','report.financial','Run financial reports'),
 ('00000000-0000-7000-8000-000000000119','settings.read','View system settings'),
 ('00000000-0000-7000-8000-00000000011a','settings.write','Change system settings'),
 ('00000000-0000-7000-8000-00000000011b','user.manage','Manage staff users and roles'),
 ('00000000-0000-7000-8000-00000000011c','audit.read','Read the audit log');

-- ── roles (PROMPT §3) ────────────────────────────────────────────────────────
-- requires_2fa is false for kitchen and courier: they work from shared or phone
-- devices on a service floor, and their accounts are read-mostly and scoped to
-- one kitchen (docs/03 Q-16). Say the word and it becomes true for all five.
INSERT INTO role (id, code, name, description, is_staff, is_system, requires_2fa) VALUES
 ('00000000-0000-7000-8000-000000000201','customer','Customer','Self-registered customer',FALSE,TRUE,FALSE),
 ('00000000-0000-7000-8000-000000000202','staff','Staff','Menu, schedule, customer support',TRUE,TRUE,TRUE),
 ('00000000-0000-7000-8000-000000000203','finance','Finance','Payment verification and reports',TRUE,TRUE,TRUE),
 ('00000000-0000-7000-8000-000000000204','kitchen','Kitchen','Production sheets, mark prepared',TRUE,TRUE,FALSE),
 ('00000000-0000-7000-8000-000000000205','courier','Courier','Delivery manifest, mark delivered',TRUE,TRUE,FALSE),
 ('00000000-0000-7000-8000-000000000206','admin','Administrator','Everything, plus settings and users',TRUE,TRUE,TRUE);

-- admin gets everything
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000206', id FROM permission;

-- staff
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000202', id FROM permission WHERE code IN
  ('customer.read','customer.write','customer.type.change','organisation.manage',
   'catalogue.read','catalogue.write','schedule.read','schedule.write',
   'price.read','package.manage','order.read','order.write','order.cancel',
   'delivery.read','delivery.reassign','kitchen.read','report.read','settings.read');

-- finance
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000203', id FROM permission WHERE code IN
  ('customer.read','order.read','payment.verify','credit.adjust',
   'report.read','report.financial','price.read','settings.read','delivery.read');

-- kitchen
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000204', id FROM permission WHERE code IN
  ('schedule.read','catalogue.read','delivery.read','delivery.fulfil','report.read','kitchen.read');

-- courier
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000205', id FROM permission WHERE code IN
  ('delivery.read','delivery.fulfil');

-- ── customer types (PROMPT §4.1) ─────────────────────────────────────────────
INSERT INTO customer_type (id, name, slug, is_corporate, is_system, sort_order, description) VALUES
 ('00000000-0000-7000-8000-000000000301','Customer Default','customer-default',FALSE,TRUE,1,
  'Every new registration lands here. Cannot be deleted.'),
 ('00000000-0000-7000-8000-000000000302','Siloam Customer','siloam-customer',TRUE,FALSE,2,''),
 ('00000000-0000-7000-8000-000000000303','Company A','company-a',TRUE,FALSE,3,''),
 ('00000000-0000-7000-8000-000000000304','Company B','company-b',TRUE,FALSE,4,'');

-- ── diet types (PROMPT §4.2) ─────────────────────────────────────────────────
INSERT INTO diet_type (id, name, slug, description, has_subtypes, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000401','Healthy','healthy',
  'Seimbang, porsi terkontrol, untuk menjaga pola makan sehat sehari-hari.',FALSE,1),
 ('00000000-0000-7000-8000-000000000402','Weight Gain','weight-gain',
  'Kalori dan protein lebih tinggi untuk menambah massa tubuh.',FALSE,2),
 ('00000000-0000-7000-8000-000000000403','Weight Loss','weight-loss',
  'Defisit kalori terukur dengan protein tetap tinggi agar kenyang lebih lama.',FALSE,3),
 ('00000000-0000-7000-8000-000000000404','High Protein','high-protein',
  'Protein tinggi untuk mendukung latihan dan pemulihan otot.',FALSE,4),
 ('00000000-0000-7000-8000-000000000405','Special Diet','special-diet',
  'Menu khusus sesuai kondisi kesehatan tertentu.',TRUE,5);

INSERT INTO diet_subtype (id, diet_type_id, name, slug, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000411','00000000-0000-7000-8000-000000000405','Diabetic','diabetic',1),
 ('00000000-0000-7000-8000-000000000412','00000000-0000-7000-8000-000000000405','Cholesterol','cholesterol',2);

-- ── allergens ────────────────────────────────────────────────────────────────
INSERT INTO allergen (id, code, name_id, name_en, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000501','peanut','Kacang tanah','Peanut',1),
 ('00000000-0000-7000-8000-000000000502','tree_nut','Kacang pohon','Tree nut',2),
 ('00000000-0000-7000-8000-000000000503','milk','Susu','Milk',3),
 ('00000000-0000-7000-8000-000000000504','egg','Telur','Egg',4),
 ('00000000-0000-7000-8000-000000000505','fish','Ikan','Fish',5),
 ('00000000-0000-7000-8000-000000000506','shellfish','Kerang-kerangan','Shellfish',6),
 ('00000000-0000-7000-8000-000000000507','soy','Kedelai','Soy',7),
 ('00000000-0000-7000-8000-000000000508','gluten','Gluten','Gluten',8),
 ('00000000-0000-7000-8000-000000000509','sesame','Wijen','Sesame',9);

-- ── delivery time slots (PROMPT §8.1) ────────────────────────────────────────
-- Exactly two active, as specified. Others exist inactive so an admin can
-- switch one on without creating it.
INSERT INTO delivery_time_slot (id, slot_time, alias, sort_order, is_active) VALUES
 ('00000000-0000-7000-8000-000000000601','11:30','Lunch',1,TRUE),
 ('00000000-0000-7000-8000-000000000602','18:30','Dinner',2,TRUE),
 ('00000000-0000-7000-8000-000000000603','07:00','Breakfast',0,FALSE),
 ('00000000-0000-7000-8000-000000000604','10:00','Mid-morning',3,FALSE),
 ('00000000-0000-7000-8000-000000000605','15:00','Afternoon',4,FALSE);

-- ── meal price tiers (PROMPT §5.4) ───────────────────────────────────────────
-- Contiguous from 1 with no gaps, which the admin form also validates.
INSERT INTO meal_price_tier (id, label, min_qty, max_qty, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000701','1–4 meals',1,4,1),
 ('00000000-0000-7000-8000-000000000702','5–9 meals',5,9,2),
 ('00000000-0000-7000-8000-000000000703','10–19 meals',10,19,3),
 ('00000000-0000-7000-8000-000000000704','20+ meals',20,NULL,4);

-- ── packages (PROMPT §5.5) ───────────────────────────────────────────────────
-- No package_diet_type rows = any diet type (docs/02 D-12).
INSERT INTO package (id, name, slug, description, meal_credits, validity_days, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000801','Paket 10','paket-10','10 kredit makan, berlaku 30 hari.',10,30,1),
 ('00000000-0000-7000-8000-000000000802','Paket 20','paket-20','20 kredit makan, berlaku 45 hari.',20,45,2),
 ('00000000-0000-7000-8000-000000000803','Paket 30','paket-30','30 kredit makan, berlaku 60 hari.',30,60,3);

-- ── bank account (PROMPT §10) ────────────────────────────────────────────────
-- PLACEHOLDER. docs/03 Q-19: the real details are still needed from Steven and
-- payment instructions must not go out with these.
INSERT INTO bank_account (id, bank_name, account_number, account_holder, is_active, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000901','BCA','0000000000','PT EVERMORE PLACEHOLDER',TRUE,1);

-- ── sys_parameters (CLAUDE.md §7) ────────────────────────────────────────────
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_system, sort_order) VALUES
 -- tax (D-30)
 ('00000000-0000-7000-8000-000000000a01','tax.rate_bps','1100','bps','tax','PPN rate (basis points)',
  'PLACEHOLDER 11%. docs/03 Q-1a: the real rate, and whether Evermore is PKP-registered, are still needed. Prices are tax-INCLUSIVE; this drives the base/tax split snapshotted on each order.',FALSE,1),
 ('00000000-0000-7000-8000-000000000a02','tax.inclusive','true','bool','tax','Prices include tax',
  'Steven, 2026-08-12: all prices are inclusive. Changing this is a pricing-model change, not a setting.',TRUE,2),
 ('00000000-0000-7000-8000-000000000a03','company.npwp','','string','company','NPWP',
  'Needed on corporate invoices (docs/03 Q-20).',FALSE,3),
 ('00000000-0000-7000-8000-000000000a04','company.legal_name','PT Evermore Placeholder','string','company','Legal entity name','',FALSE,4),
 ('00000000-0000-7000-8000-000000000a05','company.address','','string','company','Registered address','',FALSE,5),
 ('00000000-0000-7000-8000-000000000a06','company.phone','','string','company','Public phone number','',FALSE,6),
 ('00000000-0000-7000-8000-000000000a07','company.email','halo@evermore.co.id','string','company','Public email','',FALSE,7),
 ('00000000-0000-7000-8000-000000000a08','company.whatsapp','','string','company','Customer-service WhatsApp',
  'Used for the CS deep link on every order page (PROMPT §13.10).',FALSE,8),

 -- ordering (PROMPT §6)
 ('00000000-0000-7000-8000-000000000a10','order.cutoff_time','18:00','time','ordering','Order cut-off time (WIB)',
  'Orders for date D close at this time on D minus the lead days.',FALSE,10),
 ('00000000-0000-7000-8000-000000000a11','order.cutoff_lead_days','1','int','ordering','Cut-off lead days','',FALSE,11),
 ('00000000-0000-7000-8000-000000000a12','order.max_qty_per_line','999','int','ordering','Maximum meals per line','',FALSE,12),
 ('00000000-0000-7000-8000-000000000a13','order.payment_window','2h','duration','ordering','Payment window',
  'Capped at the cut-off (docs/02 D-13): a flat window applied at 17:30 would otherwise hold capacity past the 18:00 cut-off it was placed against.',FALSE,13),
 ('00000000-0000-7000-8000-000000000a14','order.unique_code_enabled','true','bool','ordering','Add a unique 3-digit transfer suffix',
  'The Indonesian kode unik. Not taxable — it is a matching device, not consideration.',FALSE,14),

 -- delivery fee (D-19) — PLACEHOLDERS, docs/03 Q-3
 ('00000000-0000-7000-8000-000000000a20','delivery.fee_bands','[{"max_km":5,"fee":0},{"max_km":10,"fee":15000},{"max_km":null,"fee":25000}]','json','delivery','Delivery fee bands (km from the assigned kitchen)',
  'PLACEHOLDER figures. docs/03 Q-3: Steven has not given the real bands.',FALSE,20),
 ('00000000-0000-7000-8000-000000000a21','delivery.free_above_idr','300000','money','delivery','Free delivery above order value',
  'PLACEHOLDER. docs/03 Q-3.',FALSE,21),
 ('00000000-0000-7000-8000-000000000a22','delivery.package_included','true','bool','delivery','Package deliveries include delivery',
  'Assumed: otherwise a 20-credit package has an unpredictable total (docs/03 Q-3).',FALSE,22),

 -- geography (docs/03 Q-11)
 ('00000000-0000-7000-8000-000000000a30','geo.envelope','{"min_lat":-6.60,"max_lat":-5.90,"min_lng":106.50,"max_lng":107.10}','json','geography','Plausible coordinate envelope',
  'Jabodetabek. Outside this is an INPUT error (a mis-signed or missing pin); inside but uncovered is "not serviceable yet", which is a different message.',FALSE,30),
 ('00000000-0000-7000-8000-000000000a31','geo.places_fallback_blocked','true','bool','geography','Block address entry without a map pin',
  'docs/02 D-17: a hand-typed coordinate is the likeliest cause of a delivery sent across the city. Staff can set coordinates from the back office, audit-logged.',FALSE,31),

 -- packages and credits
 ('00000000-0000-7000-8000-000000000a40','credit.low_threshold','2','int','credits','Warn when credits remaining drop to','',FALSE,40),
 ('00000000-0000-7000-8000-000000000a41','credit.expiry_warning_days','3','int','credits','Warn this many days before expiry','',FALSE,41),
 ('00000000-0000-7000-8000-000000000a42','credit.package_capacity_reserve_pct','0','int','credits','Kitchen capacity reserved for package holders',
  'docs/03 Q-9: 0 = off. A prepaid customer refused a slot is a worse experience than a new order refused; turn this up if that complaint arrives.',FALSE,42),

 -- schedule
 ('00000000-0000-7000-8000-000000000a50','schedule.publish_horizon_days','7','int','schedule','Target menu publication horizon',
  'Package customers cannot book what is not published (docs/03 Q-17). Surfaced as a dashboard warning, not an error.',FALSE,50),

 -- notifications
 ('00000000-0000-7000-8000-000000000a60','notify.email_enabled','true','bool','notifications','Send transactional email','',FALSE,60),
 ('00000000-0000-7000-8000-000000000a61','notify.whatsapp_enabled','false','bool','notifications','Send WhatsApp notifications',
  'Off until a provider is chosen (docs/02 D-22, docs/03 Q-23).',FALSE,61),
 ('00000000-0000-7000-8000-000000000a62','notify.reminder_hour','19:00','time','notifications','Evening-before reminder time','',FALSE,62),

 -- security
 ('00000000-0000-7000-8000-000000000a70','security.max_login_attempts','5','int','security','Failed logins before lockout','',FALSE,70),
 ('00000000-0000-7000-8000-000000000a71','security.lockout_duration','15m','duration','security','Lockout duration','',FALSE,71),
 ('00000000-0000-7000-8000-000000000a72','security.require_email_verification','true','bool','security','Require a verified email before the first order','',FALSE,72);

-- ── kitchens ─────────────────────────────────────────────────────────────────
-- PLACEHOLDERS. docs/03 Q-4: Steven gave the routing RULE ("nearest kitchen")
-- but not the kitchen data. These two are real Jakarta coordinates chosen so
-- routing can be demonstrated and tested end to end; they must be replaced
-- before launch. Equal priority, so ranking collapses to nearest-first (D-34).
INSERT INTO kitchen (id, code, name, address_line, district, city, latitude, longitude,
                     service_radius_km, pic_name, priority, is_active, notes) VALUES
 ('00000000-0000-7000-8000-000000000b01','JKT-S','Evermore Kitchen Selatan',
  'PLACEHOLDER — Kemang, Jakarta Selatan','Mampang Prapatan','Jakarta Selatan',
  -6.260700,106.814500,12,'PLACEHOLDER',100,TRUE,
  'PLACEHOLDER kitchen for development. Replace with real data before launch (docs/03 Q-4).'),
 ('00000000-0000-7000-8000-000000000b02','JKT-P','Evermore Kitchen Pusat',
  'PLACEHOLDER — Menteng, Jakarta Pusat','Menteng','Jakarta Pusat',
  -6.175400,106.827200,12,'PLACEHOLDER',100,TRUE,
  'PLACEHOLDER kitchen for development. Replace with real data before launch (docs/03 Q-4).');

INSERT INTO kitchen_slot (kitchen_id, slot_id)
  SELECT k.id, s.id FROM kitchen k CROSS JOIN delivery_time_slot s WHERE s.is_active;

-- Monday to Saturday.
INSERT INTO kitchen_operating_day (kitchen_id, weekday)
  SELECT k.id, d FROM kitchen k CROSS JOIN generate_series(1,6) AS d;
