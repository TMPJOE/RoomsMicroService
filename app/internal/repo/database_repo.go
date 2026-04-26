package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"hotel.com/app/internal/helper"
	"hotel.com/app/internal/models"
)

type roomRepo struct {
	db *pgxpool.Pool
}

func NewRoomRepo(db *pgxpool.Pool) RoomRepository {
	return &roomRepo{db: db}
}

func (r *roomRepo) DbPing() error {
	return r.db.Ping(context.Background())
}

// ─── Rooms CRUD ──────────────────────────────────────────────────────────────

func (r *roomRepo) CreateRoom(ctx context.Context, room *models.Room) error {
	query := `
INSERT INTO rooms (
    id, hotel_id, name, type, price, capacity, description,
    space_info, bed_distribution, quantity, amenity_count,
    recommendation_coef, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

	_, err := r.db.Exec(ctx, query,
		room.ID, room.HotelID, room.Name, room.Type, room.Price,
		room.Capacity, room.Description, room.SpaceInfo, room.BedDistribution,
		room.Quantity, room.AmenityCount, room.RecommendationCoef,
		room.CreatedAt, room.UpdatedAt,
	)
	if err != nil {
		return helper.MapError(err)
	}
	return nil
}

func (r *roomRepo) CreateRooms(ctx context.Context, rooms []*models.Room) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return helper.MapError(err)
	}
	defer tx.Rollback(ctx)

	query := `
INSERT INTO rooms (
    id, hotel_id, name, type, price, capacity, description,
    space_info, bed_distribution, quantity, amenity_count,
    recommendation_coef, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

	for _, room := range rooms {
		_, err := tx.Exec(ctx, query,
			room.ID, room.HotelID, room.Name, room.Type, room.Price,
			room.Capacity, room.Description, room.SpaceInfo, room.BedDistribution,
			room.Quantity, room.AmenityCount, room.RecommendationCoef,
			room.CreatedAt, room.UpdatedAt,
		)
		if err != nil {
			return helper.MapError(err)
		}
	}
	return tx.Commit(ctx)
}

func (r *roomRepo) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	query := `
SELECT id, hotel_id, name, type, price, capacity, description,
       space_info, bed_distribution, quantity, amenity_count,
       recommendation_coef, created_at, updated_at
FROM rooms
WHERE id = $1`

	room := &models.Room{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.HotelID, &room.Name, &room.Type, &room.Price,
		&room.Capacity, &room.Description, &room.SpaceInfo, &room.BedDistribution,
		&room.Quantity, &room.AmenityCount, &room.RecommendationCoef,
		&room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, helper.ErrRecordNotFound) || strings.Contains(err.Error(), "no rows") {
			return nil, helper.ErrRecordNotFound
		}
		return nil, helper.MapError(err)
	}

	// Load amenities from child tables
	amenityMap, err := r.GetHighlightedAmenitiesByRooms(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	categoryMap, err := r.GetAmenityCategoriesByRooms(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	room.HighlightedAmenities = amenityMap[id]
	room.AmenityCategories = categoryMap[id]

	return room, nil
}

func (r *roomRepo) UpdateRoom(ctx context.Context, room *models.Room) error {
	query := `
UPDATE rooms SET
    name = $3, type = $4, price = $5, capacity = $6,
    description = $7, space_info = $8, bed_distribution = $9,
    quantity = $10, amenity_count = $11, recommendation_coef = $12,
    updated_at = $13
WHERE id = $1 AND hotel_id = $2`

	result, err := r.db.Exec(ctx, query,
		room.ID, room.HotelID, room.Name, room.Type, room.Price,
		room.Capacity, room.Description, room.SpaceInfo, room.BedDistribution,
		room.Quantity, room.AmenityCount, room.RecommendationCoef, time.Now(),
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
	// Child rows are removed by ON DELETE CASCADE on the amenity tables.
	result, err := r.db.Exec(ctx, `DELETE FROM rooms WHERE id = $1`, id)
	if err != nil {
		return helper.MapError(err)
	}
	if result.RowsAffected() == 0 {
		return helper.ErrRecordNotFound
	}
	return nil
}

// ─── Query operations ─────────────────────────────────────────────────────────

func (r *roomRepo) ListRooms(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	return r.GetRoomsByFilters(ctx, filter)
}

func (r *roomRepo) ListRoomsByHotel(ctx context.Context, hotelID string) ([]models.Room, error) {
	query := `
SELECT id, hotel_id, name, type, price, capacity, description,
       space_info, bed_distribution, quantity, amenity_count,
       recommendation_coef, created_at, updated_at
FROM rooms
WHERE hotel_id = $1
ORDER BY id ASC`

	rows, err := r.db.Query(ctx, query, hotelID)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	rooms, err := r.scanRows(rows)
	if err != nil {
		return nil, err
	}
	if err := r.hydrate(ctx, rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepo) GetRoomsByFilters(ctx context.Context, filter *models.FilterRoomsRequest) ([]models.Room, int, error) {
	query := `
SELECT id, hotel_id, name, type, price, capacity, description,
       space_info, bed_distribution, quantity, amenity_count,
       recommendation_coef, created_at, updated_at
FROM rooms
WHERE 1=1`

	args := []interface{}{}
	idx := 1

	if filter.HotelID != "" {
		query += fmt.Sprintf(" AND hotel_id = $%d", idx)
		args = append(args, filter.HotelID)
		idx++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, filter.Type)
		idx++
	}
	if filter.MinCapacity > 0 {
		query += fmt.Sprintf(" AND capacity >= $%d", idx)
		args = append(args, filter.MinCapacity)
		idx++
	}
	if filter.MaxCapacity > 0 {
		query += fmt.Sprintf(" AND capacity <= $%d", idx)
		args = append(args, filter.MaxCapacity)
		idx++
	}
	if filter.MinPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", idx)
		args = append(args, filter.MinPrice)
		idx++
	}
	if filter.MaxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", idx)
		args = append(args, filter.MaxPrice)
		idx++
	}
	if filter.MinCoef > 0 {
		query += fmt.Sprintf(" AND recommendation_coef >= $%d", idx)
		args = append(args, filter.MinCoef)
		idx++
	}
	if filter.MaxCoef > 0 {
		query += fmt.Sprintf(" AND recommendation_coef <= $%d", idx)
		args = append(args, filter.MaxCoef)
		idx++
	}

	query += " ORDER BY recommendation_coef DESC, price ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filter.Limit)
		idx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", idx)
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
	if err := r.hydrate(ctx, rooms); err != nil {
		return nil, 0, err
	}
	return rooms, len(rooms), nil
}

