package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
)

type databaseRepo struct {
	db *pgxpool.Pool
}

func NewDatabaseRepo(conn *pgxpool.Pool) ServiceRepository {
	return &databaseRepo{
		db: conn,
	}
}

func (dbr *databaseRepo) DbPing() error {
	err := dbr.db.Ping(context.Background())
	return err
}

type roomRepo struct {
	db *pgxpool.Pool
}

func NewRoomRepo(db *pgxpool.Pool) RoomRepository {
	return &roomRepo{
		db: db,
	}
}

func (r *roomRepo) CreateRoom(ctx context.Context, room *models.Room) error {
	query := `
	INSERT INTO rooms (
	id, hotel_id, name, type, price, capacity, description,
	space_info, bed_distribution, quantity, highlighted_amenities,
	amenity_categories, amenity_count, recommendation_coef,
	created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	var highlightedAmenitiesJSON []byte
	if len(room.HighlightedAmenities) > 0 {
		var err error
		highlightedAmenitiesJSON, err = json.Marshal(room.HighlightedAmenities)
		if err != nil {
			return helper.ErrInternalServer
		}
	}

	_, err := r.db.Exec(ctx, query,
		room.ID,
		room.HotelID,
		room.Name,
		room.Type,
		room.Price,
		room.Capacity,
		room.Description,
		room.SpaceInfo,
		room.BedDistribution,
		room.Quantity,
		highlightedAmenitiesJSON,
		room.AmenityCategories,
		room.AmenityCount,
		room.RecommendationCoef,
		room.CreatedAt,
		room.UpdatedAt,
	)

	if err != nil {
		return helper.MapError(err)
	}

	return nil
}

func (r *roomRepo) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	query := `
	SELECT id, hotel_id, name, type, price, capacity, description,
	space_info, bed_distribution, quantity, highlighted_amenities,
	amenity_categories, amenity_count, recommendation_coef,
	created_at, updated_at
	FROM rooms
	WHERE id = $1
	`

	room := &models.Room{}
	var highlightedAmenitiesJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID,
		&room.HotelID,
		&room.Name,
		&room.Type,
		&room.Price,
		&room.Capacity,
		&room.Description,
		&room.SpaceInfo,
		&room.BedDistribution,
		&room.Quantity,
		&highlightedAmenitiesJSON,
		&room.AmenityCategories,
		&room.AmenityCount,
		&room.RecommendationCoef,
		&room.CreatedAt,
		&room.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, helper.ErrRecordNotFound) || strings.Contains(err.Error(), "no rows") {
			return nil, helper.ErrRecordNotFound
		}
		return nil, helper.MapError(err)
	}

	if len(highlightedAmenitiesJSON) > 0 {
		room.HighlightedAmenities = highlightedAmenitiesJSON
	}

	return room, nil
}

func (r *roomRepo) UpdateRoom(ctx context.Context, room *models.Room) error {
	query := `
	UPDATE rooms SET
	name = $3, type = $4, price = $5, capacity = $6,
	description = $7, space_info = $8, bed_distribution = $9,
	quantity = $10, highlighted_amenities = $11,
	amenity_categories = $12, amenity_count = $13,
	recommendation_coef = $14, updated_at = $15
	WHERE id = $1 AND hotel_id = $2
	`

	var highlightedAmenitiesJSON []byte
	if len(room.HighlightedAmenities) > 0 {
		var err error
		highlightedAmenitiesJSON, err = json.Marshal(room.HighlightedAmenities)
		if err != nil {
			return helper.ErrInternalServer
		}
	}

	result, err := r.db.Exec(ctx, query,
		room.ID,
		room.HotelID,
		room.Name,
		room.Type,
		room.Price,
		room.Capacity,
		room.Description,
		room.SpaceInfo,
		room.BedDistribution,
		room.Quantity,
		highlightedAmenitiesJSON,
		room.AmenityCategories,
		room.AmenityCount,
		room.RecommendationCoef,
		time.Now(),
	)

	if err != nil {
		return helper.MapError(err)
	}

	if result.RowsAffected() == 0 {
		return helper.ErrRecordNotFound
	}

	return nil
}

func (r *roomRepo) DeleteRoom(ctx context.Context, id string) error {
	query := `DELETE FROM rooms WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return helper.MapError(err)
	}

	if result.RowsAffected() == 0 {
		return helper.ErrRecordNotFound
	}

	return nil
}

