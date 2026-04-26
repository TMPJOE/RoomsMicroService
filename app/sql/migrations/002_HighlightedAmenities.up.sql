-- Migration: Create highlighted_amenities table
-- These are the featured/important amenities shown with icons on the room card.
-- Each row belongs to one room and carries an icon name (must match AmenityIcon constants)
-- and a short display text shown beneath the icon in the UI.
CREATE TABLE IF NOT EXISTS highlighted_amenities (
    id          TEXT PRIMARY KEY,
    room_id     TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    icon        TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_highlighted_amenities_room_id ON highlighted_amenities(room_id);
