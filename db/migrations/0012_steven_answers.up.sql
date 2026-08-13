-- 0012 — Steven's answers of 2026-08-13 (docs/00 D13).
--
-- Every one of these is a data change rather than a schema change, which is the
-- point: they were parameters and seed rows precisely so answers like these
-- cost a migration of INSERTs instead of a rewrite.

-- ── B2: kitchens operate every day ───────────────────────────────────────────
-- Seeded Monday–Saturday; Steven: "our kitchen is servicing every day".
INSERT INTO kitchen_operating_day (kitchen_id, weekday)
  SELECT k.id, 7 FROM kitchen k
 WHERE NOT EXISTS (
   SELECT 1 FROM kitchen_operating_day o WHERE o.kitchen_id = k.id AND o.weekday = 7);

-- ── B6: delivery is free, everywhere, at every order value ───────────────────
-- Steven: "all is free delivery all price all range" — but keep it configurable.
-- One open-ended band at zero does that: the fee engine still runs on every
-- order and every report, so turning charging on later is one settings edit and
-- not a code path that has never executed.
UPDATE sys_parameters
   SET value = '[{"max_km":null,"fee":0}]',
       description = 'Steven, 2026-08-13: delivery is free at every distance and every order value for now. The engine still evaluates this on every order, so charging later is a settings change — replace with e.g. [{"max_km":5,"fee":0},{"max_km":10,"fee":15000},{"max_km":null,"fee":25000}].'
 WHERE key = 'delivery.fee_bands';

UPDATE sys_parameters
   SET value = '0',
       description = 'Zero means no threshold is applied — delivery is free regardless of order value (Steven, 2026-08-13). Set a rupiah amount to make free delivery conditional again.'
 WHERE key = 'delivery.free_above_idr';

-- ── B3: PPN confirmed at 11% ─────────────────────────────────────────────────
UPDATE sys_parameters
   SET description = 'Confirmed 11% by Steven, 2026-08-13, changeable here without a deploy. Prices are tax-INCLUSIVE; this drives the base/tax split snapshotted on each order. PKP status and NPWP are still outstanding for the first real invoice (docs/03 Q-1a).'
 WHERE key = 'tax.rate_bps';

-- ── B7: mail settings become parameters, editable in the back office ─────────
-- Steven: "make this can be change via backend, so i can change later".
--
-- The password is secret-flagged, so it is masked in the UI and in logs. It
-- lives here rather than only in the env because the whole point of the answer
-- is that Steven can change the relay without a deploy; the env var still wins
-- when set, so a production secret can stay out of the database entirely.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c01','mail.host','127.0.0.1','string','mail','SMTP host',
  'Borrowed from ruuma: the shared mailpit satellite on this dev server. Replace with the real relay for production.',FALSE,FALSE,1),
 ('00000000-0000-7000-8000-000000000c02','mail.port','1025','int','mail','SMTP port',
  'mailpit listens on 1025; its web UI is on 8025. A real relay is 587.',FALSE,FALSE,2),
 ('00000000-0000-7000-8000-000000000c03','mail.username','','string','mail','SMTP username','',FALSE,FALSE,3),
 ('00000000-0000-7000-8000-000000000c04','mail.password','','string','mail','SMTP password',
  'Masked in the UI and in logs. The SMTP_PASSWORD environment variable overrides this when set, so a production secret need never be stored in the database.',TRUE,FALSE,4),
 ('00000000-0000-7000-8000-000000000c05','mail.from_email','no-reply@evermore.co.id','string','mail','From address',
  'The domain needs SPF, DKIM and DMARC published before real sending, or the mail lands in spam (docs/03 Q-21).',FALSE,FALSE,5),
 ('00000000-0000-7000-8000-000000000c06','mail.from_name','Evermore','string','mail','From name','',FALSE,FALSE,6),
 ('00000000-0000-7000-8000-000000000c07','mail.tls','false','bool','mail','Use TLS',
  'False for the local mailpit satellite. MUST be true for any real relay.',FALSE,FALSE,7),

-- ── B8: WhatsApp via WAHA ────────────────────────────────────────────────────
-- Steven: "use WAHA", which matches 99 §9 — one self-hosted shared container,
-- with the Meta Cloud API as the documented swap-in behind the same port.
 ('00000000-0000-7000-8000-000000000c10','whatsapp.provider','WAHA','string','whatsapp','WhatsApp provider',
  'WAHA (self-hosted) or META_CLOUD. Both sit behind the same NotificationChannel port, so switching is a config change. Note the tradeoff: WAHA is free and unofficial, and a banned number takes the channel down mid-service.',FALSE,FALSE,10),
 ('00000000-0000-7000-8000-000000000c11','whatsapp.waha_url','http://127.0.0.1:3000','string','whatsapp','WAHA base URL',
  'The shared WAHA container already running on this dev server.',FALSE,FALSE,11),
 ('00000000-0000-7000-8000-000000000c12','whatsapp.waha_session','default','string','whatsapp','WAHA session name','',FALSE,FALSE,12),
 ('00000000-0000-7000-8000-000000000c13','whatsapp.waha_api_key','','string','whatsapp','WAHA API key',
  'Masked. The WAHA_API_KEY environment variable overrides this when set.',TRUE,FALSE,13);

-- WhatsApp stays OFF until a sender number is confirmed; the channel exists and
-- is exercised by tests, but nothing goes out to a customer yet.
UPDATE sys_parameters
   SET description = 'Steven chose WAHA, 2026-08-13. Still false until a sender number is confirmed — turn on here, no deploy needed.'
 WHERE key = 'notify.whatsapp_enabled';

-- ── B13: no minimum order ────────────────────────────────────────────────────
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c20','order.min_qty','1','int','ordering','Minimum meals per order',
  'Steven, 2026-08-13: no minimum. 1 is the floor a cart needs anyway.',FALSE,FALSE,15),
 ('00000000-0000-7000-8000-000000000c21','order.min_value_idr','0','money','ordering','Minimum order value',
  'Zero = no minimum (Steven, 2026-08-13).',FALSE,FALSE,16);

-- ── B4: the dummy bank account says so ───────────────────────────────────────
UPDATE bank_account
   SET account_holder = 'PT EVERMORE (DUMMY — REPLACE BEFORE LAUNCH)',
       account_number = '1234567890'
 WHERE bank_name = 'BCA';
