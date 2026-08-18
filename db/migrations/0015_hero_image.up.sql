-- The home page hero picture.
--
-- A sys_parameters row rather than a hard-coded path (CLAUDE.md §7): swapping
-- the picture is exactly the kind of change that must not need a deploy. The
-- default points at the illustration committed under web/public/images; set
-- this to an uploaded photograph when one exists.
INSERT INTO sys_parameters (id, key, value, value_type, param_group, label, description, is_secret, is_system, sort_order) VALUES
 ('00000000-0000-7000-8000-000000000c30','public.hero_image','/images/hero-meditation.svg','string','public','Home hero picture',
  'Path or URL of the large picture beside the home-page headline. Ships as a brand illustration; replace with a photograph (recommended: square or 4:5, at least 900px on the short edge). Clearing it hides the picture and the headline goes full width.',FALSE,FALSE,1);
