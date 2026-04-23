// Package models defines domain data structures used across the application.
// All entities, DTOs, and shared types should be defined here
// to ensure consistency between repository, service, and handler layers.
package models

import (
	"encoding/json"
	"time"
)

// Room represents a room type in the rooms table
type Room struct {
	ID                   string          `json:"id"`
	HotelID              string          `json:"hotel_id"`
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	Price                float64         `json:"price"`
	Capacity             int             `json:"capacity"`
	Description          string          `json:"description"`
	SpaceInfo            string          `json:"space_info"`
	BedDistribution      string          `json:"bed_distribution"`
	Quantity             int             `json:"quantity"`
	HighlightedAmenities json.RawMessage `json:"highlighted_amenities"`
	AmenityCategories    string          `json:"amenity_categories"`
	AmenityCount         int             `json:"amenity_count"`
	RecommendationCoef   float64         `json:"recommendation_coef"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// HighlightedAmenity represents a single highlighted amenity with icon and text
type HighlightedAmenity struct {
	Icon string `json:"icon"`
	Text string `json:"text"`
}

// CreateRoomRequest represents the request to create a room
type CreateRoomRequest struct {
	Name                 string                    `json:"name" validate:"required"`
	Type                 string                    `json:"type" validate:"required,oneof=Single Double Double/Double Suite"`
	Price                float64                   `json:"price" validate:"required,gt=0"`
	Capacity             int                       `json:"capacity" validate:"required,gt=0"`
	Description          string                    `json:"description"`
	SpaceInfo            string                    `json:"space_info"`
	BedDistribution      string                    `json:"bed_distribution"`
	Quantity             int                       `json:"quantity" validate:"required,gt=0"`
	HighlightedAmenities []HighlightedAmenityInput `json:"highlighted_amenities"`
	AmenityCategories    string                    `json:"amenity_categories"`
}

// HighlightedAmenityInput represents input for a highlighted amenity
type HighlightedAmenityInput struct {
	Icon string `json:"icon"`
	Text string `json:"text"`
}

// UpdateRoomRequest represents the request to update a room
type UpdateRoomRequest struct {
	Name                 string                    `json:"name"`
	Type                 string                    `json:"type" validate:"omitempty,oneof=Single Double Double/Double Suite"`
	Price                float64                   `json:"price" validate:"omitempty,gt=0"`
	Capacity             int                       `json:"capacity" validate:"omitempty,gt=0"`
	Description          string                    `json:"description"`
	SpaceInfo            string                    `json:"space_info"`
	BedDistribution      string                    `json:"bed_distribution"`
	Quantity             int                       `json:"quantity" validate:"omitempty,gt=0"`
	HighlightedAmenities []HighlightedAmenityInput `json:"highlighted_amenities"`
	AmenityCategories    string                    `json:"amenity_categories"`
}

// UpdateRoomQuantityRequest represents the request to update room quantity
type UpdateRoomQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

// RoomResponse represents the response for room operations
type RoomResponse struct {
	ID                   string          `json:"id"`
	HotelID              string          `json:"hotel_id"`
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	Price                float64         `json:"price"`
	Capacity             int             `json:"capacity"`
	Description          string          `json:"description"`
	SpaceInfo            string          `json:"space_info"`
	BedDistribution      string          `json:"bed_distribution"`
	Quantity             int             `json:"quantity"`
	HighlightedAmenities json.RawMessage `json:"highlighted_amenities"`
	AmenityCategories    string          `json:"amenity_categories"`
	AmenityCount         int             `json:"amenity_count"`
	RecommendationCoef   float64         `json:"recommendation_coef"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// RoomListResponse represents a list of rooms with metadata
type RoomListResponse struct {
	Rooms      []RoomResponse `json:"rooms"`
	TotalCount int            `json:"total_count"`
}

// FilterRoomsRequest represents query parameters for filtering rooms
type FilterRoomsRequest struct {
	HotelID     string `validate:"omitempty,uuid"`
	Type        string `validate:"omitempty,oneof=Single Double Double/Double Suite"`
	Types       []string
	MinCapacity int     `validate:"omitempty,gte=0"`
	MaxCapacity int     `validate:"omitempty,gte=0"`
	MinPrice    float64 `validate:"omitempty,gte=0"`
	MaxPrice    float64 `validate:"omitempty,gte=0"`
	MinCoef     float64 `validate:"omitempty,gte=0"`
	MaxCoef     float64 `validate:"omitempty,gte=0"`
	Limit       int     `validate:"omitempty,gte=1,lte=100"`
	Offset      int     `validate:"omitempty,gte=0"`
}

// AvailabilityRequest represents request for checking room availability
type AvailabilityRequest struct {
	HotelID  string    `json:"hotel_id" validate:"required,uuid"`
	CheckIn  time.Time `json:"check_in" validate:"required"`
	CheckOut time.Time `json:"check_out" validate:"required"`
	Quantity int       `json:"quantity" validate:"required,gt=0"`
}
