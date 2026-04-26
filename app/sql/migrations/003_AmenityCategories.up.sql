-- Migration: Create amenity_categories table
-- Categories group the less-prominent amenities (e.g. "Bathroom", "Kitchen").
-- Each category has a tier that maps to a multiplier used in the recommendation
-- coefficient formula: coef += TierMultiplier[tier] * amenity_count
-- Valid tier values: 'basic' | 'essential' | 'comfort' | 'luxury'
CREATE TABLE IF NOT EXISTS amenity_categories (
    id            TEXT PRIMARY KEY,
    room_id       TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT,
    tier          TEXT NOT NULL DEFAULT 'essential',
    amenity_count INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_amenity_categories_room_id ON amenity_categories(room_id);
