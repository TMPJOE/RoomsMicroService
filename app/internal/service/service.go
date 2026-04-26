// Package service contains the business logic layer of the application.
// It defines service interfaces and implements use cases by orchestrating
// repositories, applying business rules, and returning results to handlers.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"hotel.com/app/internal/client"
	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
	"hotel.com/app/internal/repo"
)

// RoomService defines the interface for room business logic
type Service interface {
	Check() error
	// CRUD operations
	CreateRooms(ctx context.Context, req *models.CreateRoomRequest, hotelID string, files []models.FileUpload) ([]*models.Room, error)
	GetRoomByID(ctx context.Context, id string) (*models.Room, error)
	UpdateRoom(ctx context.Context, id string, req *models.UpdateRoomRequest, hotelID string, files []models.FileUpload) (*models.Room, error)
	DeleteRoom(ctx context.Context, id string, hotelID string) error

	// Query operations
	ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)
	ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error)

	// Availability
	CheckAvailability(ctx context.Context, hotelID, roomType, name string) (*models.AvailabilityResponse, error)

	// Quantity operations
	UpdateRoomQuantity(ctx context.Context, hotelID, roomType, name string, quantity int) error

	// Amenity operations
	CalculateRecommendationCoef(amenities []models.HighlightedAmenityInput, amenityCategories string, description string) float64
}

type roomService struct {
	l  *slog.Logger
	r  repo.RoomRepository
	mc client.MediaClient
}

func (s *roomService) Check() error {
	s.l.Info("Pinging db...")
	err := s.r.DbPing()
	s.l.Info("is service working", "err", err.Error())
	return err
}

func New(l *slog.Logger, r repo.RoomRepository, mc client.MediaClient) Service {
	return &roomService{
		l:  l,
		r:  r,
		mc: mc,
	}
}

// CreateRooms creates multiple rooms based on the Quantity field
func (s *roomService) CreateRooms(ctx context.Context, req *models.CreateRoomRequest, hotelID string, files []models.FileUpload) ([]*models.Room, error) {
	// Calculate recommendation coefficient once for all rooms
	amenityCount := s.countAmenitiesFromDescription(req.Description)
	recommendationCoef := s.CalculateRecommendationCoef(req.HighlightedAmenities, req.AmenityCategories, req.Description)

	// Prepare highlighted amenities JSON
	var highlightedAmenitiesJSON []byte
	if req.HighlightedAmenities != nil {
		var err error
		highlightedAmenitiesJSON, err = s.convertAmenitiesToJSON(req.HighlightedAmenities)
		if err != nil {
			return nil, helper.ErrInternalServer
		}
	}

	// Create room objects
	rooms := make([]*models.Room, req.Quantity)
	for i := 0; i < req.Quantity; i++ {
		roomID, err := uuid.NewV7()
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		rooms[i] = &models.Room{
			ID: roomID.String(),
			HotelID:              hotelID,
			Name:                 req.Name,
			Type:                 req.Type,
			Price:                req.Price,
			Capacity:             req.Capacity,
			Description:          req.Description,
			SpaceInfo:            req.SpaceInfo,
			BedDistribution:      req.BedDistribution,
			Quantity:             req.Quantity,
			HighlightedAmenities: highlightedAmenitiesJSON,
			AmenityCategories:    req.AmenityCategories,
			AmenityCount:         amenityCount,
			RecommendationCoef:   recommendationCoef,
		}
	}

	// Save all rooms to database
	if err := s.r.CreateRooms(ctx, rooms); err != nil {
		s.l.Error("failed to create rooms", "error", err)
		return nil, helper.ErrCreateFailed
	}

	// Upload files to media service (associate with the first room as representative)
	if len(files) > 0 && len(rooms) > 0 {
		for _, file := range files {
			_, err := s.mc.UploadFile(ctx, bytes.NewReader(file.Content), file.Filename, "room", rooms[0].ID, file.ContentType)
			if err != nil {
				s.l.Error("failed to upload file to media service", "error", err, "filename", file.Filename)
				// Don't return error, just log it - room is already created
			}
		}
	}

	return rooms, nil
}

