-- Packages are described in DAYS, without the validity window.
--
-- Steven, 2026-08-18: "10 credits become 10 hari and remove valid days
-- information". The badge on the card was already changed in the template;
-- this is the other half — the descriptions are database rows and still read
-- "10 kredit makan, berlaku 30 hari."
--
-- Only the WORDING moves. validity_days is untouched and still governs when
-- the credits actually expire, and meal_credits is still what a booking
-- decrements. Nothing about how the product behaves changes here.
UPDATE package SET description = meal_credits || ' hari makan sehat.'
 WHERE description LIKE '%kredit makan%';
