# Go Modular Monolithic Style Guide (Gin + PostgreSQL)

> Go modular monolith style guide for high-traffic applications.
> Stack: **Go 1.23+ · Gin · PostgreSQL (pgx) · golang-migrate · zap · viper · Redis**
> This edition focuses on the **Modular Monolithic** pattern — a single deployable unit with strict domain boundaries, combining monolith simplicity with microservices-ready discipline.

---

## Table of Contents

1. [File & Directory Conventions](#1-file--directory-conventions)
2. [Modular Monolithic Architecture](#2-modular-monolithic-architecture)
3. [Domain Structure](#3-domain-structure)
4. [Handler Layer](#4-handler-layer)
5. [Service Layer](#5-service-layer)
6. [Repository Layer](#6-repository-layer)
7. [Models & DTOs](#7-models--dtos)
8. [Shared Kernel](#8-shared-kernel)
9. [Domain Communication](#9-domain-communication)
10. [Router Setup](#10-router-setup)
11. [Middleware](#11-middleware)
12. [Error Handling](#12-error-handling)
13. [Validation](#13-validation)
14. [Authentication](#14-authentication)
15. [Database & Migrations](#15-database--migrations)
16. [Configuration](#16-configuration)
17. [Performance Optimizations](#17-performance-optimizations)
18. [Testing](#18-testing)
19. [Project Tooling](#19-project-tooling)
20. [Graceful Shutdown](#20-graceful-shutdown)

---

## 1. File & Directory Conventions

| Item                   | Convention            | Example                              |
| ---------------------- | --------------------- | ------------------------------------ |
| Go source files        | snake_case            | `handler.go`, `service.go`           |
| Exported functions     | PascalCase            | `func CreateUser(...)`               |
| Unexported functions   | camelCase             | `func buildQuery(...)`               |
| Test files             | `_test.go` suffix     | `handler_test.go`, `service_test.go` |
| Migration files        | `NNNNNN_name.up.sql`  | `000001_create_users.up.sql`         |
| Directories            | kebab-case            | `internal/common/`, `internal/users/` |
| Interface names        | `[Method]er` suffix   | `UserCreator`, `UserProvider`        |
| Domain packages        | plural-snake_case     | `users`, `products`, `orders`        |
| Config structs         | PascalCase            | `Config`, `DatabaseConfig`           |

**Rules:**

- Each domain is a self-contained package — handler, service, repository, model all co-located
- Max **5 exported functions** per file — split if exceeded (e.g., `handler.go`, `handler_query.go`)
- Max **300 lines** per file
- Max **7 files** per domain directory — if exceeded, split the domain
- `internal/common/` is the **shared kernel** — only truly cross-cutting concerns
- Domains never import each other's internal packages — use interfaces or API calls
- No circular imports — enforced by Go compiler + `internal/` package boundary

```
project-root/
├── cmd/
│   ├── server/
│   │   └── main.go                 # Entry point — wire domains, start server
│   └── migrate/
│       └── main.go                 # Migration runner
│
├── internal/
│   ├── common/                     # Shared kernel — cross-cutting only
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT middleware
│   │   │   ├── ratelimit.go        # Redis rate limiter middleware
│   │   │   ├── requestid.go        # X-Request-ID middleware
│   │   │   ├── logger.go           # Structured logging middleware
│   │   │   └── cors.go             # CORS middleware
│   │   ├── cache/
│   │   │   └── redis.go            # Redis client + rate limiter
│   │   ├── response.go            # APIResponse, success(), fail()
│   │   ├── errors.go              # Shared sentinel errors
│   │   ├── config.go              # Viper-based typed config
│   │   └── validator.go            # go-playground/validator setup
│   │
│   ├── users/                      # 👤 User domain — complete vertical slice
│   │   ├── handler.go             # HTTP handlers (Create, GetByID, List, Update, Delete)
│   │   ├── service.go             # Business logic
│   │   ├── repository.go          # Data access
│   │   ├── model.go               # User struct, enums, DTOs
│   │   ├── errors.go              # Domain-specific errors
│   │   └── routes.go              # Route registration for this domain
│   │
│   ├── products/                   # 📦 Product domain
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── model.go
│   │   ├── errors.go
│   │   └── routes.go
│   │
│   └── orders/                     # 📋 Order domain
│       ├── handler.go
│       ├── service.go
│       ├── repository.go
│       ├── model.go
│       ├── errors.go
│       └── routes.go
│
├── migrations/                     # Domain-prefixed SQL files
│   ├── 000001_users_create.up.sql
│   ├── 000001_users_create.down.sql
│   ├── 000002_products_create.up.sql
│   ├── 000002_products_create.down.sql
│   ├── 000003_orders_create.up.sql
│   └── 000003_orders_create.down.sql
│
├── .env.example
├── .golangci.yml
├── Makefile
├── Dockerfile
└── go.mod
```

### Domain dependency rule

```
internal/users/           internal/products/        internal/orders/
     │                          │                        │
     │       ┌──────────────────┼────────────────────┐   │
     │       │                  │                    │   │
     └───────┼──────┬───────────┘                    │   │
             │      │                                │   │
             ▼      ▼                                ▼   ▼
       internal/common/ (shared kernel — no domain logic)
```

- Domains can import `internal/common/` — never the reverse
- Domains never import each other directly — use interfaces or events
- `internal/common/` never imports any domain
- Circular imports between domains are **impossible** by design

---

## 2. Modular Monolithic Architecture

A **Modular Monolith** is a single deployable application where code is organized into strict domain boundaries — every domain (users, products, orders) is a self-contained vertical slice. This combines the operational simplicity of a monolith with the conceptual clarity of microservices.

### Core Principles

| Principle                    | Description                                                                      |
| ---------------------------- | -------------------------------------------------------------------------------- |
| **Single Deployment**        | One Go binary, one deploy — no separate backend services                         |
| **Domain Boundaries**        | Code grouped by business domain, not by technical layer                          |
| **Explicit Public API**      | Each domain exposes its routes; internals stay private                           |
| **No Circular Dependencies** | Domains import shared kernel but never each other's internals                    |
| **Extractability**           | Any domain can be extracted into a standalone microservice with minimal friction |

### File Structure as Domain Boundaries

```
internal/
├── users/              ← User domain (complete vertical slice)
│   ├── handler.go      ← HTTP entry point (public)
│   ├── service.go      ← Business logic (public interface, private implementation)
│   ├── repository.go   ← Data access (private)
│   ├── model.go        ← Domain types (public)
│   ├── errors.go       ← Domain errors (public)
│   └── routes.go       ← Route registration (public)
├── common/             ← SHARED kernel only
│   ├── response.go     ← Response helpers
│   ├── errors.go       ← Shared errors
│   └── config.go       ← Configuration
```

### Architecture layers — per domain

Every domain follows the same layered pattern internally:

```
   HTTP Request
       │
       ▼
 ┌──────────────┐
 │   Handler     │  routes.go + handler.go — parse request, delegate, respond
 │  (~25 lines)  │
 └──────┬───────┘
        │
        ▼
 ┌──────────────┐
 │   Service     │  service.go — business logic, orchestration
 │  (~40 lines)  │
 └──────┬───────┘
        │
        ▼
 ┌──────────────┐
 │  Repository   │  repository.go — SQL queries, data access
 │  (~40 lines)  │
 └──────┬───────┘
        │
        ▼
 ┌──────────────┐
 │  Database     │  PostgreSQL (pgxpool)
 └──────────────┘
```

### Modular Monolith vs Microservices

| Aspect                 | Modular Monolith                     | Microservices                                |
| ---------------------- | ------------------------------------ | -------------------------------------------- |
| Deployment             | Single deploy                        | Multiple independent deploys                 |
| Operational complexity | Low (one binary)                     | High (Kubernetes, service mesh)              |
| Domain boundaries      | Code-level (Go packages)             | Network-level (HTTP, messaging)              |
| Extractability         | Low friction — just move the package | Already separate                             |
| Development speed      | Fast — no cross-service coordination | Slower — contracts, networking, CI pipelines |
| Team scaling           | Better for small-medium teams        | Better for large teams (10+)                 |

### When to use this pattern

| Scenario                  | Recommendation                                           |
| ------------------------- | -------------------------------------------------------- |
| Startup / MVP             | ✅ Modular monolith — ship fast, refine boundaries later |
| Internal tool             | ✅ Modular monolith — no distributed complexity needed   |
| Small team (1–5 devs)     | ✅ Modular monolith — faster iteration                   |
| Large team (10+ devs)     | ⚠️ Consider microservices — independent deploys          |
| Different tech per domain | ⚠️ Microservices — modular monolith uses one stack       |

---

## 3. Domain Structure

Each domain is a complete vertical slice. Every file has a single responsibility.

### Domain package layout

```
internal/users/
├── handler.go           # HTTP handlers — thin, delegates to service
├── service.go           # Business logic — exported interface, unexported struct
├── repository.go        # Data access — unexported, created by service
├── model.go             # Domain model + request/response DTOs
├── errors.go            # Domain-specific sentinel errors
└── routes.go            # Registers routes on the Gin engine
```

### File responsibility

| File | Responsibility | Exported |
|---|---|---|
| `handler.go` | Parse HTTP, validate input, call service, write response | `NewHandler()`, `Handler` struct |
| `service.go` | Business rules, orchestration, transaction boundaries | `NewService()`, `Service` interface |
| `repository.go` | SQL queries, scans, connection management | `NewRepository()`, `Repository` interface |
| `model.go` | Domain structs, enums, request/response DTOs | All types |
| `errors.go` | Sentinel errors for this domain | All errors |
| `routes.go` | Register routes on Gin group | `RegisterRoutes()` |

### Domain rules

| Rule | Reason |
|---|---|
| Each domain has its own models | No shared DTOs — prevents coupling |
| Service is defined as an interface in the domain | Enables mocking in tests, future extraction |
| Repository is an interface in the domain | Swap implementations, test with mocks |
| Handler never calls repository directly | Strict layered isolation |
| Errors are defined per domain | Granular error handling, no shared error bloat |

### routes.go — each domain registers itself

```go
// internal/users/routes.go
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, m *common.Middleware) {
    users := rg.Group("/users")
    users.Use(m.Auth())
    {
        users.GET("", h.List)
        users.GET("/:id", h.GetByID)
        users.POST("", h.Create)
        users.PATCH("/:id", h.Update)
        users.DELETE("/:id", h.Delete)
    }
}
```

---

## 4. Handler Layer

Handlers live inside each domain. Rules are identical to the monolith — thin, ~25 lines.

### Handler rules

| Rule                          | Max        | Reason                                  |
| ----------------------------- | ---------- | --------------------------------------- |
| Lines per handler function    | **~25 (soft)** | Aim for ≤25; ok to exceed when needed |
| Exported functions per file   | **5**      | Split if exceeded (e.g., `handler_query.go`) |
| Dependencies via constructor  | required   | Explicit via struct injection           |

```go
// internal/users/handler.go — ✅ Good
type Handler struct {
    svc Service
}

func NewHandler(svc Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Fail(c, 400, "VALIDATION_ERROR", common.FormatValidationError(err))
        return
    }
    user, err := h.svc.Create(c.Request.Context(), &req)
    if err != nil {
        common.HandleError(c, err)
        return
    }
    common.Success(c, 201, user)
}

// ❌ Bad — fat handler, missing validation, no error mapping
func CreateUserHandler(c *gin.Context) {
    var req CreateUserRequest
    c.ShouldBindJSON(&req)
    // raw SQL inline...
    c.JSON(200, gin.H{"user_id": id})
}
```

### Response helpers (shared kernel)

```go
// internal/common/response.go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func Success(c *gin.Context, status int, data interface{}) {
    c.JSON(status, APIResponse{Success: true, Data: data})
}

func Fail(c *gin.Context, status int, code, message string) {
    c.JSON(status, APIResponse{
        Success: false,
        Error:   &APIError{Code: code, Message: message},
    })
}
```

---

## 5. Service Layer

Services contain business logic. Defined as interfaces for testability and future extraction.

### Service rules

| Rule                          | Max        | Reason                                  |
| ----------------------------- | ---------- | --------------------------------------- |
| Lines per service function    | **~40 (soft)** | Aim for ≤40; ok to exceed when needed |
| Exported functions per file   | **5**      | One domain per file                     |
| Return domain errors          | required   | Never return HTTP concepts              |

### Interface + implementation pattern

```go
// internal/users/service.go — ✅ Good

// Service is the public contract for the user domain.
type Service interface {
    Create(ctx context.Context, req *CreateUserRequest) (*User, error)
    GetByID(ctx context.Context, id string) (*User, error)
    List(ctx context.Context, filter *UserFilter) ([]User, int, error)
}

type service struct {
    repo Repository
    val  *validator.Validate
}

func NewService(repo Repository, val *validator.Validate) Service {
    return &service{repo: repo, val: val}
}

func (s *service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
    if err := s.val.Struct(req); err != nil {
        return nil, ErrValidationFailed
    }
    exists, err := s.repo.ExistsByEmail(ctx, req.Email)
    if err != nil {
        return nil, fmt.Errorf("check email: %w", err)
    }
    if exists {
        return nil, ErrEmailAlreadyExists
    }
    hashed, err := hashPassword(req.Password)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }
    user := &User{
        Name:     req.Name,
        Email:    req.Email,
        Password: hashed,
    }
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }
    return user, nil
}

// ❌ Bad — no interface, concrete struct exposed
type UserService struct {
    repo *UserRepository  // ← can't mock
}

func (s *UserService) Create(...) { ... }
```

### Transaction across domains

```go
// ✅ Good — transaction helper via pool
type service struct {
    pool  *pgxpool.Pool
    user  user.Repository      // interface
    order order.Repository     // interface
}

func (s *service) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*Order, error) {
    var order *Order
    err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
        if err := s.user.DeductBalanceTx(ctx, tx, req.UserID, req.Total); err != nil {
            return err
        }
        o, err := s.order.CreateTx(ctx, tx, req.ToOrder())
        if err != nil {
            return err
        }
        order = o
        return nil
    })
    return order, err
}

// ❌ Bad — service manages transactions from other domains directly
func (s *service) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*Order, error) {
    tx, _ := s.pool.Begin(ctx)
    s.userRepo.DeductBalance(ctx, req.UserID, req.Total)  // ← wrong pool
    s.orderRepo.Create(ctx, req.ToOrder())
    tx.Commit(ctx)
}
```

---

## 6. Repository Layer

Repositories handle data access. Each domain owns its repository.

### Repository rules

| Rule                          | Max        | Reason                                  |
| ----------------------------- | ---------- | --------------------------------------- |
| Lines per function            | **~40 (soft)** | Aim for ≤40; ok to exceed when needed |
| Exported functions per file   | **5**      | One domain per file                     |
| No business logic             | enforced   | Repository is pure data access          |

### Interface + implementation

```go
// internal/users/repository.go — ✅ Good

// Repository is the data contract for the user domain.
type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    List(ctx context.Context, filter *UserFilter) ([]User, int, error)
    ExistsByEmail(ctx context.Context, email string) (bool, error)
    DeductBalanceTx(ctx context.Context, tx pgx.Tx, userID string, amount float64) error
}

type repository struct {
    pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
    return &repository{pool: pool}
}

func (r *repository) List(ctx context.Context, filter *UserFilter) ([]User, int, error) {
    query := `
        SELECT id, name, email, is_active, created_at, updated_at,
               COUNT(*) OVER() AS total_count
        FROM users
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
    rows, err := r.pool.Query(ctx, query, filter.Limit, filter.Offset)
    if err != nil {
        return nil, 0, fmt.Errorf("list users: %w", err)
    }
    defer rows.Close()

    var users []User
    var total int
    for rows.Next() {
        var u User
        if err := rows.Scan(
            &u.ID, &u.Name, &u.Email, &u.IsActive,
            &u.CreatedAt, &u.UpdatedAt, &total,
        ); err != nil {
            return nil, 0, fmt.Errorf("scan user: %w", err)
        }
        users = append(users, u)
    }
    return users, total, nil
}
```

### Batch operations

```go
// ✅ Good — batch methods live in the domain's repository
func (r *repository) BatchCreate(ctx context.Context, users []*User) ([]*User, error) {
    query := `
        INSERT INTO users (name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `
    batch := &pgx.Batch{}
    for _, u := range users {
        batch.Queue(query, u.Name, u.Email, u.Password)
    }
    br := r.pool.SendBatch(ctx, batch)
    defer br.Close()
    for _, u := range users {
        if err := br.QueryRow().Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
            return nil, fmt.Errorf("batch create: %w", err)
        }
    }
    return users, nil
}

func (r *repository) BatchDelete(ctx context.Context, ids []string) error {
    query := `UPDATE users SET deleted_at = NOW() WHERE id = ANY($1)`
    _, err := r.pool.Exec(ctx, query, ids)
    return fmt.Errorf("batch delete: %w", err)
}
```

### Connection pool (shared kernel)

```go
// internal/common/database.go
func NewPool(ctx context.Context, cfg DatabaseConfig) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("parse pool config: %w", err)
    }
    config.MaxConns = cfg.MaxConns
    config.MinConns = cfg.MinConns
    config.MaxConnLifetime = 30 * time.Minute
    config.HealthCheckPeriod = 30 * time.Second
    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }
    return pool, nil
}

// internal/common/redis.go
func NewRedis(cfg RedisConfig) (*redis.Client, error) {
    opts, err := redis.ParseURL(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("parse redis url: %w", err)
    }
    if cfg.Password != "" {
        opts.Password = cfg.Password
    }
    client := redis.NewClient(opts)
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, fmt.Errorf("ping redis: %w", err)
    }
    return client, nil
}
```

---

## 7. Models & DTOs

Each domain defines its own models and DTOs. No shared DTOs between domains.

```go
// internal/users/model.go — ✅ Good
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`
    Role      Role      `json:"role"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

type CreateUserRequest struct {
    Name            string `json:"name"     validate:"required,min=2,max=100"`
    Email           string `json:"email"    validate:"required,email"`
    Password        string `json:"password" validate:"required,min=8"`
}

type UserResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Role      Role      `json:"role"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
}

func ToUserResponse(u *User) *UserResponse {
    return &UserResponse{
        ID:        u.ID,
        Name:      u.Name,
        Email:     u.Email,
        Role:      u.Role,
        IsActive:  u.IsActive,
        CreatedAt: u.CreatedAt,
    }
}

// ❌ Bad — shared model across domains
type User struct { ... }   // in internal/common/ ← never!
```

### Domain errors

```go
// internal/users/errors.go — ✅ Good
var (
    ErrNotFound         = errors.New("user not found")
    ErrEmailAlreadyExists = errors.New("email already exists")
    ErrValidationFailed = errors.New("validation failed")
)
```

---

## 8. Shared Kernel

The shared kernel (`internal/common/`) contains only truly cross-cutting concerns. It never imports any domain.

### What belongs in common

| File | Purpose |
|---|---|
| `response.go` | `Success()`, `Fail()`, `APIResponse` struct |
| `errors.go` | `HandleError()`, standard error mapping |
| `config.go` | Typed config struct, viper loader |
| `validator.go` | Validator setup, custom validators |
| `database.go` | Connection pool factory |
| `middleware/auth.go` | JWT middleware |
| `middleware/ratelimit.go` | Redis rate limiter |
| `middleware/requestid.go` | Request ID middleware |
| `middleware/logger.go` | Structured logging middleware |
| `middleware/cors.go` | CORS middleware |
| `cache/redis.go` | Redis client + rate limiter |

### What does NOT belong in common

```go
// ❌ Bad — domain logic in shared kernel
package common

type User struct { ... }          // ← belongs in internal/users/
type Product struct { ... }       // ← belongs in internal/products/
var ErrUserNotFound               // ← belongs in internal/users/

// ✅ Good — common only has infrastructure
package common

func HandleError(c *gin.Context, err error) { ... }
func Success(c *gin.Context, status int, data interface{}) { ... }
func NewPool(cfg DatabaseConfig) (*pgxpool.Pool, error) { ... }
```

### Config struct (shared kernel)

```go
// internal/common/config.go
type Config struct {
    Port      string          `mapstructure:"PORT"`
    LogLevel  string          `mapstructure:"LOG_LEVEL"`
    Database  DatabaseConfig  `mapstructure:",squash"`
    Redis     RedisConfig     `mapstructure:",squash"`
    RateLimit RateLimitConfig `mapstructure:",squash"`
    JWT       JWTConfig       `mapstructure:",squash"`
}
```

---

## 9. Domain Communication

Domains never import each other. Communication happens through:

### 1. Service interfaces (recommended)

Define the interface in the consuming domain, inject the implementation at startup.

```go
// internal/orders/service.go — Order domain needs user info

// UserService is the interface orders need from the user domain.
type UserService interface {
    GetByID(ctx context.Context, id string) (*users.User, error)
    DeductBalance(ctx context.Context, userID string, amount float64) error
}

type service struct {
    userSvc UserService   // injected at wiring time
    repo    Repository
}

func NewService(userSvc UserService, repo Repository) Service {
    return &service{userSvc: userSvc, repo: repo}
}
```

```go
// cmd/server/main.go — wiring
userSvc := users.NewService(userRepo, val)
orderSvc := orders.NewService(userSvc, orderRepo)   // ✅ inject by interface
```

### 2. Events / asynchronous (for eventual consistency)

```go
// internal/common/events.go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(topic string, handler EventHandler)
}

// internal/orders/service.go
func (s *service) Create(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
    order, err := s.repo.Create(ctx, req.ToOrder())
    if err != nil {
        return nil, err
    }
    s.bus.Publish(ctx, Event{
        Topic: "order.created",
        Data:  order,
    })
    return order, nil
}
```

### 3. Direct repository sharing (not recommended)

```go
// ❌ Bad — order domain imports user repository directly
import "internal/users"  // ← creates tight coupling

func (s *service) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*Order, error) {
    user, err := users.NewRepository(s.pool).GetByID(ctx, req.UserID)
    // ...
}
```

---

## 10. Router Setup

Each domain registers its own routes. The main function only wires dependencies.

### Wire everything in main.go

```go
// cmd/server/main.go
func main() {
    cfg := common.LoadConfig()
    log := zap.NewExample()

    pool, err := common.NewPool(cfg.Database)
    if err != nil {
        log.Fatal("failed to connect to database", zap.Error(err))
    }
    defer pool.Close()

    rdb, err := common.NewRedis(cfg.Redis)
    if err != nil {
        log.Warn("redis unavailable, rate limiting disabled", zap.Error(err))
    }
    if rdb != nil {
        defer rdb.Close()
    }

    val := common.NewValidator()

    // Init repositories
    userRepo := users.NewRepository(pool)
    productRepo := products.NewRepository(pool)
    orderRepo := orders.NewRepository(pool)

    // Init services
    userSvc := users.NewService(userRepo, val)
    productSvc := products.NewService(productRepo, val)
    orderSvc := orders.NewService(userSvc, productSvc, orderRepo, val)

    // Init handlers
    userHdl := users.NewHandler(userSvc)
    productHdl := products.NewHandler(productSvc)
    orderHdl := orders.NewHandler(orderSvc)

    // Init middleware
    mw := common.NewMiddleware(cfg, rdb, log)

    // Router
    r := gin.New()
    r.Use(mw.RequestID())
    r.Use(mw.Logger())
    r.Use(mw.Recovery())
    r.Use(mw.CORS())

    // Health
    r.GET("/health", func(c *gin.Context) {
        common.Success(c, 200, gin.H{"status": "ok"})
    })

    // Each domain registers its routes
    api := r.Group("/api/v1")
    {
        users.RegisterRoutes(api, userHdl, mw)
        products.RegisterRoutes(api, productHdl, mw)
        orders.RegisterRoutes(api, orderHdl, mw)
    }

    // Graceful shutdown
    // ...
}
```

### Each domain owns its routes

```go
// internal/users/routes.go
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *common.Middleware) {
    u := rg.Group("/users")
    u.Use(mw.Auth())
    {
        u.GET("", h.List)
        u.GET("/:id", h.GetByID)
        u.POST("", h.Create)
        u.PATCH("/:id", h.Update)
        u.DELETE("/:id", h.Delete)
    }
}

// ✅ Good — auth is domain's concern, applied per-domain
// ❌ Bad — flat routes, all in main.go
r.GET("/api/v1/users", userHdl.List)   // ← scattered route definition
```

---

## 11. Middleware

Middleware is defined once in the shared kernel and applied per-domain at registration time.

### Middleware list

| Middleware       | Order | Purpose                                          |
| ---------------- | ----- | ------------------------------------------------ |
| Recovery         | 1     | Panic recovery — always first                    |
| RequestID        | 2     | Add X-Request-ID to every request/response       |
| Logger           | 3     | Structured request logging (zap)                 |
| CORS             | 4     | CORS headers for frontend                        |
| Auth             | 5     | JWT validation — applied per-domain              |
| Rate Limit       | 6     | Per-IP or per-user rate limiting                 |
| Timeout          | 7     | Request timeout — always last in chain           |

### Auth middleware

```go
// internal/common/middleware/middleware.go
type Middleware struct {
    cfg common.Config
    rdb *redis.Client
    log *zap.Logger
}

func NewMiddleware(cfg common.Config, rdb *redis.Client, log *zap.Logger) *Middleware {
    return &Middleware{cfg: cfg, rdb: rdb, log: log}
}

// internal/common/middleware/auth.go
func (mw *Middleware) Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractBearerToken(c.GetHeader("Authorization"))
        if token == "" {
            common.Fail(c, 401, "UNAUTHORIZED", "missing token")
            c.Abort()
            return
        }
        claims, err := validateJWT(token, mw.cfg.JWT.Secret)
        if err != nil {
            common.Fail(c, 401, "UNAUTHORIZED", "invalid token")
            c.Abort()
            return
        }
        c.Set("user_id", claims.UserID)
        c.Set("user_role", claims.Role)
        c.Next()
    }
}
```

### Rate limiter middleware (Redis)

```go
// internal/common/middleware/ratelimit.go
func (mw *Middleware) RateLimit() gin.HandlerFunc {
    rl := cache.NewRateLimiter(mw.rdb, mw.cfg.RateLimit.Limit, mw.cfg.RateLimit.Window)
    return func(c *gin.Context) {
        key := c.ClientIP()
        if userID, exists := c.Get("user_id"); exists {
            key = fmt.Sprintf("user:%v", userID)
        }
        allowed, remaining, err := rl.Allow(c.Request.Context(), key)
        if err != nil {
            c.Next()
            return
        }
        c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
        if !allowed {
            common.Fail(c, 429, "RATE_LIMIT_EXCEEDED", "too many requests")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Per-domain middleware

```go
// ✅ Good — each domain decides its own middleware
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *common.Middleware) {
    rg.POST("/login", h.Login)                              // no auth
    rg.POST("/register", h.Register)                        // no auth

    authenticated := rg.Group("")
    authenticated.Use(mw.Auth())
    authenticated.Use(mw.RateLimit())                       // rate limit applies here
    {
        authenticated.GET("/profile", h.GetProfile)
        authenticated.PATCH("/profile", h.UpdateProfile)
    }
}
```

---

## 12. Error Handling

### Error mapping (shared kernel)

```go
// internal/common/errors.go
// HandleError maps domain errors to HTTP responses.
// Each domain registers its error types via error mapping functions.
var errorMappers []func(error) (int, string)

func RegisterErrorMapper(fn func(error) (int, string)) {
    errorMappers = append(errorMappers, fn)
}

func HandleError(c *gin.Context, err error) {
    for _, mapper := range errorMappers {
        if status, code := mapper(err); status != 0 {
            Fail(c, status, code, err.Error())
            return
        }
    }
    log.Error("unexpected error", zap.Error(err))
    Fail(c, 500, "INTERNAL_ERROR", "an unexpected error occurred")
}

// Each domain registers its error mappers in its own init().
// Example from internal/users/errors.go:
//
// func init() {
//     common.RegisterErrorMapper(func(err error) (int, string) {
//         switch {
//         case errors.Is(err, ErrNotFound):          return 404, "NOT_FOUND"
//         case errors.Is(err, ErrEmailAlreadyExists): return 409, "CONFLICT"
//         case errors.Is(err, ErrValidationFailed):    return 400, "VALIDATION_ERROR"
//         default:                                    return 0,  ""
//         }
//     })
// }
```

### Error wrapping

```go
// ✅ Good — wrap with context
if err != nil {
    return nil, fmt.Errorf("get user %s: %w", id, err)
}

// ❌ Bad — bare error
return nil, err
```

---

## 13. Validation

Validation setup is in the shared kernel. Each domain uses it on its own DTOs.

```go
// internal/common/validator.go
func NewValidator() *validator.Validate {
    v := validator.New()
    v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
        return true
    })
    v.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := fld.Tag.Get("json")
        return strings.Split(name, ",")[0]
    })
    return v
}

func validationMessage(fe validator.FieldError) string {
    switch fe.Tag() {
    case "required":
        return fe.Field() + " is required"
    case "email":
        return "must be a valid email address"
    case "min":
        return fe.Field() + " must be at least " + fe.Param() + " characters"
    case "max":
        return fe.Field() + " must be at most " + fe.Param() + " characters"
    case "password":
        return "must contain at least one uppercase, one lowercase, and one number"
    default:
        return fe.Field() + " is invalid"
    }
}

func FormatValidationError(err error) []map[string]string {
    var ve validator.ValidationErrors
    if !errors.As(err, &ve) {
        return nil
    }
    out := make([]map[string]string, len(ve))
    for i, fe := range ve {
        out[i] = map[string]string{
            "field":   fe.Field(),
            "message": validationMessage(fe),
        }
    }
    return out
}
```

---

## 14. Authentication

Authentication is handled by the shared kernel middleware.

### Token generation

```go
// internal/common/auth.go
type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func GenerateToken(userID, role, secret string, duration time.Duration) (string, error) {
    claims := Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "my-app",
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

---

## 15. Database & Migrations

### Domain-prefixed migration files

```
migrations/
├── 000001_users_create.up.sql
├── 000001_users_create.down.sql
├── 000002_products_create.up.sql
├── 000002_products_create.down.sql
├── 000003_orders_create.up.sql
└── 000003_orders_create.down.sql
```

```sql
-- 000001_users_create.up.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    role       VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
```

### Migration runner

```go
// cmd/migrate/main.go
func main() {
    cfg := common.LoadConfig()
    m, err := migrate.New("file://migrations", cfg.Database.URL)
    if err != nil {
        log.Fatal(err)
    }
    cmd := os.Args[1]
    switch cmd {
    case "up":
        if err := m.Up(); err != nil && err != migrate.ErrNoChange {
            log.Fatal(err)
        }
    case "down":
        if err := m.Down(); err != nil && err != migrate.ErrNoChange {
            log.Fatal(err)
        }
    }
}
```

---

## 16. Configuration

```go
// internal/common/config.go
type Config struct {
    Port      string          `mapstructure:"PORT"`
    LogLevel  string          `mapstructure:"LOG_LEVEL"`
    Database  DatabaseConfig  `mapstructure:",squash"`
    Redis     RedisConfig     `mapstructure:",squash"`
    RateLimit RateLimitConfig `mapstructure:",squash"`
    JWT       JWTConfig       `mapstructure:",squash"`
}

type DatabaseConfig struct {
    URL      string `mapstructure:"DATABASE_URL"`
    MaxConns int32  `mapstructure:"DB_MAX_CONNS"`
    MinConns int32  `mapstructure:"DB_MIN_CONNS"`
}

type RedisConfig struct {
    URL      string `mapstructure:"REDIS_URL"`
    Password string `mapstructure:"REDIS_PASSWORD"`
}

type RateLimitConfig struct {
    Enabled bool          `mapstructure:"RATE_LIMIT_ENABLED"`
    Limit   int           `mapstructure:"RATE_LIMIT_LIMIT"`
    Window  time.Duration `mapstructure:"RATE_LIMIT_WINDOW"`
}

type JWTConfig struct {
    Secret     string        `mapstructure:"JWT_SECRET"`
    AccessTTL  time.Duration `mapstructure:"JWT_ACCESS_TTL"`
    RefreshTTL time.Duration `mapstructure:"JWT_REFRESH_TTL"`
}

func LoadConfig() Config {
    v := viper.New()
    v.SetConfigFile(".env")
    v.SetConfigType("env")
    v.AutomaticEnv()
    v.SetDefault("PORT", "8080")
    v.SetDefault("DB_MAX_CONNS", 25)
    v.SetDefault("JWT_ACCESS_TTL", "15m")

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        log.Fatal("failed to parse config", zap.Error(err))
    }
    return cfg
}
```

---

## 17. Performance Optimizations

### Connection pooling

```go
config.MaxConns = 50
config.MinConns = 10
config.MaxConnLifetime = 30 * time.Minute
config.HealthCheckPeriod = 30 * time.Second
```

### Context timeouts

```go
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()
user, err := h.svc.Create(ctx, &req)
```

### JSON pooling

```go
var bufferPool = sync.Pool{
    New: func() interface{} { return &bytes.Buffer{} },
}

func writeJSON(c *gin.Context, status int, data interface{}) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()
    json.NewEncoder(buf).Encode(data)
    c.Data(status, "application/json; charset=utf-8", buf.Bytes())
}
```

### General rules

| Practice                     | Reason                                         |
| ---------------------------- | ---------------------------------------------- |
| Use `sync.Pool` for hot paths | Reduce GC pressure under high load             |
| Set `GOMAXPROCS`             | Match CPU quota in container environments      |
| Prepared statements          | pgx does this automatically                    |
| Always `defer rows.Close()`  | Prevent connection leaks                       |
| Add DB indexes early         | Index `WHERE`, `ORDER BY`, `JOIN` columns      |

---

## 18. Testing

### Unit tests — mock domain interfaces

```go
// internal/users/service_test.go
func TestService_Create(t *testing.T) {
    mockRepo := new(MockRepository)
    mockRepo.On("ExistsByEmail", mock.Anything, "john@test.com").Return(false, nil)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

    svc := NewService(mockRepo, validator.New())
    user, err := svc.Create(context.Background(), &CreateUserRequest{
        Name:     "John",
        Email:    "john@test.com",
        Password: "SecurePass123",
    })

    assert.NoError(t, err)
    assert.NotNil(t, user)
    mockRepo.AssertExpectations(t)
}
```

### Handler tests

```go
func TestHandler_Create(t *testing.T) {
    mockSvc := new(MockService)
    h := NewHandler(mockSvc, validator.New())
    router := gin.New()
    router.POST("/api/v1/users", h.Create)

    body := `{"name":"John","email":"john@test.com","password":"SecurePass123"}`
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    router.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)
}
```

### Test organization — co-located per domain

```
internal/
├── users/
│   ├── handler.go
│   ├── handler_test.go            # ← co-located
│   ├── service.go
│   ├── service_test.go            # ← co-located
│   ├── repository.go
│   ├── repository_test.go         # ← co-located
│   ├── model.go
│   ├── errors.go
│   └── routes.go
├── products/
│   └── ...
└── orders/
    └── ...
```

---

## 19. Project Tooling

### Makefile

```makefile
.PHONY: run build test lint migrate-up migrate-down

run:
	go run ./cmd/server/main.go

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server
	CGO_ENABLED=0 go build -o bin/migrate ./cmd/migrate

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
```

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bin/server ./cmd/server
RUN CGO_ENABLED=0 go build -o bin/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/server .
COPY --from=builder /app/bin/migrate .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./server"]
```

### .golangci.yml

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - ineffassign
    - misspell
    - revive

linters-settings:
  revive:
    rules:
      - name: exported
        severity: warning
```

---

## 20. Graceful Shutdown

```go
// cmd/server/main.go
func main() {
    // ... setup ...

    srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        log.Info("server started", zap.String("port", cfg.Port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("server error", zap.Error(err))
        }
    }()

    <-quit
    log.Info("shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("server forced to shutdown", zap.Error(err))
    }

    if rdb != nil {
        rdb.Close()
    }
    pool.Close()
    log.Info("server exited")
}
```

### Shutdown order

1. **Stop accepting new requests** — `srv.Shutdown(ctx)`
2. **Drain in-flight requests**
3. **Close Redis client**
4. **Close database pool**
5. **Flush logs**
6. **Exit**

---

## Appendix: Rule Reference Card

| Layer      | Max Lines/Function | Max Exports/File | Package          |
| ---------- | ------------------ | ---------------- | ---------------- |
| Handler    | ~25 (soft)         | 5                | Per domain       |
| Service    | ~40 (soft)         | 5                | Per domain       |
| Repository | ~40 (soft)         | 5                | Per domain       |
| Middleware | 30                 | 1                | `common/middleware` |
| Config     | —                  | 2                | `common`         |
| Model      | —                  | —                | Per domain       |

### General rules

- **Domains never import each other** — use interfaces or events
- **Shared kernel never imports domains** — strict one-way dependency
- **Max 7 files per domain directory** — split domain if exceeded
- Each domain has its own: handler, service, repository, model, errors, routes
- Service and Repository are **interfaces** — enables mocking and extraction
- Always wrap errors with `fmt.Errorf("context: %w", err)`
- Never use `SELECT *` in production queries
- Always set request context timeouts

