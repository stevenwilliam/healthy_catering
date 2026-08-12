DROP INDEX IF EXISTS staff_profile_kitchen_idx;
ALTER TABLE staff_profile DROP CONSTRAINT IF EXISTS staff_profile_kitchen_fk;
DROP TABLE IF EXISTS kitchen_travel_cache;
DROP TABLE IF EXISTS out_of_range_attempt;
DROP TABLE IF EXISTS kitchen_capacity;
DROP TABLE IF EXISTS kitchen_operating_day;
DROP TABLE IF EXISTS kitchen_slot;
DROP TABLE IF EXISTS kitchen;
