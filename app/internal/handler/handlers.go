// Package handler provides HTTP request handlers, routing, and middleware.
// It handles incoming HTTP requests, delegates to the service layer for
// business logic, and returns JSON responses with appropriate status codes.
package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
	"hotel.com/app/internal/service"
)

type Handler struct {
	s       service.Service
	l       *slog.Logger
	jwtAuth *JWTAuthenticator
}

func New(s service.Service, l *slog.Logger, jwtAuth *JWTAuthenticator) *Handler {
	return &Handler{
		s:       s,
		l:       l,
		jwtAuth: jwtAuth,
	}
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// readinessCheck verifies if the service is ready to accept traffic
// by pinging the database and other critical dependencies.
func (h *Handler) readinessCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.s.Check(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready", "reason": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
		"db":     "ok",
	})
}

// CreateRoom handles room creation with multipart form data support
func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}
	if !strings.EqualFold(claims.UserType, "admin") {
		helper.RespondError(w, http.StatusForbidden, "your account does not have admin privileges")
		return
	}

	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	var req *models.CreateRoomRequest
	var files []models.FileUpload

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			helper.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		req = &models.CreateRoomRequest{
			Name:            r.FormValue("name"),
			Type:            r.FormValue("type"),
			Description:     r.FormValue("description"),
			SpaceInfo:       r.FormValue("space_info"),
			BedDistribution: r.FormValue("bed_distribution"),
			AmenityCategories: r.FormValue("amenity_categories"),
		}

		if priceStr := r.FormValue("price"); priceStr != "" {
			if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
				req.Price = price
			}
		}
		if capacityStr := r.FormValue("capacity"); capacityStr != "" {
			if capacity, err := strconv.Atoi(capacityStr); err == nil {
				req.Capacity = capacity
			}
		}
		if quantityStr := r.FormValue("quantity"); quantityStr != "" {
			if quantity, err := strconv.Atoi(quantityStr); err == nil {
				req.Quantity = quantity
			}
		}

		// Parse highlighted amenities from form if provided
		if amenitiesStr := r.FormValue("highlighted_amenities"); amenitiesStr != "" {
			json.Unmarshal([]byte(amenitiesStr), &req.HighlightedAmenities)
		}

		// Extract files
		if r.MultipartForm != nil {
			for _, fileHeaders := range r.MultipartForm.File {
				for _, header := range fileHeaders {
					file, err := header.Open()
					if err != nil {
						h.l.Error("failed to open uploaded file", "error", err)
						helper.RespondError(w, http.StatusBadRequest, "failed to open uploaded file")
						return
					}
					defer file.Close()

					content := make([]byte, header.Size)
					if _, err := io.ReadFull(file, content); err != nil {
						h.l.Error("failed to read uploaded file", "error", err)
						helper.RespondError(w, http.StatusBadRequest, "failed to read uploaded file")
						return
					}

					files = append(files, models.FileUpload{
						Filename:    header.Filename,
						Content:     content,
						ContentType: header.Header.Get("Content-Type"),
					})
				}
			}
		}
	} else {
		req = &models.CreateRoomRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
			return
		}
	}

	// Validate required fields
	if req.Name == "" || req.Type == "" || req.Price <= 0 || req.Capacity <= 0 {
		helper.RespondError(w, http.StatusBadRequest, "name, type, price, and capacity are required")
		return
	}

	// Default quantity to 1 if not provided
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	rooms, err := h.s.CreateRooms(r.Context(), req, hotelID, files)
	if err != nil {
		h.l.Error("failed to create rooms", "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	helper.RespondJSON(w, http.StatusCreated, rooms)
}

// GetRoom handles getting a room by ID
func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	room, err := h.s.GetRoomByID(r.Context(), id)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		h.l.Error("failed to get room", "id", id, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// UpdateRoom handles room updates with multipart form data support
func (h *Handler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}
	if !strings.EqualFold(claims.UserType, "admin") {
		helper.RespondError(w, http.StatusForbidden, "your account does not have admin privileges")
		return
	}

	var req *models.UpdateRoomRequest
	var files []models.FileUpload

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			helper.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		req = &models.UpdateRoomRequest{}
		if name := r.FormValue("name"); name != "" {
			req.Name = name
		}
		if roomType := r.FormValue("type"); roomType != "" {
			req.Type = roomType
		}
		if desc := r.FormValue("description"); desc != "" {
			req.Description = desc
		}
		if spaceInfo := r.FormValue("space_info"); spaceInfo != "" {
			req.SpaceInfo = spaceInfo
		}
		if bedDist := r.FormValue("bed_distribution"); bedDist != "" {
			req.BedDistribution = bedDist
		}
		if amenityCat := r.FormValue("amenity_categories"); amenityCat != "" {
			req.AmenityCategories = amenityCat
		}
		if priceStr := r.FormValue("price"); priceStr != "" {
			if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
				req.Price = price
			}
		}
		if capacityStr := r.FormValue("capacity"); capacityStr != "" {
			if capacity, err := strconv.Atoi(capacityStr); err == nil {
				req.Capacity = capacity
			}
		}
		if quantityStr := r.FormValue("quantity"); quantityStr != "" {
			if quantity, err := strconv.Atoi(quantityStr); err == nil {
				req.Quantity = quantity
			}
		}

		// Parse highlighted amenities from form if provided
		if amenitiesStr := r.FormValue("highlighted_amenities"); amenitiesStr != "" {
			json.Unmarshal([]byte(amenitiesStr), &req.HighlightedAmenities)
		}

		// Extract files
		if r.MultipartForm != nil {
			for _, fileHeaders := range r.MultipartForm.File {
				for _, header := range fileHeaders {
					file, err := header.Open()
					if err != nil {
						h.l.Error("failed to open uploaded file", "error", err)
						helper.RespondError(w, http.StatusBadRequest, "failed to open uploaded file")
						return
					}
					defer file.Close()

					content := make([]byte, header.Size)
					if _, err := io.ReadFull(file, content); err != nil {
						h.l.Error("failed to read uploaded file", "error", err)
						helper.RespondError(w, http.StatusBadRequest, "failed to read uploaded file")
						return
					}

					files = append(files, models.FileUpload{
						Filename:    header.Filename,
						Content:     content,
						ContentType: header.Header.Get("Content-Type"),
					})
				}
			}
		}
	} else {
		req = &models.UpdateRoomRequest{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
			return
		}
	}

	room, err := h.s.UpdateRoom(r.Context(), id, req, hotelID, files)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		h.l.Error("failed to update room", "id", id, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// DeleteRoom handles room deletion
func (h *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}
	if !strings.EqualFold(claims.UserType, "admin") {
		helper.RespondError(w, http.StatusForbidden, "your account does not have admin privileges")
		return
	}

	err := h.s.DeleteRoom(r.Context(), id, hotelID)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		h.l.Error("failed to delete room", "id", id, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListRooms handles listing rooms with filters
func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	hotelID := r.URL.Query().Get("hotel_id")

	filter := &models.FilterRoomsRequest{
		HotelID:     hotelID,
		Type:        r.URL.Query().Get("type"),
		MinCapacity: parseInt(r.URL.Query().Get("min_capacity")),
		MaxCapacity: parseInt(r.URL.Query().Get("max_capacity")),
		MinPrice:    parseFloat(r.URL.Query().Get("min_price")),
		MaxPrice:    parseFloat(r.URL.Query().Get("max_price")),
		MinCoef:     parseFloat(r.URL.Query().Get("min_coef")),
		MaxCoef:     parseFloat(r.URL.Query().Get("max_coef")),
		Limit:       parseIntOrDefault(r.URL.Query().Get("limit"), 20),
		Offset:      parseIntOrDefault(r.URL.Query().Get("offset"), 0),
	}

	typeParam := r.URL.Query().Get("type")
	if typeParam != "" {
		filter.Types = strings.Split(typeParam, ",")
	}

	rooms, total, err := h.s.ListRooms(r.Context(), filter)
	if err != nil {
		h.l.Error("failed to list rooms", "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	response := models.RoomListResponse{
		Rooms:      convertToRoomResponse(rooms),
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListRoomsByHotel handles listing rooms by hotel ID
func (h *Handler) ListRoomsByHotel(w http.ResponseWriter, r *http.Request) {
	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	rooms, err := h.s.ListRoomsByHotel(r.Context(), hotelID)
	if err != nil {
		h.l.Error("failed to list rooms by hotel", "hotel_id", hotelID, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	response := models.RoomListResponse{
		Rooms:      convertToRoomResponse(rooms),
		TotalCount: len(rooms),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckAvailability handles checking room availability by type and name
func (h *Handler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	roomType := r.URL.Query().Get("type")
	if roomType == "" {
		helper.RespondError(w, http.StatusBadRequest, "room type is required")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		helper.RespondError(w, http.StatusBadRequest, "room name is required")
		return
	}

	response, err := h.s.CheckAvailability(r.Context(), hotelID, roomType, name)
	if err != nil {
		h.l.Error("failed to check availability", "hotel_id", hotelID, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateRoomQuantity handles updating room quantity
func (h *Handler) UpdateRoomQuantity(w http.ResponseWriter, r *http.Request) {
	hotelID := r.PathValue("hotel_id")
	if hotelID == "" {
		helper.RespondError(w, http.StatusBadRequest, "hotel ID is required")
		return
	}

	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}
	if !strings.EqualFold(claims.UserType, "admin") {
		helper.RespondError(w, http.StatusForbidden, "your account does not have admin privileges")
		return
	}

	roomType := r.PathValue("type")
	if roomType == "" {
		helper.RespondError(w, http.StatusBadRequest, "room type is required")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		helper.RespondError(w, http.StatusBadRequest, "room name is required")
		return
	}

	var req models.UpdateRoomQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
		return
	}

	err := h.s.UpdateRoomQuantity(r.Context(), hotelID, roomType, name, req.Quantity)
	if err != nil {
		h.l.Error("failed to update room quantity", "hotel_id", hotelID, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func convertToRoomResponse(rooms []models.Room) []models.RoomResponse {
	response := make([]models.RoomResponse, len(rooms))
	for i, room := range rooms {
		response[i] = models.RoomResponse{
			ID:                   room.ID,
			HotelID:              room.HotelID,
			Name:                 room.Name,
			Type:                 room.Type,
			Price:                room.Price,
			Capacity:             room.Capacity,
			Description:          room.Description,
			SpaceInfo:            room.SpaceInfo,
			BedDistribution:      room.BedDistribution,
			Quantity:             room.Quantity,
			HighlightedAmenities: room.HighlightedAmenities,
			AmenityCategories:    room.AmenityCategories,
			AmenityCount:         room.AmenityCount,
			RecommendationCoef:   room.RecommendationCoef,
			CreatedAt:            room.CreatedAt,
			UpdatedAt:            room.UpdatedAt,
		}
	}
	return response
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}

func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return i
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