func (r *roomRepo) ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	return r.GetRoomsByFilters(ctx, filter)
}

func (r *roomRepo) ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error) {
	query := `
	SELECT id, hotel_id, name, type, price, capacity, description,
	space_info, bed_distribution, quantity, highlighted_amenities,
	amenity_categories, amenity_count, recommendation_coef,
	created_at, updated_at
	FROM rooms
	WHERE hotel_id = $1
	ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, hotelID)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	rooms, err := r.scanRows(rows)
	if err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *roomRepo) GetRoomsByFilters(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	query := `
	SELECT id, hotel_id, name, type, price, capacity, description,
	space_info, bed_distribution, quantity, highlighted_amenities,
	amenity_categories, amenity_count, recommendation_coef,
	created_at, updated_at
	FROM rooms
	WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if filter.HotelID != "" {
		query += ` AND hotel_id = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.HotelID)
		argIndex++
	}

	if filter.Type != "" {
		query += ` AND type = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.Type)
		argIndex++
	}

	if filter.MinCapacity > 0 {
		query += ` AND capacity >= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MinCapacity)
		argIndex++
	}

	if filter.MaxCapacity > 0 {
		query += ` AND capacity <= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MaxCapacity)
		argIndex++
	}

	if filter.MinPrice > 0 {
		query += ` AND price >= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice > 0 {
		query += ` AND price <= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MaxPrice)
		argIndex++
	}

	if filter.MinCoef > 0 {
		query += ` AND recommendation_coef >= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MinCoef)
		argIndex++
	}

	if filter.MaxCoef > 0 {
		query += ` AND recommendation_coef <= $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.MaxCoef)
		argIndex++
	}

	query += ` ORDER BY recommendation_coef DESC, price ASC`

	if filter.Limit > 0 {
		query += ` LIMIT $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += ` OFFSET $` + fmt.Sprintf("%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, helper.MapError(err)
	}
	defer rows.Close()

	rooms, err := r.scanRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return rooms, len(rooms), nil
}

func (r *roomRepo) CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error) {
	query := `
	SELECT id, hotel_id, name, type, price, capacity, description,
	space_info, bed_distribution, quantity, highlighted_amenities,
	amenity_categories, amenity_count, recommendation_coef,
	created_at, updated_at
	FROM rooms
	WHERE hotel_id = $1 AND quantity >= $2
	ORDER BY recommendation_coef DESC, price ASC
	`

	rows, err := r.db.Query(ctx, query, hotelID, quantity)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	rooms, err := r.scanRows(rows)
	if err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *roomRepo) UpdateAmenities(ctx context.Context, id string, amenities []models.HighlightedAmenity, amenityCategories string, amenityCount int, recommendationCoef float64) error {
	query := `
	UPDATE rooms SET
	highlighted_amenities = $2,
	amenity_categories = $3,
	amenity_count = $4,
	recommendation_coef = $5,
	updated_at = $6
	WHERE id = $1
	`

	amenitiesJSON, err := json.Marshal(amenities)
	if err != nil {
		return helper.ErrInternalServer
	}

	result, err := r.db.Exec(ctx, query, id, amenitiesJSON, amenityCategories, amenityCount, recommendationCoef, time.Now())
	if err != nil {
		return helper.MapError(err)
	}

	if result.RowsAffected() == 0 {
		return helper.ErrRecordNotFound
	}

	return nil
}

func (r *roomRepo) scanRows(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close()
}) ([]models.Room, error) {
	var rooms []models.Room

	for rows.Next() {
		room := models.Room{}
		var highlightedAmenitiesJSON []byte

		err := rows.Scan(
			&room.ID,
			&room.HotelID,
			&room.Name,
			&room.Type,
			&room.Price,
			&room.Capacity,
			&room.Description,
			&room.SpaceInfo,
			&room.BedDistribution,
			&room.Quantity,
			&highlightedAmenitiesJSON,
			&room.AmenityCategories,
			&room.AmenityCount,
			&room.RecommendationCoef,
			&room.CreatedAt,
			&room.UpdatedAt,
		)

		if err != nil {
			return nil, helper.MapError(err)
		}

		if len(highlightedAmenitiesJSON) > 0 {
			room.HighlightedAmenities = highlightedAmenitiesJSON
		}

		rooms = append(rooms, room)
	}

	return rooms, nil
}
