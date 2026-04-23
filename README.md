# Rooms MicroService

A Go microservice for managing hotel room inventory, built with a clean architecture pattern. Features PostgreSQL integration, structured logging, JWT authentication, room CRUD operations, availability checking, and integration with Media Service for image management.

## Architecture

The project follows a layered architecture:

```
cmd/api/main.go → Entry point, wires dependencies
internal/handler → HTTP handlers, routing, and middleware
internal/service → Business logic layer (room service, recommendation coefficient calculation)
internal/repo → Data access layer (room repository, database operations)
internal/database → Database connection management
internal/logging → Structured logging setup
internal/models → Domain models and DTOs
internal/helper → Utility functions and error handling
internal/config → YAML configuration loader
internal/client → External service clients (Media Service)
```

## Tech Stack

- **Router**: [go-chi/chi/v5](https://github.com/go-chi/chi)
- **Logging**: [go-chi/httplog/v3](https://github.com/go-chi/httplog) + `log/slog`
- **Database**: [jackc/pgx/v5](https://github.com/jackc/pgx) (PostgreSQL connection pool)
- **JWT Authentication**: [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt)
- **Validation**: [go-playground/validator/v10](https://github.com/go-playground/validator)
- **UUID Generation**: [google/uuid](https://github.com/google/uuid)

## Features

### Room Management
- **CRUD Operations**: Create, read, update, and delete rooms
- **Room Types**: Single, Double, Double/Double, Suite
- **Filtering**: Filter by hotel_id, type, capacity, price, recommendation coefficient
- **Availability Checking**: Check room availability for date ranges
- **Quantity Tracking**: Track number of rooms of each type

### Recommendation System
- **Highlighted Amenities**: Individual amenity items with icons and weights
- **Amenity Categories**: Category-based amenity grouping
- **Recommendation Coefficient**: Calculated as `(highlighted_sum) × (category_count) × (amenity_count)`
- **Amenity Weights**: Customizable multipliers (wifi=1.5, ac=1.2, tv=1.1, etc.)

### Security
- **JWT Authentication**: RSA-based token validation with configurable issuer and expiration
- **Security Headers**: X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, HSTS, CSP
- **Request ID**: Unique request tracking for debugging and logging
- **Hotel Ownership Validation**: Admins can only manage rooms for their hotel

### Resilience
- **Rate Limiting**: Token bucket algorithm with configurable requests/second and burst
- **Circuit Breaker**: Automatic failure detection with half-open state for recovery
- **Graceful Shutdown**: 30-second timeout for in-flight requests

### Configuration
- **YAML Config**: All settings loaded from `config.yaml` with environment variable expansion
- **Media Service Integration**: Configurable media service URL for image uploads
- **No hardcoded values**: Server port, timeouts, rate limits all configurable

## Prerequisites

- Go 1.25.7+
- PostgreSQL database
- Docker & Docker Compose (optional, for local development)
- RSA key pair for JWT signing (`public.pem`, `private.pem`)

## Getting Started

### 1. Generate JWT Keys

```bash
# Generate private key
openssl genrsa -out private.pem 2048

# Generate public key
openssl rsa -in private.pem -pubout -out public.pem
```

### 2. Set Environment Variables

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"
export MEDIA_SERVICE_URL="http://media-service:8080"
```

### 3. Configure the Service

Edit `config.yaml` to customize:
- Server host/port and timeouts
- Logging level and format
- Rate limiting parameters
- Circuit breaker settings
- Health check paths
- Media service URL

### 4. Run the Service

```bash
go run app/cmd/api/main.go
```

The server starts on `localhost:8080` (or configured port).

### 5. Test the Health Endpoint

```bash
curl http://localhost:8080/health
```

Response:
```json
{"status": "ok"}
```

## Docker

### Build the Image

```bash
docker build -t rooms-microservice .
```

### Run with Docker

```bash
docker run -p 8080:8080 \
-e DATABASE_URL="postgres://user:password@host:5432/dbname?sslmode=disable" \
-e MEDIA_SERVICE_URL="http://media-service:8080" \
-v /path/to/keys:/app/keys \
rooms-microservice
```

### Docker Compose

Use `docker-compose.yml` to spin up dependencies (e.g., PostgreSQL):

```bash
docker-compose up -d
```

## Project Structure

| Path | Description |
|------|-------------|
| `app/cmd/api/main.go` | Application entry point. Wires together database, repositories, services, and handlers. |
| `app/internal/config/` | YAML configuration loader with environment variable expansion. |
| `app/internal/database/` | Database connection pool initialization using `pgx`. |
| `app/internal/handler/` | HTTP handlers, request routing (`chi`), and middleware (security, JWT, rate limiting). |
| `app/internal/service/` | Business logic layer. Room service with CRUD operations and recommendation coefficient calculation. |
| `app/internal/repo/` | Data access layer. Room repository with PostgreSQL queries. |
| `app/internal/client/` | External service clients (Media Service for image uploads). |
| `app/internal/logging/` | Structured JSON logger configuration using `slog` and `httplog`. |
| `app/internal/models/` | Domain models and data structures shared across layers. |
| `app/internal/helper/` | Utility/helper functions including comprehensive error definitions. |
| `app/sql/` | SQL migration files and queries. |
| `config.yaml` | Service configuration file. |
| `Dockerfile` | Multi-stage Docker build with healthcheck. |

## API Endpoints

### Public Routes (No Authentication)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check endpoint. Returns service health status. |
| `GET` | `/ready` | Readiness check. Verifies database connectivity. |
| `GET` | `/rooms` | List rooms with filters (hotel_id, type, capacity, price, coef). |
| `GET` | `/rooms/{id}` | Get room by ID. |
| `GET` | `/hotels/{hotel_id}/rooms` | List all rooms for a hotel. |
| `GET` | `/rooms/available` | Check available rooms (params: hotel_id, check_in, check_out, quantity). |

### Protected Routes (JWT Required - Admin)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/rooms` | Create a new room (admin only, validates hotel_id from JWT). |
| `PUT` | `/rooms/{id}` | Update an existing room (admin only, ownership validation). |
| `DELETE` | `/rooms/{id}` | Delete a room (admin only, ownership validation). |
| `PATCH` | `/rooms/{id}/quantity` | Update room quantity (admin only). |

### Filter Parameters

```
GET /rooms?hotel_id=uuid&type=Single,Double&min_capacity=2&max_price=500&min_coef=0.5&limit=20&offset=0
```

## Configuration Reference

### config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s

logging:
  level: "info"
  format: "json"

rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200

circuit_breaker:
  enabled: true
  max_failures: 5
  timeout: 30s

health:
  path: "/health"
  ready_path: "/ready"

media_service_url: "http://media-service:8080"
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string (required) |
| `MEDIA_SERVICE_URL` | Media service URL for image uploads (optional, defaults to `http://media-service:8080`) |

## Database Schema

### Rooms Table

```sql
rooms:
- id (UUID, PK)
- hotel_id (UUID, FK to hotels)
- name (VARCHAR) - Room name/number
- type (VARCHAR) - Single, Double, Double/Double, Suite
- price (DECIMAL) - Price per night
- capacity (INT) - Guest capacity
- description (TEXT)
- space_info (VARCHAR) - e.g., "1 habitación, 2 personas • 350 pies"
- bed_distribution (VARCHAR) - e.g., "1 cama King"
- quantity (INT) - Number of rooms of this type
- highlighted_amenities (JSONB) - Array of {icon, text} objects
- amenity_categories (TEXT) - Comma-separated list of selected categories
- amenity_count (INT) - Calculated count from description parsing
- recommendation_coef (DECIMAL) - Calculated coefficient for ranking
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
```

## Adding New Features

1. **Models**: Define structs in `app/internal/models/models.go`
2. **Repository**: Add data access methods to `app/internal/repo/repo.go` and implement in `app/internal/repo/room_repo.go`
3. **Service**: Add business logic methods to `app/internal/service/service.go` (update the `Service` interface)
4. **Handler**: Add HTTP handler functions to `app/internal/handler/handlers.go` or create new handler file
5. **Routing**: Register new routes in `app/internal/handler/routing.go`
6. **Configuration**: Add any new config options to `config.yaml` and `app/internal/config/config.go`

## Error Handling

The service uses a comprehensive error system defined in `app/internal/helper/util.go`:

- **General errors**: `ErrInternalServer`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, etc.
- **Database errors**: `ErrDBConnection`, `ErrDBQuery`, `ErrRecordNotFound`, `ErrDuplicateEntry`, etc.
- **Authentication errors**: `ErrInvalidCredentials`, `ErrInvalidToken`, `ErrTokenExpired`, etc.
- **Service errors**: `ErrServiceUnavailable`, `ErrCreateFailed`, `ErrProcessingFailed`, etc.

Use `helper.MapError()` in the repository layer to convert raw database errors to application sentinel errors.

## Recommendation Coefficient Calculation

The recommendation coefficient is calculated using the formula:

```
coef = (Σ highlighted_amenity_multipliers) × (amenity_category_count) × (total_amenity_count)
```

Where:
- **Highlighted amenities**: Each amenity has a weight (wifi=1.5, ac=1.2, tv=1.1, etc.)
- **Amenity categories**: Count of selected category toggles
- **Total amenity count**: Parsed from description (colon-separated items)

This coefficient is used to rank rooms in search results, with higher coefficients indicating more desirable rooms.