// GetRoomByID retrieves a room by ID
func (s *roomService) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	room, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		s.l.Error("failed to get room", "id", id, "error", err)
		return nil, helper.ErrFetchFailed
	}

	return room, nil
}

// UpdateRoom updates an existing room
func (s *roomService) UpdateRoom(ctx context.Context, id string, req *models.UpdateRoomRequest, hotelID string, files []models.FileUpload) (*models.Room, error) {
	existingRoom, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		return nil, helper.ErrRecordNotFound
	}

	if req.Name != "" {
		existingRoom.Name = req.Name
	}
	if req.Type != "" {
		existingRoom.Type = req.Type
	}
	if req.Price > 0 {
		existingRoom.Price = req.Price
	}
	if req.Capacity > 0 {
		existingRoom.Capacity = req.Capacity
	}
	if req.Description != "" {
		existingRoom.Description = req.Description
	}
	if req.SpaceInfo != "" {
		existingRoom.SpaceInfo = req.SpaceInfo
	}
	if req.BedDistribution != "" {
		existingRoom.BedDistribution = req.BedDistribution
	}
	if req.Quantity > 0 {
		existingRoom.Quantity = req.Quantity
	}
	if req.AmenityCategories != "" {
		existingRoom.AmenityCategories = req.AmenityCategories
	}

	if req.HighlightedAmenities != nil {
		amenitiesJSON, err := s.convertAmenitiesToJSON(req.HighlightedAmenities)
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		existingRoom.HighlightedAmenities = amenitiesJSON
	}

	// Recalculate recommendation coefficient
	amenityCount := s.countAmenitiesFromDescription(existingRoom.Description)
	recommendationCoef := s.calculateCoefFromAmenities(req.HighlightedAmenities, req.AmenityCategories, amenityCount)

	existingRoom.AmenityCount = amenityCount
	existingRoom.RecommendationCoef = recommendationCoef

	if err := s.r.UpdateRoom(ctx, existingRoom); err != nil {
		s.l.Error("failed to update room", "id", id, "error", err)
		return nil, helper.ErrUpdateFailed
	}

	// Upload new files to media service if provided
	if len(files) > 0 {
		for _, file := range files {
			_, err := s.mc.UploadFile(ctx, bytes.NewReader(file.Content), file.Filename, "room", existingRoom.ID, file.ContentType)
			if err != nil {
				s.l.Error("failed to upload file to media service", "error", err, "filename", file.Filename)
				// Don't return error, just log it - room is already updated
			}
		}
	}

	return existingRoom, nil
}

// DeleteRoom deletes a room by ID
func (s *roomService) DeleteRoom(ctx context.Context, id string, hotelID string) error {
	_, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		return helper.ErrRecordNotFound
	}

	if err := s.r.DeleteRoom(ctx, id); err != nil {
		s.l.Error("failed to delete room", "id", id, "error", err)
		return helper.ErrDeleteFailed
	}

	return nil
}

// ListRooms retrieves rooms with filters
func (s *roomService) ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	rooms, total, err := s.r.ListRooms(ctx, filter)
	if err != nil {
		s.l.Error("failed to list rooms", "error", err)
		return nil, 0, helper.ErrFetchFailed
	}

	return rooms, total, nil
}

// ListRoomsByHotel retrieves all rooms for a hotel
func (s *roomService) ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error) {
	rooms, err := s.r.ListRoomsByHotel(ctx, hotelID)
	if err != nil {
		s.l.Error("failed to list rooms by hotel", "hotel_id", hotelID, "error", err)
		return nil, helper.ErrFetchFailed
	}

	return rooms, nil
}

