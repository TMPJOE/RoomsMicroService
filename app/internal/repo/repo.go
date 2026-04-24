// Package repo implements the data access layer of the application.
// It handles all database queries, transactions, and data mapping,
// providing a clean interface for the service layer to interact with PostgreSQL.
package repo

import (
	"context"

	"hotel.com/app/internal/models"
)

// RoomRepository defines the interface for room data access
type RoomRepository interface {
	DbPing() error
	// CRUD operations
	CreateRoom(ctx context.Context, room *models.Room) error
	GetRoomByID(ctx context.Context, id string) (*models.Room, error)
	UpdateRoom(ctx context.Context, room *models.Room) error
	DeleteRoom(ctx context.Context, id string) error

	// Query operations
	ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)
	ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error)
	GetRoomsByFilters(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)

	// Availability
	CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error)

	// Amenity operations
	UpdateAmenities(ctx context.Context, id string, amenities []models.HighlightedAmenity, amenityCategories string, amenityCount int, recommendationCoef float64) error
}

//REMEMBER TRANSACTION CODE LOGIC
