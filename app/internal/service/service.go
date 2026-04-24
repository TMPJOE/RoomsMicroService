// Package service contains the business logic layer of the application.
// It defines service interfaces and implements use cases by orchestrating
// repositories, applying business rules, and returning results to handlers.
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
	"hotel.com/app/internal/repo"
)

// RoomService defines the interface for room business logic
type Service interface {
	Check() error
	// CRUD operations
	CreateRoom(ctx context.Context, req *models.CreateRoomRequest, hotelID string) (*models.Room, error)
	GetRoomByID(ctx context.Context, id string) (*models.Room, error)
	UpdateRoom(ctx context.Context, id string, req *models.UpdateRoomRequest, hotelID string) (*models.Room, error)
	DeleteRoom(ctx context.Context, id string, hotelID string) error

	// Query operations
	ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error)
	ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error)

	// Availability
	CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error)

	// Amenity operations
	CalculateRecommendationCoef(amenities []models.HighlightedAmenityInput, amenityCategories string, description string) float64
}

type roomService struct {
	l *slog.Logger
	r repo.RoomRepository
}

func (s *roomService) Check() error {
	s.l.Info("Pinging db...")
	err := s.r.DbPing()
	s.l.Info("is service working", "err", err.Error())
	return err
}

func New(l *slog.Logger, r repo.RoomRepository) Service {
	return &roomService{
		l: l,
		r: r,
	}
}

// CreateRoom creates a new room
func (s *roomService) CreateRoom(ctx context.Context, req *models.CreateRoomRequest, hotelID string) (*models.Room, error) {

	roomID := uuid.New()

	room := &models.Room{
		ID:                roomID.String(),
		HotelID:           hotelID,
		Name:              req.Name,
		Type:              req.Type,
		Price:             req.Price,
		Capacity:          req.Capacity,
		Description:       req.Description,
		SpaceInfo:         req.SpaceInfo,
		BedDistribution:   req.BedDistribution,
		Quantity:          req.Quantity,
		AmenityCategories: req.AmenityCategories,
	}

	if req.HighlightedAmenities != nil {
		amenitiesJSON, err := s.convertAmenitiesToJSON(req.HighlightedAmenities)
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		room.HighlightedAmenities = amenitiesJSON
	}

	amenityCount := s.countAmenitiesFromDescription(req.Description)
	recommendationCoef := s.CalculateRecommendationCoef(req.HighlightedAmenities, req.AmenityCategories, req.Description)

	room.AmenityCount = amenityCount
	room.RecommendationCoef = recommendationCoef

	if err := s.r.CreateRoom(ctx, room); err != nil {
		s.l.Error("failed to create room", "error", err)
		return nil, helper.ErrCreateFailed
	}

	return room, nil
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
func (s *roomService) UpdateRoom(ctx context.Context, id string, req *models.UpdateRoomRequest, hotelID string) (*models.Room, error) {
	existingRoom, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		return nil, helper.ErrRecordNotFound
	}

	// Skip hotelID check for now as ownership is not enforced

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

	amenityCount := s.countAmenitiesFromDescription(existingRoom.Description)
	recommendationCoef := s.calculateCoefFromAmenities(req.HighlightedAmenities, req.AmenityCategories, amenityCount)

	existingRoom.AmenityCount = amenityCount
	existingRoom.RecommendationCoef = recommendationCoef

	if err := s.r.UpdateRoom(ctx, existingRoom); err != nil {
		s.l.Error("failed to update room", "id", id, "error", err)
		return nil, helper.ErrUpdateFailed
	}

	return existingRoom, nil
}

// DeleteRoom deletes a room by ID
func (s *roomService) DeleteRoom(ctx context.Context, id string, hotelID string) error {
	_, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		return helper.ErrRecordNotFound
	}

	// Skip hotelID check for now as ownership is not enforced

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

// CheckAvailability checks room availability
func (s *roomService) CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error) {
	rooms, err := s.r.CheckAvailability(ctx, hotelID, checkIn, checkOut, quantity)
	if err != nil {
		s.l.Error("failed to check availability", "hotel_id", hotelID, "error", err)
		return nil, helper.ErrFetchFailed
	}

	return rooms, nil
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
