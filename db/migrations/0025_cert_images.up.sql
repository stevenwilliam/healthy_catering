-- The certification badges become IMAGES (Steven, 2026-08-19).
--
-- One parameter per badge, so replacing a seal with the certifying body's own
-- logo file is a settings change, not a deploy. That matters here more than
-- usual: the MUI/BPJPH halal mark, HACCP marks and the ISO 22000 mark are
-- trademarks issued to the certificate holder, with their own rules on size,
-- colour and clear space. They arrive as files from the issuer and are dropped
-- in; they are never redrawn by us.
--
-- The defaults are Evermore's own seal artwork (scripts/mkseals.py), which
-- carries the standard's NAME and nothing more. Clearing a value hides that
-- badge rather than rendering a broken image.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c34','public.cert_halal_image','/images/seal-halal.svg','string','public','HALAL badge image',
  'Path or URL of the HALAL badge. Replace with the certifying body''s own logo file when you have it. Empty hides this badge.',FALSE,FALSE,5),
 ('00000000-0000-7000-8000-000000000c35','public.cert_haccp_image','/images/seal-haccp.svg','string','public','HACCP badge image',
  'Path or URL of the HACCP badge. Empty hides this badge.',FALSE,FALSE,6),
 ('00000000-0000-7000-8000-000000000c36','public.cert_iso_image','/images/seal-iso-22000.svg','string','public','ISO 22000 badge image',
  'Path or URL of the ISO 22000 badge. Empty hides this badge.',FALSE,FALSE,7);
