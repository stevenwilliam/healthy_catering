-- The corner ribbon.
--
-- Steven, 2026-08-19: a diagonal gold ribbon in the top-right of every page,
-- reading "free delivery".
--
-- A SWITCH, not hard-coded markup, because the claim is only true while the
-- pricing says so. Today delivery.fee_bands is [{"max_km":null,"fee":0}] and
-- delivery.free_above_idr is 0 — free at any distance, on any order. The day a
-- fee band is introduced, this ribbon becomes an advertised promise the
-- checkout does not keep, and taking it down must not need a deploy.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c32','public.ribbon_enabled','true','bool','public','Show the corner ribbon',
  'The diagonal ribbon in the top-right of every public page. Its wording is edited on the Content screen (ribbon.text). TURN THIS OFF if delivery stops being free — delivery.fee_bands and delivery.free_above_idr are what make the claim true.',FALSE,FALSE,3);

-- Wording, editable and translated like the rest of the public copy.
INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html) VALUES
 ('ribbon.text', 'id', 'Gratis ongkir', FALSE, '', FALSE)
ON CONFLICT (key, locale) DO NOTHING;

INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html)
SELECT v.key, v.locale, v.value, TRUE,
       encode(sha256(convert_to(src.value, 'UTF8')), 'hex'), FALSE
  FROM (VALUES
    ('ribbon.text', 'en', 'Free delivery'),
    ('ribbon.text', 'zh', '免费配送')
  ) AS v(key, locale, value)
  JOIN public_content src ON src.key = v.key AND src.locale = 'id'
ON CONFLICT (key, locale) DO NOTHING;
