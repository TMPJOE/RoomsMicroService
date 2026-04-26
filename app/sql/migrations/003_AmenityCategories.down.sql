-- Migration: Drop amenity_categories table
DROP INDEX IF EXISTS idx_amenity_categories_room_id;
DROP TABLE IF EXISTS amenity_categories;
