-- Migration: Drop rooms table and its indexes
DROP INDEX IF EXISTS idx_rooms_updated_at;
DROP INDEX IF EXISTS idx_rooms_created_at;
DROP INDEX IF EXISTS idx_rooms_coef;
DROP INDEX IF EXISTS idx_rooms_price;
DROP INDEX IF EXISTS idx_rooms_capacity;
DROP INDEX IF EXISTS idx_rooms_type;
DROP INDEX IF EXISTS idx_rooms_hotel_id;

DROP TABLE IF EXISTS rooms;
