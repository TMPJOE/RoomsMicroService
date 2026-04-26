package models

import (
	"time"
)

// HighlightedAmenity is a featured amenity displayed with an icon on the room card.
// json:"text" matches the field name the frontend reads (highlight.text in SvgIcon.vue).
type HighlightedAmenity struct {
	ID        string      `json:"id"`
	RoomID    string      `json:"room_id"`
	Icon      AmenityIcon `json:"icon"`
	Text      string      `json:"text"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// AmenityCategory groups less-prominent amenities under a named heading.
// AmenityCount is the number of individual amenities listed in Description,
// and is used (with Tier) to contribute to the recommendation coefficient.
type AmenityCategory struct {
	ID           string       `json:"id"`
	RoomID       string       `json:"room_id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Tier         CategoryTier `json:"tier"`
	AmenityCount int          `json:"amenity_count"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Room is the canonical domain model, including child amenity slices.
type Room struct {
	ID                   string               `json:"id"`
	HotelID              string               `json:"hotel_id"`
	Name                 string               `json:"name"`
	Type                 string               `json:"type"`
	Price                float64              `json:"price"`
	Capacity             int                  `json:"capacity"`
	Description          string               `json:"description"`
	SpaceInfo            string               `json:"space_info"`
	BedDistribution      string               `json:"bed_distribution"`
	Quantity             int                  `json:"quantity"`
	AmenityCount         int                  `json:"amenity_count"`
	RecommendationCoef   float64              `json:"recommendation_coef"`
	HighlightedAmenities []HighlightedAmenity `json:"highlighted_amenities"`
	AmenityCategories    []AmenityCategory    `json:"amenity_categories"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

// RoomResponse mirrors Room for API responses.
type RoomResponse struct {
	ID                   string               `json:"id"`
	HotelID              string               `json:"hotel_id"`
	Name                 string               `json:"name"`
	Type                 string               `json:"type"`
	Price                float64              `json:"price"`
	Capacity             int                  `json:"capacity"`
	Description          string               `json:"description"`
	SpaceInfo            string               `json:"space_info"`
	BedDistribution      string               `json:"bed_distribution"`
	Quantity             int                  `json:"quantity"`
	AmenityCount         int                  `json:"amenity_count"`
	RecommendationCoef   float64              `json:"recommendation_coef"`
	HighlightedAmenities []HighlightedAmenity `json:"highlighted_amenities"`
	AmenityCategories    []AmenityCategory    `json:"amenity_categories"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

// RoomListResponse wraps a paginated list of rooms.
type RoomListResponse struct {
	Rooms      []RoomResponse `json:"rooms"`
	TotalCount int            `json:"total_count"`
}

// HighlightedAmenityInput is used in create/update requests.
// No ID or RoomID — those are assigned server-side.
type HighlightedAmenityInput struct {
	Icon AmenityIcon `json:"icon" validate:"required"`
	Text string      `json:"text" validate:"required"`
}

// AmenityCategoryInput is used in create/update requests.
type AmenityCategoryInput struct {
	Name         string       `json:"name"          validate:"required"`
	Description  string       `json:"description"`
	Tier         CategoryTier `json:"tier"          validate:"required,oneof=basic essential comfort luxury"`
	AmenityCount int          `json:"amenity_count" validate:"required,gt=0"`
}

// CreateRoomRequest carries all fields needed to create one or more rooms.
type CreateRoomRequest struct {
	HotelID              string                   `json:"hotel_id"              validate:"required"`
	Name                 string                   `json:"name"                  validate:"required"`
	Type                 string                   `json:"type"                  validate:"required,oneof=Single Double Double/Double Suite"`
	Price                float64                  `json:"price"                 validate:"required,gt=0"`
	Capacity             int                      `json:"capacity"              validate:"required,gt=0"`
	Description          string                   `json:"description"           validate:"required,min=10,max=300"`
	SpaceInfo            string                   `json:"space_info"            validate:"required,min=10,max=300"`
	BedDistribution      string                   `json:"bed_distribution"      validate:"required,min=5,max=100"`
	Quantity             int                      `json:"quantity"              validate:"required,gt=0"`
	HighlightedAmenities []HighlightedAmenityInput `json:"highlighted_amenities" validate:"omitempty,dive"`
	AmenityCategories    []AmenityCategoryInput    `json:"amenity_categories"    validate:"omitempty,dive"`
}

// UpdateRoomRequest carries optional fields for partial room updates.
type UpdateRoomRequest struct {
	Name                 string                   `json:"name"                  validate:"omitempty"`
	Type                 string                   `json:"type"                  validate:"omitempty,oneof=Single Double Double/Double Suite"`
	Price                float64                  `json:"price"                 validate:"omitempty,gt=0"`
	Capacity             int                      `json:"capacity"              validate:"omitempty,gt=0"`
	Description          string                   `json:"description"           validate:"omitempty,min=10,max=300"`
	SpaceInfo            string                   `json:"space_info"            validate:"omitempty,min=10,max=300"`
	BedDistribution      string                   `json:"bed_distribution"      validate:"omitempty,min=5,max=100"`
	Quantity             int                      `json:"quantity"              validate:"omitempty,gt=0"`
	HighlightedAmenities []HighlightedAmenityInput `json:"highlighted_amenities" validate:"omitempty,dive"`
	AmenityCategories    []AmenityCategoryInput    `json:"amenity_categories"    validate:"omitempty,dive"`
}

// UpdateRoomQuantityRequest is used by the reservation service to adjust quantity.
type UpdateRoomQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}

// FilterRoomsRequest contains query parameters for listing/filtering rooms.
type FilterRoomsRequest struct {
	HotelID     string   `json:"hotel_id"      validate:"omitempty"`
	Type        string   `json:"type"          validate:"omitempty"`
	Types       []string `json:"types"         validate:"omitempty"`
	MinCapacity int      `json:"min_capacity"  validate:"omitempty,gt=0"`
	MaxCapacity int      `json:"max_capacity"  validate:"omitempty,gt=0"`
	MinPrice    float64  `json:"min_price"     validate:"omitempty,gt=0"`
	MaxPrice    float64  `json:"max_price"     validate:"omitempty,gt=0"`
	MinCoef     float64  `json:"min_coef"      validate:"omitempty,gt=0"`
	MaxCoef     float64  `json:"max_coef"      validate:"omitempty,gt=0"`
	Limit       int      `json:"limit"         validate:"omitempty,gt=0"`
	Offset      int      `json:"offset"        validate:"omitempty,gte=0"`
}

// CheckAvailabilityRequest is used by the reservation service.
type CheckAvailabilityRequest struct {
	HotelID  string `json:"hotel_id"  validate:"required"`
	CheckIn  string `json:"check_in"  validate:"required"`
	CheckOut string `json:"check_out" validate:"required"`
	Quantity int    `json:"quantity"  validate:"required,gt=0"`
}

// AvailabilityResponse is returned by the CheckAvailability endpoint.
type AvailabilityResponse struct {
	Available bool `json:"available"`
	Count     int  `json:"count"`
}

// FileUpload holds an uploaded file's data for forwarding to the media service.
type FileUpload struct {
	Filename    string
	Content     []byte
	ContentType string
}
