# 🛏️ Rooms Microservice

> Room inventory, amenity management, and availability service for the Hotel Reservation Platform.

## Overview

The Rooms Microservice manages **room types** within hotels, including their **highlighted amenities**, **amenity categories** (with tier-based recommendation coefficients), **availability checking**, and **quantity tracking**. It exposes both public read endpoints and admin-protected write endpoints. The recommendation coefficient system helps the frontend surface the best rooms to guests.

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Router | [go-chi/chi](https://github.com/go-chi/chi) v5 |
| Database | PostgreSQL 16 |
| DB Driver | [pgx](https://github.com/jackc/pgx) v5 |
| Auth | JWT verification (RSA-256 public key) |
| UUID | Google UUID v7 (time-sortable) |
| Container | Docker (multi-stage Alpine build) |

## Architecture

```
app/
├── cmd/api/          # Application entrypoint
│   └── main.go
├── internal/
│   ├── client/       # Media Service HTTP client
│   ├── config/       # YAML config loader
│   ├── database/     # PostgreSQL connection pool
│   ├── handler/      # HTTP handlers, routing, JWT middleware
│   ├── helper/       # Validators, error types, response helpers
│   ├── logging/      # Structured slog logger
│   ├── models/       # Domain entities (Room, Amenities, DTOs, Enums)
│   ├── repo/         # Repository interface + PostgreSQL implementation
│   └── service/      # Business logic (CRUD + recommendation coef)
├── sql/
│   └── migrations/   # SQL migrations (rooms, highlighted_amenities, amenity_categories)
├── config.yaml
├── Dockerfile
└── go.mod
```

## API Endpoints

### Public Routes (No Authentication)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness probe |
| `GET` | `/ready` | Readiness probe |
| `GET` | `/rooms` | List all rooms (with filters) |
| `GET` | `/rooms/{id}` | Get room by ID |
| `GET` | `/rooms/list/{hotel_id}` | List rooms for a specific hotel |
| `GET` | `/rooms/available/{hotel_id}` | Check room availability |

### Protected Routes (JWT Required)

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/hotels/{hotel_id}/rooms` | Admin | Create a new room type |
| `PUT` | `/hotels/{hotel_id}/rooms/{id}` | Admin | Update room details |
| `DELETE` | `/hotels/{hotel_id}/rooms/{id}` | Admin | Delete a room type |
| `PATCH` | `/hotels/{hotel_id}/rooms/{type}/{name}/quantity` | Admin | Update room quantity |

## Data Model

### `rooms` Table

| Column | Type | Description |
|---|---|---|
| `id` | UUID v7 | Primary key |
| `hotel_id` | UUID | FK → Hotel service |
| `name` | VARCHAR | Room name (e.g., "Deluxe King") |
| `type` | VARCHAR | Room type (`Single`, `Double`, `Double/Double`, `Suite`) |
| `price` | FLOAT | Price per night |
| `capacity` | INT | Maximum guest capacity |
| `description` | TEXT | Room description |
| `space_info` | TEXT | Space/size information |
| `bed_distribution` | VARCHAR | Bed layout description |
| `quantity` | INT | Number of rooms of this type |
| `amenity_count` | INT | Total amenity count |
| `recommendation_coef` | FLOAT | Auto-calculated recommendation score |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

### `highlighted_amenities` Table

| Column | Type | Description |
|---|---|---|
| `id` | UUID v7 | Primary key |
| `room_id` | UUID | FK → `rooms.id` |
| `icon` | VARCHAR | Icon identifier (enum: `wifi`, `tv`, `ac`, etc.) |
| `text` | VARCHAR | Display text |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

### `amenity_categories` Table

| Column | Type | Description |
|---|---|---|
| `id` | UUID v7 | Primary key |
| `room_id` | UUID | FK → `rooms.id` |
| `name` | VARCHAR | Category name (e.g., "Bathroom") |
| `description` | TEXT | Comma-separated amenity list |
| `tier` | VARCHAR | `basic`, `essential`, `comfort`, `luxury` |
| `amenity_count` | INT | Number of amenities in this category |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

## Flow Diagram

```mermaid
flowchart TD
    A["Client Request"] --> B{"Route Type?"}
    B -->|Public| C{"Endpoint?"}
    B -->|Protected| D["JWT Middleware"]
    
    C -->|GET /rooms| E["Parse Filter Params"]
    E --> E1["ListRooms from DB"]
    E1 --> E2["Batch-load Amenities"]
    E2 --> E3["Batch-load Categories"]
    E3 --> E4["Return Rooms JSON"]
    
    C -->|GET /rooms/id| F["GetRoomByID"]
    F --> F1["Load Highlighted Amenities"]
    F1 --> F2["Load Amenity Categories"]
    F2 --> F3["Return Room JSON"]
    
    C -->|GET /rooms/list/hotel_id| G["ListRoomsByHotel"]
    G --> G1["Batch-load all child data"]
    G1 --> G2["Return Rooms JSON"]
    
    C -->|GET /rooms/available/hotel_id| H["CheckAvailability"]
    H --> H1["Query available rooms"]
    H1 --> H2["Return AvailabilityResponse"]
    
    D --> D1{"Token Valid?"}
    D1 -->|No| D2["401 Unauthorized"]
    D1 -->|Yes| D3["Extract Claims"]
    
    D3 --> I{"Endpoint?"}
    I -->|POST /hotels/hotel_id/rooms| J["Decode CreateRoomRequest"]
    J --> J1["Validate Input"]
    J1 --> J2["Generate UUID v7"]
    J2 --> J3["Calculate Recommendation Coef"]
    J3 --> J4["Insert Room"]
    J4 --> J5["Insert Highlighted Amenities"]
    J5 --> J6["Insert Amenity Categories"]
    J6 --> J7["201 Created"]
    
    I -->|PATCH .../quantity| K["Decode UpdateQuantityRequest"]
    K --> K1["Update Room Quantity"]
    K1 --> K2["200 OK"]
```

## Use Case Diagram

```mermaid
graph LR
    subgraph Actors
        Guest["🧑 Guest"]
        User["👤 Authenticated User"]
        Admin["🔑 Admin"]
        BFF["🔄 BFF Service"]
    end
    
    subgraph "Rooms Microservice"
        UC1["Browse Room Types"]
        UC2["View Room Details"]
        UC3["Check Availability"]
        UC4["Create Room Type"]
        UC5["Update Room"]
        UC6["Delete Room"]
        UC7["Update Room Quantity"]
        UC8["List Rooms by Hotel"]
    end
    
    Guest --> UC1
    Guest --> UC2
    Guest --> UC3
    Guest --> UC8
    User --> UC1
    User --> UC2
    User --> UC3
    Admin --> UC4
    Admin --> UC5
    Admin --> UC6
    Admin --> UC7
    BFF --> UC8
    BFF --> UC2
    BFF --> UC4
```

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> NonExistent
    NonExistent --> Active : Admin creates room
    Active --> Active : Update details
    Active --> Active : Update quantity
    Active --> [*] : Admin deletes room
    
    state Active {
        [*] --> Available
        Available --> Occupied : Booking confirmed
        Occupied --> Available : Booking completed/cancelled
        Available --> LowStock : quantity decremented
        LowStock --> Available : quantity incremented
        LowStock --> OutOfStock : quantity = 0
        OutOfStock --> LowStock : quantity incremented
    }
```

## Package Diagram

```mermaid
graph TB
    subgraph "cmd/api"
        Main["main.go"]
    end
    
    subgraph "internal"
        subgraph "handler"
            Handlers["handlers.go"]
            Routing["routing.go"]
            MW["middleware.go (JWT)"]
        end
        
        subgraph "service"
            SVC["service.go"]
        end
        
        subgraph "repo"
            RepoIF["repo.go (RoomRepository)"]
            DBRepo["database_repo.go"]
        end
        
        subgraph "models"
            Models["models.go"]
            Amenities["amenities.go (enums)"]
        end
        
        subgraph "client"
            MediaClient["media_client.go"]
        end
        
        subgraph "helper"
            Helper["validators, errors"]
        end
        
        subgraph "config"
            Config["config.go"]
        end
        
        subgraph "database"
            DB["connection.go"]
        end
    end
    
    Main --> Config
    Main --> DB
    Main --> SVC
    Main --> Handlers
    
    Handlers --> SVC
    Handlers --> Helper
    Handlers --> Models
    Routing --> Handlers
    Routing --> MW
    
    SVC --> RepoIF
    SVC --> Models
    SVC --> Amenities
    SVC --> MediaClient
    
    DBRepo -.->|implements| RepoIF
    DBRepo --> DB
    DBRepo --> Models
```

## Recommendation Coefficient

The recommendation coefficient is auto-calculated using the amenity tier system:

| Tier | Multiplier | Description |
|---|---|---|
| `basic` | 1.0× | Standard amenities |
| `essential` | 1.5× | Important amenities |
| `comfort` | 2.0× | Comfort-enhancing amenities |
| `luxury` | 3.0× | Premium amenities |

**Formula**: `coef = Σ(category.amenity_count × tier_multiplier)`

## Configuration

### Environment Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |

### Volume Mounts (Docker)

| Host Path | Container Path | Description |
|---|---|---|
| `./keys/public.pem` | `/app/keys/public.pem` | JWT verification key |

## Port Mapping

| Context | Port |
|---|---|
| Internal (container) | `8080` |
| External (host) | `8085` |
| Database (host) | `5436` → `5432` |
