-- Migration: Drop highlighted_amenities table
DROP INDEX IF EXISTS idx_highlighted_amenities_room_id;
DROP TABLE IF EXISTS highlighted_amenities;