// CheckAvailability checks room availability by type and name
func (s *roomService) CheckAvailability(ctx context.Context, hotelID, roomType, name string) (*models.AvailabilityResponse, error) {
	count, err := s.r.CheckAvailabilityByType(ctx, hotelID, roomType, name)
	if err != nil {
		s.l.Error("failed to check availability", "hotel_id", hotelID, "error", err)
		return nil, helper.ErrFetchFailed
	}

	return &models.AvailabilityResponse{
		Available: count > 0,
		Count:     count,
	}, nil
}

// UpdateRoomQuantity updates the quantity of rooms matching hotel, type and name
func (s *roomService) UpdateRoomQuantity(ctx context.Context, hotelID, roomType, name string, quantity int) error {
	if err := s.r.UpdateRoomQuantity(ctx, hotelID, roomType, name, quantity); err != nil {
		s.l.Error("failed to update room quantity", "hotel_id", hotelID, "error", err)
		return helper.ErrUpdateFailed
	}

	return nil
}

// CalculateRecommendationCoef calculates the recommendation coefficient
func (s *roomService) CalculateRecommendationCoef(amenities []models.HighlightedAmenityInput, amenityCategories string, description string) float64 {
	amenityCount := s.countAmenitiesFromDescription(description)
	categoryCount := s.countCategories(amenityCategories)
	highlightedSum := s.sumHighlightedMultipliers(amenities)

	if highlightedSum == 0 {
		highlightedSum = 1.0
	}
	if categoryCount == 0 {
		categoryCount = 1
	}

	coef := highlightedSum * float64(categoryCount) * float64(amenityCount)
	return coef
}

func (s *roomService) calculateCoefFromAmenities(amenities []models.HighlightedAmenityInput, amenityCategories string, amenityCount int) float64 {
	categoryCount := s.countCategories(amenityCategories)
	highlightedSum := s.sumHighlightedMultipliers(amenities)

	if highlightedSum == 0 {
		highlightedSum = 1.0
	}
	if categoryCount == 0 {
		categoryCount = 1
	}

	coef := highlightedSum * float64(categoryCount) * float64(amenityCount)
	return coef
}

func (s *roomService) sumHighlightedMultipliers(amenities []models.HighlightedAmenityInput) float64 {
	multipliers := map[string]float64{
		"wifi":       1.5,
		"ac":         1.2,
		"tv":         1.1,
		"coffee":     1.1,
		"sofa":       1.2,
		"utensils":   1.3,
		"headphones": 1.1,
		"sparkle":    1.4,
		"briefcase":  1.1,
		"mountain":   1.3,
		"lock":       1.2,
		"info":       1.0,
		"bed":        1.5,
	}

	sum := 0.0
	for _, amenity := range amenities {
		icon := strings.ToLower(amenity.Icon)
		if multiplier, exists := multipliers[icon]; exists {
			sum += multiplier
		}
	}

	if sum == 0 {
		return 1.0
	}

	return sum
}

func (s *roomService) countCategories(amenityCategories string) int {
	if amenityCategories == "" {
		return 0
	}
	categories := strings.Split(amenityCategories, ",")
	count := 0
	for _, cat := range categories {
		if strings.TrimSpace(cat) != "" {
			count++
		}
	}
	return count
}

func (s *roomService) countAmenitiesFromDescription(description string) int {
	if description == "" {
		return 0
	}

	count := 0
	parts := strings.Split(description, ":")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			count++
		}
	}

	if count == 0 {
		return 1
	}

	return count
}

func (s *roomService) convertAmenitiesToJSON(amenities []models.HighlightedAmenityInput) ([]byte, error) {
	var jsonAmenities []map[string]string
	for _, amenity := range amenities {
		jsonAmenity := map[string]string{
			"icon": amenity.Icon,
			"text": amenity.Text,
		}
		jsonAmenities = append(jsonAmenities, jsonAmenity)
	}

	return json.Marshal(jsonAmenities)
}

// Helper to convert io.ReadCloser to bytes (if needed)
func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
