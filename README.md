# Go CRM Learning API

This project is your step-by-step Go REST API learning journey.

## Learning goals

- Build auth flows: login, forgot password, change password.
- Build Users CRUD.
- Build Customers CRUD.
- Build Transactions CRUD linked to customers.
- Learn Go project structure, clean code layers, and practical API design.

## Folder structure

- `cmd/server`: app entrypoint.
- `internal/config`: environment and config loading.
- `internal/platform/database`: DB connection and query setup.
- `internal/http`: router, middleware, and handlers.
- `internal/auth`: auth use-cases (login/password flows/JWT).
- `internal/users`: users module (model/repo/service/handler).
- `internal/crm/customers`: customers module (model/repo/service/handler).
- `internal/crm/transactions`: transactions module linked with customers.
- `migrations`: SQL schema and migration scripts.
- `scripts`: helper scripts (run, migrate, seed).

## Step 1 (today)

1. Create project folder and Go module.
2. Set learning rules in Cursor.
3. Confirm structure and explain why each folder exists.

## Next step

Step 2 will set up:
- HTTP router (`chi`)
- health check endpoint
- env config
- PostgreSQL connection

## Step 2 run commands

1. Copy env values:
   - Windows PowerShell:
     - `Copy-Item .env.example .env`
2. Update `DB_URL` in `.env` to your local PostgreSQL credentials.
3. Export env vars in PowerShell for current terminal:
   - `$env:APP_ENV="development"`
   - `$env:HTTP_PORT="8080"`
   - `$env:DB_URL="postgres://postgres:postgres@localhost:5432/go_crm_learning_api?sslmode=disable"`
4. Run:
   - `go mod tidy`
   - `go run ./cmd/server`
5. Test health:
   - `Invoke-WebRequest http://localhost:8080/health | Select-Object -Expand Content`
