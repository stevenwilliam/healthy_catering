DELETE FROM nav_item WHERE key = 'order';
ALTER TABLE nav_item DROP COLUMN IF EXISTS icon;
ALTER TABLE nav_item DROP COLUMN IF EXISTS is_localised;
