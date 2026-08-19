-- The certification badges on the home page.
--
-- Steven, 2026-08-19: three large badges — HALAL, HACCP, ISO 22000.
--
-- BEHIND A SWITCH, and this one is not housekeeping. These are formal
-- certification claims:
--
--   * Halal labelling in Indonesia is regulated (UU 33/2014, BPJPH/MUI).
--     Displaying it without a current certificate is an offence, not a
--     marketing exaggeration.
--   * HACCP and ISO 22000 are audited certifications held by a named legal
--     entity with a certificate number and an expiry.
--
-- So it is one parameter to switch the row off the moment a certificate
-- lapses, and the caption under each badge is editable content — put the real
-- certificate number there. The badges are TYPOGRAPHIC, not the certifying
-- bodies' logos: those are trademarks, issued with their own usage rules, and
-- must be supplied by the issuer rather than drawn by us.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c33','public.certifications_enabled','true','bool','public','Show certification badges',
  'The HALAL / HACCP / ISO 22000 badges at the foot of the home page. TURN THIS OFF unless Evermore currently holds each certificate — halal labelling in Indonesia is regulated under UU 33/2014 and displaying it uncertified is an offence. Put the certificate numbers in the captions on the Content screen.',FALSE,FALSE,4);

-- Captions, editable and translated. Empty by default: an invented certificate
-- number would be worse than none.
INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html) VALUES
 ('cert.heading', 'id', 'Standar keamanan pangan kami', FALSE, '', FALSE),
 ('cert.halal_note',  'id', '', FALSE, '', FALSE),
 ('cert.haccp_note',  'id', '', FALSE, '', FALSE),
 ('cert.iso_note',    'id', '', FALSE, '', FALSE)
ON CONFLICT (key, locale) DO NOTHING;

INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html)
SELECT v.key, v.locale, v.value, TRUE,
       encode(sha256(convert_to(src.value, 'UTF8')), 'hex'), FALSE
  FROM (VALUES
    ('cert.heading', 'en', 'Our food-safety standards'),
    ('cert.heading', 'zh', '我们的食品安全标准')
  ) AS v(key, locale, value)
  JOIN public_content src ON src.key = v.key AND src.locale = 'id'
ON CONFLICT (key, locale) DO NOTHING;