// ─── Availability ─────────────────────────────────────────────────────────────

func (r *roomRepo) CheckAvailability(ctx context.Context, hotelID string, checkIn, checkOut string, quantity int) ([]models.Room, error) {
	query := `
SELECT id, hotel_id, name, type, price, capacity, description,
       space_info, bed_distribution, quantity, amenity_count,
       recommendation_coef, created_at, updated_at
FROM rooms
WHERE hotel_id = $1 AND quantity >= $2
ORDER BY recommendation_coef DESC, price ASC`

	rows, err := r.db.Query(ctx, query, hotelID, quantity)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	rooms, err := r.scanRows(rows)
	if err != nil {
		return nil, err
	}
	if err := r.hydrate(ctx, rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepo) CheckAvailabilityByType(ctx context.Context, hotelID, roomType, name string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM rooms WHERE hotel_id = $1 AND type = $2 AND name = $3`,
		hotelID, roomType, name,
	).Scan(&count)
	if err != nil {
		return 0, helper.MapError(err)
	}
	return count, nil
}

// ─── Quantity ─────────────────────────────────────────────────────────────────

func (r *roomRepo) UpdateRoomQuantity(ctx context.Context, hotelID, roomType, name string, quantity int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE rooms SET quantity = $4, updated_at = $5
         WHERE hotel_id = $1 AND type = $2 AND name = $3`,
		hotelID, roomType, name, quantity, time.Now(),
	)
	if err != nil {
		return helper.MapError(err)
	}
	return nil
}

// ─── Highlighted amenities ────────────────────────────────────────────────────

// UpsertHighlightedAmenities replaces all highlighted amenities for a room
// within a single transaction (delete existing, then insert new).
func (r *roomRepo) UpsertHighlightedAmenities(ctx context.Context, roomID string, amenities []models.HighlightedAmenity) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return helper.MapError(err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM highlighted_amenities WHERE room_id = $1`, roomID); err != nil {
		return helper.MapError(err)
	}

	for _, a := range amenities {
		_, err := tx.Exec(ctx,
			`INSERT INTO highlighted_amenities (id, room_id, icon, text, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, roomID, string(a.Icon), a.Text, a.CreatedAt, a.UpdatedAt,
		)
		if err != nil {
			return helper.MapError(err)
		}
	}
	return tx.Commit(ctx)
}

