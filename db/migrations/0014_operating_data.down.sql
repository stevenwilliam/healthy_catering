-- Down for 0014. Restores the placeholder operating data.
--
-- This is a data migration, so "down" means putting the seed state back, not
-- dropping anything. It is exercised in CI (migrate up → down → up), which is
-- the only place a broken down file is ever caught — production is forward-only.

-- ── 4. WhatsApp ──────────────────────────────────────────────────────────────
UPDATE sys_parameters SET value = 'false', updated_at = now()
 WHERE key = 'notify.whatsapp_enabled';
UPDATE sys_parameters SET value = '', updated_at = now()
 WHERE key = 'whatsapp.waha_api_key';

-- ── 3. Company ───────────────────────────────────────────────────────────────
UPDATE sys_parameters SET value = 'PT Evermore Placeholder', updated_at = now()
 WHERE key = 'company.legal_name';
UPDATE sys_parameters SET value = '', updated_at = now()
 WHERE key IN ('company.address', 'company.phone', 'company.whatsapp', 'company.npwp');

-- ── 2. Bank account ──────────────────────────────────────────────────────────
DELETE FROM bank_account WHERE account_number = '16830226665';
UPDATE bank_account SET is_active = TRUE, updated_at = now()
 WHERE account_number = '1234567890';

-- ── 1. Kitchens ──────────────────────────────────────────────────────────────
-- The child rows go first: kitchen_slot and kitchen_operating_day cascade on
-- delete, but being explicit keeps this readable and order-independent.
DELETE FROM kitchen_operating_day
 WHERE kitchen_id IN (SELECT id FROM kitchen WHERE code IN ('LPV','MRCCC','CDG','TBS','KJ'));
DELETE FROM kitchen_slot
 WHERE kitchen_id IN (SELECT id FROM kitchen WHERE code IN ('LPV','MRCCC','CDG','TBS','KJ'));
DELETE FROM kitchen WHERE code IN ('LPV','MRCCC','CDG','TBS','KJ');

UPDATE kitchen
   SET is_active = TRUE,
       notes = replace(notes, ' Superseded by the real kitchens in migration 0014.', ''),
       updated_at = now()
 WHERE code IN ('JKT-S','JKT-P');
