// Package repo implements the data access layer of the application.
// It handles all database queries, transactions, and data mapping,
// providing a clean interface for the service layer to interact with PostgreSQL.
package repo

import (
	"context"

	"hotel.com/app/internal/models"
)

// RoomRepository defines the interface for room data access.
type RoomRepository interface {
	DbPing() error

	// CRUD operations
	CreateRoom(ctx context.Context, room *models.Room) error
	CreateRooms(ctx context.Context, rooms []*models.Room) error
	GetRoomByID(ctx context.Context, id string) (*models.Room, error)
	UpdateRoom(ctx context.Context, room *models.Room) error
	DeleteRoom(ctx context.Context, id string) error

	// Query operations
	ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)
	ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error)
	GetRoomsByFilters(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)

	// Availability
	CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error)
	CheckAvailabilityByType(ctx context.Context, hotelID, roomType, name string) (int, error)

	// Quantity operations
	UpdateRoomQuantity(ctx context.Context, hotelID, roomType, name string, quantity int) error

	// Highlighted amenity operations (replace-all strategy: delete then insert)
	UpsertHighlightedAmenities(ctx context.Context, roomID string, amenities []models.HighlightedAmenity) error
	GetHighlightedAmenitiesByRooms(ctx context.Context, roomIDs []string) (map[string][]models.HighlightedAmenity, error)

	// Amenity category operations (replace-all strategy: delete then insert)
	UpsertAmenityCategories(ctx context.Context, roomID string, categories []models.AmenityCategory) error
	GetAmenityCategoriesByRooms(ctx context.Context, roomIDs []string) (map[string][]models.AmenityCategory, error)
}
