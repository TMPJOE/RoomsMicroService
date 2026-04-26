// Package service contains the business logic layer of the application.
// It defines service interfaces and implements use cases by orchestrating
// repositories, applying business rules, and returning results to handlers.
package service

import (
	"bytes"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"hotel.com/app/internal/client"
	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
	"hotel.com/app/internal/repo"
)

// Service defines the interface for room business logic.
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
}

type roomService struct {
	l  *slog.Logger
	r  repo.RoomRepository
	mc client.MediaClient
}

func New(l *slog.Logger, r repo.RoomRepository, mc client.MediaClient) Service {
	return &roomService{l: l, r: r, mc: mc}
}

func (s *roomService) Check() error {
	s.l.Info("Pinging db...")
	return s.r.DbPing()
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

// CreateRooms creates Quantity copies of a room and persists their amenities.
func (s *roomService) CreateRooms(ctx context.Context, req *models.CreateRoomRequest, hotelID string, files []models.FileUpload) ([]*models.Room, error) {
	now := time.Now()

	// Build highlighted amenity objects (assign IDs server-side)
	highlightedAmenities := make([]models.HighlightedAmenity, len(req.HighlightedAmenities))
	for i, a := range req.HighlightedAmenities {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		highlightedAmenities[i] = models.HighlightedAmenity{
			ID:        id.String(),
			Icon:      a.Icon,
			Text:      a.Text,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	// Build category objects (assign IDs server-side)
	amenityCategories := make([]models.AmenityCategory, len(req.AmenityCategories))
	totalAmenityCount := 0
	for i, c := range req.AmenityCategories {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		amenityCategories[i] = models.AmenityCategory{
			ID:           id.String(),
			Name:         c.Name,
			Description:  c.Description,
			Tier:         c.Tier,
			AmenityCount: c.AmenityCount,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		totalAmenityCount += c.AmenityCount
	}

	recommendationCoef := calculateCoef(highlightedAmenities, amenityCategories)

	// Create Quantity room copies
	rooms := make([]*models.Room, req.Quantity)
	for i := 0; i < req.Quantity; i++ {
		roomID, err := uuid.NewV7()
		if err != nil {
			return nil, helper.ErrInternalServer
		}
		rooms[i] = &models.Room{
			ID:                   roomID.String(),
			HotelID:              hotelID,
			Name:                 req.Name,
			Type:                 req.Type,
			Price:                req.Price,
			Capacity:             req.Capacity,
			Description:          req.Description,
			SpaceInfo:            req.SpaceInfo,
			BedDistribution:      req.BedDistribution,
			Quantity:             req.Quantity,
			AmenityCount:         totalAmenityCount,
			RecommendationCoef:   recommendationCoef,
			HighlightedAmenities: highlightedAmenities,
			AmenityCategories:    amenityCategories,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
	}

	if err := s.r.CreateRooms(ctx, rooms); err != nil {
		s.l.Error("failed to create rooms", "error", err)
		return nil, helper.ErrCreateFailed
	}

	// Persist amenities for each created room
	for _, room := range rooms {
		// Clone amenities with fresh IDs per room copy
		roomAmenities := cloneHighlightedWithRoomID(highlightedAmenities, room.ID)
		roomCategories := cloneCategoriesWithRoomID(amenityCategories, room.ID)

		if err := s.r.UpsertHighlightedAmenities(ctx, room.ID, roomAmenities); err != nil {
			s.l.Error("failed to upsert highlighted amenities", "room_id", room.ID, "error", err)
			return nil, helper.ErrCreateFailed
		}
		if err := s.r.UpsertAmenityCategories(ctx, room.ID, roomCategories); err != nil {
			s.l.Error("failed to upsert amenity categories", "room_id", room.ID, "error", err)
			return nil, helper.ErrCreateFailed
		}
	}

	// Upload images to media service (associated with first room as representative)
	if len(files) > 0 && len(rooms) > 0 {
		for _, file := range files {
			if _, err := s.mc.UploadFile(ctx, bytes.NewReader(file.Content), file.Filename, "room", rooms[0].ID, file.ContentType); err != nil {
				s.l.Error("failed to upload file", "error", err, "filename", file.Filename)
				// Non-fatal: room is already created
			}
		}
	}

	return rooms, nil
}

// GetRoomByID retrieves a room by ID including its amenities.
func (s *roomService) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	room, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		s.l.Error("failed to get room", "id", id, "error", err)
		return nil, helper.ErrFetchFailed
	}
	return room, nil
}

// UpdateRoom applies partial updates to a room and replaces its amenities.
func (s *roomService) UpdateRoom(ctx context.Context, id string, req *models.UpdateRoomRequest, hotelID string, files []models.FileUpload) (*models.Room, error) {
	existing, err := s.r.GetRoomByID(ctx, id)
	if err != nil {
		return nil, helper.ErrRecordNotFound
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Price > 0 {
		existing.Price = req.Price
	}
	if req.Capacity > 0 {
		existing.Capacity = req.Capacity
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.SpaceInfo != "" {
		existing.SpaceInfo = req.SpaceInfo
	}
	if req.BedDistribution != "" {
		existing.BedDistribution = req.BedDistribution
	}
	if req.Quantity > 0 {
		existing.Quantity = req.Quantity
	}

	now := time.Now()

	// Replace amenities if provided
	if req.HighlightedAmenities != nil {
		newAmenities := make([]models.HighlightedAmenity, len(req.HighlightedAmenities))
		for i, a := range req.HighlightedAmenities {
			aid, err := uuid.NewV7()
			if err != nil {
				return nil, helper.ErrInternalServer
			}
			newAmenities[i] = models.HighlightedAmenity{
				ID: aid.String(), RoomID: id,
				Icon: a.Icon, Text: a.Text,
				CreatedAt: now, UpdatedAt: now,
			}
		}
		existing.HighlightedAmenities = newAmenities
	}

	if req.AmenityCategories != nil {
		newCategories := make([]models.AmenityCategory, len(req.AmenityCategories))
		totalCount := 0
		for i, c := range req.AmenityCategories {
			cid, err := uuid.NewV7()
			if err != nil {
				return nil, helper.ErrInternalServer
			}
			newCategories[i] = models.AmenityCategory{
				ID: cid.String(), RoomID: id,
				Name: c.Name, Description: c.Description,
				Tier: c.Tier, AmenityCount: c.AmenityCount,
				CreatedAt: now, UpdatedAt: now,
			}
			totalCount += c.AmenityCount
		}
		existing.AmenityCategories = newCategories
		existing.AmenityCount = totalCount
	}

	existing.RecommendationCoef = calculateCoef(existing.HighlightedAmenities, existing.AmenityCategories)

	if err := s.r.UpdateRoom(ctx, existing); err != nil {
		s.l.Error("failed to update room", "id", id, "error", err)
		return nil, helper.ErrUpdateFailed
	}

	if req.HighlightedAmenities != nil {
		if err := s.r.UpsertHighlightedAmenities(ctx, id, existing.HighlightedAmenities); err != nil {
			s.l.Error("failed to upsert highlighted amenities", "room_id", id, "error", err)
			return nil, helper.ErrUpdateFailed
		}
	}
	if req.AmenityCategories != nil {
		if err := s.r.UpsertAmenityCategories(ctx, id, existing.AmenityCategories); err != nil {
			s.l.Error("failed to upsert amenity categories", "room_id", id, "error", err)
			return nil, helper.ErrUpdateFailed
		}
	}

	if len(files) > 0 {
		for _, file := range files {
			if _, err := s.mc.UploadFile(ctx, bytes.NewReader(file.Content), file.Filename, "room", id, file.ContentType); err != nil {
				s.l.Error("failed to upload file", "error", err, "filename", file.Filename)
			}
		}
	}

	return existing, nil
}

// DeleteRoom removes a room by ID. Child amenity rows are deleted by CASCADE.
func (s *roomService) DeleteRoom(ctx context.Context, id string, hotelID string) error {
	if _, err := s.r.GetRoomByID(ctx, id); err != nil {
		return helper.ErrRecordNotFound
	}
	if err := s.r.DeleteRoom(ctx, id); err != nil {
		s.l.Error("failed to delete room", "id", id, "error", err)
		return helper.ErrDeleteFailed
	}
	return nil
}

// ─── Query ────────────────────────────────────────────────────────────────────

func (s *roomService) ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	rooms, total, err := s.r.ListRooms(ctx, filter)
	if err != nil {
		s.l.Error("failed to list rooms", "error", err)
		return nil, 0, helper.ErrFetchFailed
	}
	return rooms, total, nil
}

func (s *roomService) ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error) {
	rooms, err := s.r.ListRoomsByHotel(ctx, hotelID)
	if err != nil {
		s.l.Error("failed to list rooms by hotel", "hotel_id", hotelID, "error", err)
		return nil, helper.ErrFetchFailed
	}
	return rooms, nil
}

// ─── Availability ─────────────────────────────────────────────────────────────

func (s *roomService) CheckAvailability(ctx context.Context, hotelID, roomType, name string) (*models.AvailabilityResponse, error) {
	count, err := s.r.CheckAvailabilityByType(ctx, hotelID, roomType, name)
	if err != nil {
		s.l.Error("failed to check availability", "hotel_id", hotelID, "error", err)
		return nil, helper.ErrFetchFailed
	}
	return &models.AvailabilityResponse{Available: count > 0, Count: count}, nil
}

// ─── Quantity ─────────────────────────────────────────────────────────────────

func (s *roomService) UpdateRoomQuantity(ctx context.Context, hotelID, roomType, name string, quantity int) error {
	if err := s.r.UpdateRoomQuantity(ctx, hotelID, roomType, name, quantity); err != nil {
		s.l.Error("failed to update room quantity", "hotel_id", hotelID, "error", err)
		return helper.ErrUpdateFailed
	}
	return nil
}

// ─── Coef calculation ─────────────────────────────────────────────────────────

// calculateCoef computes the recommendation coefficient from a room's amenities.
//
//	highlighted_sum = Σ IconMultipliers[icon]
//	category_sum    = Σ TierMultipliers[tier] × category.AmenityCount
//	coef            = highlighted_sum × max(category_sum, 1.0)
func calculateCoef(amenities []models.HighlightedAmenity, categories []models.AmenityCategory) float64 {
	highlightedSum := 0.0
	for _, a := range amenities {
		if m, ok := models.IconMultipliers[a.Icon]; ok {
			highlightedSum += m
		}
	}
	if highlightedSum == 0 {
		highlightedSum = 1.0
	}

	categorySum := 0.0
	for _, c := range categories {
		if m, ok := models.TierMultipliers[c.Tier]; ok {
			categorySum += m * float64(c.AmenityCount)
		}
	}
	if categorySum == 0 {
		categorySum = 1.0
	}

	return highlightedSum * categorySum
}

// ─── Private helpers ──────────────────────────────────────────────────────────

// cloneHighlightedWithRoomID creates a copy of highlighted amenities with each
// entry assigned the given roomID (necessary when creating multiple room copies).
func cloneHighlightedWithRoomID(src []models.HighlightedAmenity, roomID string) []models.HighlightedAmenity {
	now := time.Now()
	out := make([]models.HighlightedAmenity, len(src))
	for i, a := range src {
		id, _ := uuid.NewV7()
		out[i] = models.HighlightedAmenity{
			ID: id.String(), RoomID: roomID,
			Icon: a.Icon, Text: a.Text,
			CreatedAt: now, UpdatedAt: now,
		}
	}
	return out
}

// cloneCategoriesWithRoomID creates a copy of amenity categories with each
// entry assigned the given roomID.
func cloneCategoriesWithRoomID(src []models.AmenityCategory, roomID string) []models.AmenityCategory {
	now := time.Now()
	out := make([]models.AmenityCategory, len(src))
	for i, c := range src {
		id, _ := uuid.NewV7()
		out[i] = models.AmenityCategory{
			ID: id.String(), RoomID: roomID,
			Name: c.Name, Description: c.Description,
			Tier: c.Tier, AmenityCount: c.AmenityCount,
			CreatedAt: now, UpdatedAt: now,
		}
	}
	return out
}
