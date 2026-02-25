Go Microservice Template

Go microservice template with gRPC & REST API, clean architecture, Docker support, and CI/CD pipeline. Built from real-world experience architecting systems serving 700,000+ users.

#Features

- **Dual API Layer** — gRPC (internal) + REST/HTTP (external) with shared business logic
- **Clean Architecture** — Handler → Service → Repository pattern with dependency injection
- **Database Ready** — PostgreSQL with migrations, connection pooling, and health checks
- **Redis Caching** — Built-in caching layer with TTL management
- **Authentication** — JWT middleware with role-based access control
- **Observability** — Structured logging (zerolog), Prometheus metrics, health endpoints
- **Docker** — Multi-stage build producing <20MB images
- **CI/CD** — GitHub Actions pipeline with test, lint, build, and push
- **Graceful Shutdown** — Proper signal handling for zero-downtime deployments
- **Rate Limiting** — Token bucket rate limiter middleware
- **Configuration** — Environment-based config with validation

# Architecture

```
┌─────────────────────────────────────────────────┐
│                  API Gateway                     │
│              (Nginx / Traefik)                   │
└──────────┬──────────────────┬───────────────────┘
           │                  │
     ┌─────▼─────┐     ┌─────▼─────┐
     │  REST API  │     │ gRPC API  │
     │  (HTTP/1)  │     │ (HTTP/2)  │
     └─────┬─────┘     └─────┬─────┘
           │                  │
     ┌─────▼──────────────────▼─────┐
     │         Handler Layer         │
     │    (Request/Response DTOs)    │
     └──────────────┬───────────────┘
                    │
     ┌──────────────▼───────────────┐
     │        Service Layer          │
     │     (Business Logic)          │
     └──────────────┬───────────────┘
                    │
     ┌──────────────▼───────────────┐
     │      Repository Layer         │
     │   (Data Access / Cache)       │
     └───────┬──────────┬───────────┘
             │          │
     ┌───────▼───┐ ┌───▼────────┐
     │ PostgreSQL │ │   Redis    │
     │  (Primary) │ │  (Cache)   │
     └───────────┘ └────────────┘
```

#Quick Start

#Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Protocol Buffers compiler (`protoc`)

### Run with Docker Compose

```bash
# Clone the repository
git clone https://github.com/AmirHossenAshraf/go-microservice-template.git
cd go-microservice-template

# Start all services (app + postgres + redis)
docker-compose up -d

# Check health
curl http://localhost:8080/health

# Run migrations
docker-compose exec app ./scripts/migrate.sh up
```

### Run Locally

```bash
# Install dependencies
go mod download

# Generate protobuf code
make proto

# Run database migrations
make migrate-up

# Start the server
make run
```

## 📁 Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── handler/
│   │   ├── grpc_handler.go      # gRPC request handlers
│   │   └── http_handler.go      # REST request handlers
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication
│   │   ├── logging.go           # Request logging
│   │   ├── ratelimit.go         # Rate limiting
│   │   └── recovery.go          # Panic recovery
│   ├── model/
│   │   └── user.go              # Domain models
│   ├── repository/
│   │   ├── postgres.go          # PostgreSQL implementation
│   │   └── cache.go             # Redis cache layer
│   └── service/
│       └── user_service.go      # Business logic
├── proto/
│   └── user/
│       └── user.proto           # Protocol Buffers definitions
├── api/
│   └── user/
│       └── user.pb.go           # Generated protobuf code
├── docker/
│   └── Dockerfile               # Multi-stage Docker build
├── migrations/
│   └── 001_create_users.sql     # Database migrations
├── scripts/
│   ├── migrate.sh               # Migration runner
│   └── generate_proto.sh        # Protobuf code generation
├── .github/
│   └── workflows/
│       └── ci.yml               # CI/CD pipeline
├── docker-compose.yml           # Local development setup
├── Makefile                     # Build commands
└── go.mod
```

## 🔌 API Reference

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/api/v1/users` | Create user |
| `GET` | `/api/v1/users/:id` | Get user by ID |
| `PUT` | `/api/v1/users/:id` | Update user |
| `DELETE` | `/api/v1/users/:id` | Delete user |
| `GET` | `/api/v1/users` | List users (paginated) |

### gRPC Services

```protobuf
service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (Empty);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

## ⚙️ Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `9090` | gRPC server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `microservice` | Database name |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_SECRET` | — | JWT signing key |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

## 🧪 Testing

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# Coverage report
make coverage
```

## 🐳 Docker

The multi-stage Dockerfile produces minimal images (~18MB):

```bash
# Build image
docker build -f docker/Dockerfile -t go-microservice .

# Run container
docker run -p 8080:8080 -p 9090:9090 go-microservice
```

## 📊 Performance

Benchmarked on a 2-core VM with PostgreSQL:

| Metric | Value |
|--------|-------|
| REST API (GET) | ~2,500 req/s |
| gRPC (GetUser) | ~8,000 req/s |
| P99 Latency (REST) | <15ms |
| P99 Latency (gRPC) | <5ms |
| Docker Image Size | ~18MB |
| Memory Usage | ~25MB idle |



⭐ If you find this useful, please star the repository!