// GetHighlightedAmenitiesByRooms batch-loads highlighted amenities for a set of
// room IDs, returning a map[roomID][]HighlightedAmenity. Avoids N+1 queries.
func (r *roomRepo) GetHighlightedAmenitiesByRooms(ctx context.Context, roomIDs []string) (map[string][]models.HighlightedAmenity, error) {
	result := make(map[string][]models.HighlightedAmenity, len(roomIDs))
	if len(roomIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(roomIDs))
	args := make([]interface{}, len(roomIDs))
	for i, id := range roomIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT id, room_id, icon, text, created_at, updated_at
FROM highlighted_amenities
WHERE room_id IN (%s)
ORDER BY room_id, id ASC`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var a models.HighlightedAmenity
		var icon string
		if err := rows.Scan(&a.ID, &a.RoomID, &icon, &a.Text, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, helper.MapError(err)
		}
		a.Icon = models.AmenityIcon(icon)
		result[a.RoomID] = append(result[a.RoomID], a)
	}
	return result, nil
}

// ─── Amenity categories ───────────────────────────────────────────────────────

// UpsertAmenityCategories replaces all amenity categories for a room within a
// single transaction (delete existing, then insert new).
func (r *roomRepo) UpsertAmenityCategories(ctx context.Context, roomID string, categories []models.AmenityCategory) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return helper.MapError(err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM amenity_categories WHERE room_id = $1`, roomID); err != nil {
		return helper.MapError(err)
	}

	for _, c := range categories {
		_, err := tx.Exec(ctx,
			`INSERT INTO amenity_categories (id, room_id, name, description, tier, amenity_count, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			c.ID, roomID, c.Name, c.Description, string(c.Tier), c.AmenityCount, c.CreatedAt, c.UpdatedAt,
		)
		if err != nil {
			return helper.MapError(err)
		}
	}
	return tx.Commit(ctx)
}

// GetAmenityCategoriesByRooms batch-loads amenity categories for a set of room
// IDs, returning a map[roomID][]AmenityCategory. Avoids N+1 queries.
func (r *roomRepo) GetAmenityCategoriesByRooms(ctx context.Context, roomIDs []string) (map[string][]models.AmenityCategory, error) {
	result := make(map[string][]models.AmenityCategory, len(roomIDs))
	if len(roomIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(roomIDs))
	args := make([]interface{}, len(roomIDs))
	for i, id := range roomIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT id, room_id, name, description, tier, amenity_count, created_at, updated_at
FROM amenity_categories
WHERE room_id IN (%s)
ORDER BY room_id, id ASC`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, helper.MapError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var c models.AmenityCategory
		var tier string
		if err := rows.Scan(&c.ID, &c.RoomID, &c.Name, &c.Description, &tier, &c.AmenityCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, helper.MapError(err)
		}
		c.Tier = models.CategoryTier(tier)
		result[c.RoomID] = append(result[c.RoomID], c)
	}
	return result, nil
}

// ─── Private helpers ──────────────────────────────────────────────────────────

// scanRows scans query results into a slice of Room (room table columns only).
func (r *roomRepo) scanRows(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close()
}) ([]models.Room, error) {
	var rooms []models.Room
	for rows.Next() {
		var room models.Room
		err := rows.Scan(
			&room.ID, &room.HotelID, &room.Name, &room.Type, &room.Price,
			&room.Capacity, &room.Description, &room.SpaceInfo, &room.BedDistribution,
			&room.Quantity, &room.AmenityCount, &room.RecommendationCoef,
			&room.CreatedAt, &room.UpdatedAt,
		)
		if err != nil {
			return nil, helper.MapError(err)
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// hydrate batch-loads highlighted amenities and categories for a slice of rooms
// and attaches them, avoiding N+1 queries.
func (r *roomRepo) hydrate(ctx context.Context, rooms []models.Room) error {
	if len(rooms) == 0 {
		return nil
	}
	ids := make([]string, len(rooms))
	for i, room := range rooms {
		ids[i] = room.ID
	}

	amenityMap, err := r.GetHighlightedAmenitiesByRooms(ctx, ids)
	if err != nil {
		return err
	}
	categoryMap, err := r.GetAmenityCategoriesByRooms(ctx, ids)
	if err != nil {
		return err
	}
	for i := range rooms {
		rooms[i].HighlightedAmenities = amenityMap[rooms[i].ID]
		rooms[i].AmenityCategories = categoryMap[rooms[i].ID]
	}
	return nil
}
