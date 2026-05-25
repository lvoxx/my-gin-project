# Employee Management System

A production-oriented REST API built with **Go** and the **Gin** framework, following clean
architecture principles with strict separation of concerns, dependency injection, and
centralized error handling.

---

## Table of Contents

- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Design Decisions](#design-decisions)
- [Deployment](#deployment)

---

## Architecture

```
HTTP Request
     │
     ▼
┌─────────────────────────────────────────────────────┐
│                    Gin Engine                        │
│  Recovery → Exception → RequestLogger → CORS        │  ← Middleware
├─────────────────────────────────────────────────────┤
│              Handler (DTO parse & validate)          │  ← HTTP layer
├─────────────────────────────────────────────────────┤
│           Service (business logic, caching)          │  ← Application layer
├─────────────────────────────────────────────────────┤
│         Repository (SQL, DBTX abstraction)           │  ← Data access layer
├────────────────────┬────────────────────────────────┤
│    PostgreSQL       │         Redis                  │  ← Infrastructure
└────────────────────┴────────────────────────────────┘
```

### Key Patterns

| Pattern | Implementation |
|---------|---------------|
| Clean Architecture | Handler → Service → Repository; domain entities have zero framework dependencies |
| Dependency Injection | Manual constructor injection wired in `cmd/server/main.go` |
| Repository Pattern | `EmployeeRepository` / `DepartmentRepository` interfaces with PostgreSQL implementations |
| Centralized Error Handling | `middleware.Exception` reads `c.Errors`; handlers never write error responses directly |
| Transaction Management | Context-scoped `database.WithTx` propagates `pgx.Tx` transparently to repositories |
| Read-Through Cache | Redis caches individual records and list results; write operations invalidate by key pattern |
| Full-Text Search | PostgreSQL `tsvector` column updated by trigger; queried with `plainto_tsquery` |

---

## Tech Stack

| Concern | Library |
|---------|---------|
| HTTP framework | `github.com/gin-gonic/gin` |
| PostgreSQL driver | `github.com/jackc/pgx/v5` (pgxpool) |
| Redis client | `github.com/redis/go-redis/v9` |
| JWT | `github.com/golang-jwt/jwt/v5` |
| Password hashing | `golang.org/x/crypto/bcrypt` |
| Avatar storage | `github.com/cloudinary/cloudinary-go/v2` |
| Structured logging | `go.uber.org/zap` + `lumberjack` (file rotation) |
| UUID | `github.com/google/uuid` |
| Swagger docs | `github.com/swaggo/gin-swagger` + `swag` CLI |
| Config | `github.com/joho/godotenv` + `os.Getenv` |

---

## Project Structure

```
simple-employees-crud/
├── cmd/
│   └── server/
│       └── main.go              # Entry point — DI wiring, server start
├── internal/
│   ├── config/
│   │   └── config.go            # Env-var config loader
│   ├── domain/
│   │   ├── employee.go          # Employee entity + EmployeeFilter
│   │   └── department.go        # Department entity + DepartmentFilter
│   ├── dto/
│   │   ├── common.go            # PaginationRequest, SearchRequest
│   │   ├── auth.go              # LoginRequest, TokenResponse, MeResponse
│   │   ├── employee.go          # Employee CRUD DTOs
│   │   └── department.go        # Department CRUD DTOs
│   ├── handler/
│   │   ├── auth.go              # Login, Me, HealthCheck
│   │   ├── employee.go          # Employee CRUD + search + avatar
│   │   ├── department.go        # Department CRUD
│   │   └── helpers.go           # Binding error formatter
│   ├── middleware/
│   │   ├── auth.go              # JWT validation + RequireRole
│   │   └── exception.go         # Centralized error handler + Recovery + RequestLogger
│   ├── repository/
│   │   ├── interface.go         # EmployeeRepository, DepartmentRepository interfaces
│   │   ├── employee.go          # PostgreSQL implementation
│   │   └── department.go        # PostgreSQL implementation
│   ├── service/
│   │   ├── auth.go              # Login, GetMe, HashPassword
│   │   ├── employee.go          # CRUD, Search, UploadAvatar + cache
│   │   ├── department.go        # CRUD + cache
│   │   └── helpers.go           # parseUUID
│   └── server/
│       └── server.go            # Gin engine, middleware, routes, graceful shutdown
├── pkg/
│   ├── apperror/
│   │   └── error.go             # AppError type + constructors
│   ├── database/
│   │   ├── postgres.go          # pgxpool, DBTX interface, WithTx, slow-query tracer
│   │   └── redis.go             # Redis client, CacheGet/Set/Del helpers
│   ├── jwt/
│   │   └── jwt.go               # Token generation and validation
│   ├── logger/
│   │   └── logger.go            # Zap logger with file + console output
│   ├── response/
│   │   └── response.go          # JSON response envelope helpers
│   └── storage/
│       └── cloudinary.go        # Avatar upload and deletion
├── migrations/
│   └── 001_init.sql             # Schema, indexes, triggers, seed admin
├── docs/                        # Generated by `swag init` — do not edit manually
├── .air.toml                    # Hot-reload config for air
├── .env.dev                     # Dev env template with REPLACE-ME placeholders
├── .gitignore
├── go.mod
└── Makefile
```

---

## Getting Started

### Prerequisites

- Go 1.23+
- PostgreSQL 15+
- Redis 7+
- (Optional) Cloudinary account for production avatar storage

### 1 — Clone and install dependencies

```bash
git clone <repo-url> && cd simple-employees-crud
go mod tidy
```

### 2 — Configure environment

Copy `.env.dev` and fill in all `REPLACE-ME` values:

```bash
cp .env.dev .env.local
# edit .env.local with your actual credentials
```

> **Important:** Change `APP_ENV=development` to load `.env.dev` automatically,
> or set `APP_ENV=production` and provide all vars through the deployment environment.

### 3 — Apply the database schema

Run the migration manually (the schema is **not** applied by the Go app):

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql
```

The SQL is idempotent (`CREATE TABLE IF NOT EXISTS`, `ON CONFLICT DO NOTHING`) and
safe to re-run.

### 4 — Set the admin password

The seed row in `001_init.sql` contains a placeholder hash. Replace it before going live:

```bash
# Generate a bcrypt hash (cost 12) for your chosen password:
go run scripts/hash_password/main.go 'YourStrongPassword!'

# Copy the output and update the INSERT in 001_init.sql, then re-run the migration.
```

### 5 — Generate Swagger docs

```bash
go install github.com/swaggo/swag/cmd/swag@latest
make swagger          # writes to ./docs/
```

### 6 — Start the server

```bash
# With hot-reload (recommended for development):
go install github.com/air-verse/air@latest
make dev

# Or build and run directly:
make run
```

The API is available at `http://localhost:8080/api/v1`.
Swagger UI at `http://localhost:8080/swagger/index.html` (development mode only).

---

## Environment Variables

All variables are documented in `.env.dev`. Key ones:

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | `development` or `production` | `development` |
| `APP_PORT` | HTTP listen port | `8080` |
| `GIN_MODE` | `debug` or `release` | `debug` |
| `POSTGRES_HOST` | PostgreSQL host | — |
| `POSTGRES_USER` | PostgreSQL username | — |
| `POSTGRES_PASSWORD` | PostgreSQL password | — |
| `POSTGRES_DB` | Database name | `employee_mgmt` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `JWT_SECRET` | HS256 signing secret (min 32 chars recommended) | — |
| `JWT_ACCESS_TOKEN_EXP_MINUTES` | Access token lifetime | `60` |
| `CLOUDINARY_CLOUD_NAME` | Cloudinary cloud | — |
| `CLOUDINARY_API_KEY` | Cloudinary API key | — |
| `CLOUDINARY_API_SECRET` | Cloudinary API secret | — |

---

## API Reference

### Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/login` | ❌ | Login, receive JWT |
| `GET`  | `/api/v1/auth/me`    | ✅ | Get current employee profile |

### Employees

| Method   | Path | Auth | Role | Description |
|----------|------|------|------|-------------|
| `GET`    | `/api/v1/employees`           | ✅ | Any | Paginated list with filters |
| `GET`    | `/api/v1/employees/search`    | ✅ | Any | Full-text search |
| `GET`    | `/api/v1/employees/:id`       | ✅ | Any | Get by ID |
| `POST`   | `/api/v1/employees`           | ✅ | admin, manager | Create employee |
| `PATCH`  | `/api/v1/employees/:id`       | ✅ | admin, manager | Partial update |
| `DELETE` | `/api/v1/employees/:id`       | ✅ | admin | Delete permanently |
| `POST`   | `/api/v1/employees/:id/avatar`| ✅ | admin, manager | Upload avatar |

### Departments

| Method   | Path | Auth | Role | Description |
|----------|------|------|------|-------------|
| `GET`    | `/api/v1/departments`       | ✅ | Any | Paginated list |
| `GET`    | `/api/v1/departments/:id`   | ✅ | Any | Get by ID |
| `POST`   | `/api/v1/departments`       | ✅ | admin | Create |
| `PATCH`  | `/api/v1/departments/:id`   | ✅ | admin | Partial update |
| `DELETE` | `/api/v1/departments/:id`   | ✅ | admin | Delete |

### Example: Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "admin@company.com",
  "password": "Admin@123"
}
```

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "employee": { ... }
  }
}
```

### Error envelope

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request validation failed",
    "details": {
      "email": "must be a valid email address",
      "password": "must be at least 8 characters long"
    }
  }
}
```

---

## Design Decisions

### Why handlers never write error responses
All errors flow through `middleware.Exception`. Handlers call `c.Error(err)` and `c.Abort()`.
This guarantees a single, consistent error shape across the entire API and makes it
impossible to accidentally leak stack traces or internal messages.

### Why PostgreSQL handles the schema (not the app)
Mixing DDL with application startup creates fragile boot sequences and makes zero-downtime
deployments harder. The DB container runs `001_init.sql` on first start; the app connects
and trusts the schema is already there.

### Transaction propagation via context
`database.WithTx` stores a `pgx.Tx` in the request context. Every repository method calls
`database.FromContext` to obtain either the active transaction or the pool. Services that
need ACID guarantees call `WithTx` and pass the resulting context to multiple repo methods —
no Tx parameter threading required.

### Redis caching strategy
- Individual records: `employee:<uuid>` TTL 15 min
- List cache: `employee:list:*` pattern deleted on any write
- Cache misses fall back to PostgreSQL transparently; cache failures are logged and do not
  surface as errors to callers.

### Cloudinary fallback
When `CLOUDINARY_CLOUD_NAME` is empty or `REPLACE-ME`, `NewCloudinaryClient` returns an error
and `main.go` sets the client pointer to `nil`. `EmployeeService.UploadAvatar` checks for nil
and falls back to a local URL string so development works without a Cloudinary account.

---

## Deployment

### Environment injection
Replace all `REPLACE-ME` values in `.env.dev` at deploy time using your CI/CD secrets manager
(AWS Parameter Store, GitHub Actions secrets, etc.). Never commit real credentials.

### Recommended production settings
```
APP_ENV=production
GIN_MODE=release
POSTGRES_MAX_CONNS=50
POSTGRES_MIN_CONNS=10
JWT_ACCESS_TOKEN_EXP_MINUTES=30
```

### Logging & observability
Logs are written as JSON to `./logs/app.log` with daily rotation (via lumberjack).
Configure Fluent Bit to tail this file and forward to your log aggregator (Loki, CloudWatch, etc.).

### Health check
`GET /health` returns `{"status":"ok"}` with HTTP 200. Use this as your Kubernetes liveness probe.
