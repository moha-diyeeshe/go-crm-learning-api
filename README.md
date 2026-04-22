# Go CRM Learning API

This project is your step-by-step Go REST API learning journey.

## Learning goals

- Build auth flows: login, forgot password, change password.
- Build Users CRUD.
- Build Customers CRUD.
- Build Transactions CRUD linked to customers.
- Learn Go project structure, clean code layers, and practical API design.

## Rebuild guide (from scratch)

Follow this exact order when rebuilding the project by yourself.

### 1) Folder structure

Create this structure first:

- `cmd/server`
- `internal/config`
- `internal/platform/database`
- `internal/http`
- `internal/auth`
- `internal/users`
- `internal/customers`
- `http-tests`

### 2) Server setup

In `cmd/server/main.go`, do only app wiring:

- load config
- create DB pool
- create dependencies (`repo -> service -> handler`)
- create router
- start HTTP server

### 3) Database setup

In `internal/platform/database`, create DB pool + ping.

In each module repository:

- add `EnsureSchema(ctx)` to create table if missing

### 4) Users CRUD (layer pattern)

Build users in this order:

1. `internal/users/model.go`
2. `internal/users/repository.go`
3. `internal/users/service.go`
4. `internal/users/handler.go`

### 5) Login with email/password

Add endpoint:

- `POST /api/v1/users/login`

Body:

```json
{
  "email": "user@example.com",
  "password": "Pass@1234"
}
```

### 6) JWT + refresh tokens

Use `internal/auth/jwt.go`:

- generate `access_token`
- generate `refresh_token`
- validate access token for protected routes
- refresh endpoint to rotate token pair

Add endpoint:

- `POST /api/v1/users/token/refresh`

Body:

```json
{
  "refresh_token": "..."
}
```

### 7) Customers CRUD

Build customers with the same layer pattern:

1. `internal/customers/model.go`
2. `internal/customers/repository.go`
3. `internal/customers/service.go`
4. `internal/customers/handler.go`

### 8) Protect CRUD routes (logged-in users only)

Add JWT auth middleware in `internal/http` and apply it to route groups:

- protect users CRUD routes
- protect all customers CRUD routes
- keep login/refresh as public endpoints

## Environment variables

Copy `.env.example` to `.env` and set values:

- `APP_ENV=development`
- `HTTP_PORT=8080`
- `DB_URL=postgres://...`
- `JWT_SECRET=your-strong-secret`
- `JWT_TTL_MINUTES=60`
- `JWT_REFRESH_TTL_HOURS=168`

## How to run (PowerShell)

```powershell
Copy-Item .env.example .env
go mod tidy
go run .\cmd\server\main.go
```

Health check:

```powershell
Invoke-WebRequest http://localhost:8080/health | Select-Object -Expand Content
```

## Manual API test flow

Use files in `http-tests`:

1. `users.http`:
   - create user
   - login to get `access_token` + `refresh_token`
   - call refresh endpoint
   - call protected users endpoints with bearer access token
2. `customers.http`:
   - call all customer CRUD endpoints with bearer access token

Header format for protected endpoints:

```http
Authorization: Bearer <access_token>
```

## Done checklist

- [ ] project structure created
- [ ] config + DB connection working
- [ ] users CRUD working
- [ ] login returns access + refresh tokens
- [ ] refresh endpoint returns rotated token pair
- [ ] customers CRUD working
- [ ] protected endpoints require bearer access token
