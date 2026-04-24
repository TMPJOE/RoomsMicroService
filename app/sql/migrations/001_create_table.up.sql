CREATE TABLE IF NOT EXISTS rooms (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
hotel_id UUID NOT NULL,
name VARCHAR(255) NOT NULL,
type VARCHAR(50) NOT NULL,
price DECIMAL(10, 2) NOT NULL,
capacity INTEGER NOT NULL,
description TEXT,
space_info VARCHAR(255),
bed_distribution VARCHAR(255),
quantity INTEGER NOT NULL DEFAULT 1,
highlighted_amenities JSONB,
amenity_categories TEXT,
amenity_count INTEGER DEFAULT 0,
recommendation_coef DECIMAL(10, 4) DEFAULT 0,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for filtering and performance
CREATE INDEX idx_rooms_hotel_id ON rooms(hotel_id);
CREATE INDEX idx_rooms_type ON rooms(type);
CREATE INDEX idx_rooms_capacity ON rooms(capacity);
CREATE INDEX idx_rooms_price ON rooms(price);
CREATE INDEX idx_rooms_coef ON rooms(recommendation_coef);
CREATE INDEX idx_rooms_created_at ON rooms(created_at);
CREATE INDEX idx_rooms_updated_at ON rooms(updated_at);
