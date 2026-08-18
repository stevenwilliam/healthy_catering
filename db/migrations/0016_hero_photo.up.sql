-- Point the home hero at the supplied photograph.
--
-- 0015 shipped with a placeholder illustration, which was deleted once Steven
-- supplied the real picture (2026-08-18). Forward-only, so the value is moved
-- here rather than by editing 0015: any environment that already ran 0015 has
-- the old path in its row and would otherwise keep serving a 404.
UPDATE sys_parameters
   SET value = '/images/hero-home.jpg'
 WHERE key = 'public.hero_image'
   AND value = '/images/hero-meditation.svg';
