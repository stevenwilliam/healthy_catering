-- The illustration no longer exists in the tree, so this restores the value
-- only for symmetry; 0015's down removes the row entirely.
UPDATE sys_parameters
   SET value = '/images/hero-meditation.svg'
 WHERE key = 'public.hero_image'
   AND value = '/images/hero-home.jpg';
