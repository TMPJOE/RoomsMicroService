// Package handler provides HTTP request handlers, routing, and middleware.
// It handles incoming HTTP requests, delegates to the service layer for
// business logic, and returns JSON responses with appropriate status codes.
package handler

import (
	"encoding/json"
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

// CreateRoom handles room creation
func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}

	var req models.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
		return
	}

	room, err := h.s.Room().CreateRoom(r.Context(), &req, claims.UserID)
	if err != nil {
		h.l.Error("failed to create room", "error", err)
		if err == helper.ErrValidation {
			helper.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

// GetRoom handles getting a room by ID
func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	room, err := h.s.Room().GetRoomByID(r.Context(), id)
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

// UpdateRoom handles room updates
func (h *Handler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	claims := GetClaimsFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}

	var req models.UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
		return
	}

	room, err := h.s.Room().UpdateRoom(r.Context(), id, &req, claims.UserID)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		if err == helper.ErrPermissionDenied {
			helper.RespondError(w, http.StatusForbidden, helper.ErrPermissionDenied.Error())
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

	claims := GetClaimsFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}

	err := h.s.Room().DeleteRoom(r.Context(), id, claims.UserID)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		if err == helper.ErrPermissionDenied {
			helper.RespondError(w, http.StatusForbidden, helper.ErrPermissionDenied.Error())
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
	filter := &models.FilterRoomsRequest{
		HotelID:     r.URL.Query().Get("hotel_id"),
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

	rooms, total, err := h.s.Room().ListRooms(r.Context(), filter)
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

	rooms, err := h.s.Room().ListRoomsByHotel(r.Context(), hotelID)
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

// CheckAvailability handles checking room availability
func (h *Handler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	hotelID := r.URL.Query().Get("hotel_id")
	checkIn := r.URL.Query().Get("check_in")
	checkOut := r.URL.Query().Get("check_out")
	quantityStr := r.URL.Query().Get("quantity")

	if hotelID == "" || checkIn == "" || checkOut == "" || quantityStr == "" {
		helper.RespondError(w, http.StatusBadRequest, "missing required parameters: hotel_id, check_in, check_out, quantity")
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		helper.RespondError(w, http.StatusBadRequest, "quantity must be a number")
		return
	}

	rooms, err := h.s.Room().CheckAvailability(r.Context(), hotelID, checkIn, checkOut, quantity)
	if err != nil {
		h.l.Error("failed to check availability", "hotel_id", hotelID, "error", err)
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

// UpdateRoomQuantity handles updating room quantity
func (h *Handler) UpdateRoomQuantity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		helper.RespondError(w, http.StatusBadRequest, "room ID is required")
		return
	}

	claims := GetClaimsFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		helper.RespondError(w, http.StatusUnauthorized, helper.ErrUnauthorized.Error())
		return
	}

	var req models.UpdateRoomQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondError(w, http.StatusBadRequest, helper.ErrBadRequest.Error())
		return
	}

	updateReq := &models.UpdateRoomRequest{
		Quantity: req.Quantity,
	}

	room, err := h.s.Room().UpdateRoom(r.Context(), id, updateReq, claims.UserID)
	if err != nil {
		if err == helper.ErrRecordNotFound {
			helper.RespondError(w, http.StatusNotFound, helper.ErrRecordNotFound.Error())
			return
		}
		if err == helper.ErrPermissionDenied {
			helper.RespondError(w, http.StatusForbidden, helper.ErrPermissionDenied.Error())
			return
		}
		h.l.Error("failed to update room quantity", "id", id, "error", err)
		helper.RespondError(w, http.StatusInternalServerError, helper.ErrInternalServer.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
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
