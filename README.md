# PureGo Multi Warehouse

A production-style Go backend for multi-warehouse inventory management with authentication, role-based authorization, idempotent write requests, rate limiting, audit logging, Redis caching, worker-based async jobs, and webhook notifications.

## Project Intro

This project is a REST API built with Go's standard `net/http` router style (method-aware patterns), PostgreSQL for persistent data, and Redis for cache.

It is designed around multi-warehouse stock and product management with:
- User signup/login and email verification
- Product CRUD with soft delete
- Stock batch tracking and FIFO deduction
- Multi-warehouse stock transfer with audit trails
- Admin dashboards for global and warehouse-level inventory
- Full-text product search with caching
- Background jobs (low-stock email alerts + webhook notifications)

## Techniques and Tools Used

## Language and Runtime
- Go 1.24+

## Core Libraries
- `net/http` for HTTP server and routing
- `github.com/jackc/pgx/v5/stdlib` for PostgreSQL driver
- `github.com/redis/go-redis/v9` for Redis cache
- `github.com/golang-jwt/jwt/v5` for JWT auth
- `golang.org/x/time/rate` for IP rate limiting
- `golang.org/x/crypto/bcrypt` for password hashing
- `github.com/joho/godotenv` for `.env` loading

## Data and Infrastructure
- PostgreSQL (users, products, audit logs, stock logs, idempotency keys, webhooks, warehouse batches)
- Redis (dashboard and search cache)

## Security and Reliability Patterns
- JWT authentication middleware
- Role-based access control (RBAC) middleware
- Idempotency middleware for write endpoints
- Soft delete strategy for products
- Background worker queue for async side effects

## Project Structure

- `main.go`: app bootstrapping, DB/Redis connection, worker pool, graceful shutdown.
- `routes/route.go`: all HTTP route registration.
- `controllers/`: request handlers and API layer.
- `models/`: DB operations and business-level data logic.
- `middleware/`: auth, RBAC, idempotency, logger, rate limiting.
- `utils/`: token generation, password hashing, mailer, workers, webhooks, audit helper.
- `config/`: DB and Redis configuration.

## How It Works

1. Server startup
- Loads env variables
- Connects to PostgreSQL and Redis
- Starts worker pool
- Registers all routes with middleware
- Starts HTTP server on `:8080`

2. Request flow
- Global logging + rate limit middleware
- Route matching
- Optional auth/idempotency/RBAC middleware
- Controller validation and input parsing
- Model layer performs DB transaction/query
- JSON response returned

3. Async flow
- Low stock and webhook tasks are sent to `utils.JobQueue`
- Worker pool consumes jobs in background
- Email/webhook side effects happen outside request latency path

## Features

## Authentication
- Signup (`POST /signup`)
- Login (`POST /login`)
- Verify email (`GET /verify-email`)
- Forgot password (`POST /forgot-password`)
- Reset password (`POST /reset-password`)

## Product APIs
- Get products (`GET /products`)
- Search products (`GET /products/search`)
- Cursor-style pagination (`GET /products/paginated`)
- Get product by ID (`GET /products/{id}`)
- Create product (`POST /products`) [Auth + Idempotency]
- Update product quantity with audit (`PUT /products/{id}`) [Auth + Idempotency]
- Alternate stock update route (`PUT /products/{id}/stock`) [Auth + Idempotency]
- Soft delete product (`DELETE /products/{id}`) [Auth + Idempotency]

## Stock and Warehouse APIs
- Transfer stock across warehouses (`POST /stock/transfer`) [Auth + Idempotency]

## Admin APIs
- Product dashboard (`GET /admin/dashboard`) [Auth + Admin role]
- Multi-warehouse dashboard (`GET /admin/warehouse-dashboard`) [Auth + Admin role]

## Caching
- Dashboard cache in Redis
- Product search cache in Redis
- Cache invalidation on data changes

## Audit and Tracking
- Audit logs for create/update/delete/transfer flows
- Stock logs for stock changes

## Worker Jobs
- `LOW_STOCK_EMAIL`
- `WEBHOOK_SEND`

## Environment Variables

Create a `.env` file in project root with values like:

```env
DB_URL=postgres://user:pass@localhost:5432/dbname
JWT_SECRET=your_jwt_secret

SMTP_EMAIL=you@example.com
SMTP_PASSWORD=your_password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
FROM_EMAIL=you@example.com
ADMIN_EMAIL=admin@example.com
```

Redis defaults in code:
- Host: `localhost:6379`
- DB: `0`

## Run the Project

```bash
go mod tidy
go run .
```

Server starts at:
- `http://localhost:8080`

## Build / Validation

```bash
go test ./...
```

## Notes

- This project currently has no Go unit test files (`[no test files]`), but the full package build passes.
- If you add tests later, keep controller tests focused on request/response and model tests focused on transaction logic.

## Suggested Next Improvements

- Add request body validation for all handlers.
- Add table-driven unit tests and integration tests.
- Add refresh-token persistence/rotation and logout endpoint routing.
- Add structured logging and request IDs.
- Add OpenAPI/Swagger documentation.
