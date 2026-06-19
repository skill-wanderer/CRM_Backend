# CRM Backend

A backend service for CRM functionality built with **Go**, using **Gin** for HTTP APIs and **GORM** with **PostgreSQL** for data persistence.

## Tech Stack

- **Language:** Go
- **Web Framework:** Gin (`github.com/gin-gonic/gin`)
- **ORM:** GORM (`gorm.io/gorm`)
- **Database Driver:** PostgreSQL (`gorm.io/driver/postgres`)
- **Auth:** Keycloak / OIDC with admin and client realms
- **Environment Management:** godotenv (`github.com/joho/godotenv`)
- **Containerization:** Docker, Docker Compose

## Project Structure

```text
CRM_Backend/
├── cmd/
│   └── server/            # Application entrypoint
├── internal/
│   ├── admin/             # Admin-domain tenant APIs
│   ├── auth/              # Keycloak/OIDC verifier and claims
│   ├── client/            # Client-domain tenancy foundation
│   ├── config/            # Typed environment configuration
│   ├── database/          # DB connection and setup
│   ├── handlers/          # HTTP handlers/controllers
│   ├── middleware/        # Auth, RBAC, user sync, tenant scope, CORS
│   ├── models/            # Domain/data models
│   ├── repositories/      # Data access layer
│   ├── routes/            # Route registration
│   ├── services/          # Business logic
│   ├── tenancy/           # Tenant context helpers
│   └── utils/             # Utility helpers
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── test.http              # Sample API requests
└── .env                   # Environment variables (local)
```

## Prerequisites

- Go (use a stable version supported by your environment)
- Docker and Docker Compose (optional, for containerized setup)
- PostgreSQL (if running without Docker)

## Environment Variables

Create/update your `.env` file in the project root.

See [`.env.example`](./.env.example) for the full, commented list. Common variables:

- `PORT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- `KEYCLOAK_BASE_URL`, `KEYCLOAK_ADMIN_REALM`, `KEYCLOAK_CLIENT_REALM`
- `KEYCLOAK_ADMIN_REQUIRED_ROLE`
- `TENANT_HEADER`

> Use strong secrets and never commit production credentials.

## Run Locally

1. Install dependencies:

```bash
go mod tidy
```

2. Start the server:

```bash
go run ./cmd/server
```

Or if your entrypoint file is `main.go` under `cmd/server`:

```bash
go run ./cmd/server/main.go
```

## Run with Docker

Build and start services:

```bash
docker compose up --build
```

Stop services:

```bash
docker compose down
```

## API Testing

This repository includes a `test.http` file with request examples.

You can use:
- VS Code REST Client extension, or
- IntelliJ HTTP Client

to execute requests directly.

## Tenant Management

Tenant management is implemented under `/api/admin/tenants`.

All tenant routes require an admin-realm Keycloak token with the configured
realm role, `CRM` by default:

```http
Authorization: Bearer <admin-realm-token>
```

See [docs/tenant-management.md](./docs/tenant-management.md) for the API,
database model, middleware behavior, and local validation notes.

## Dependency Management

- Add/update deps: `go get <module>`
- Clean module graph: `go mod tidy`

## Recommended Improvements

- Add structured logging and request tracing
- Add integration tests against PostgreSQL and a test Keycloak realm
- Add CI checks (`go test`, `go vet`, `golangci-lint`)

## License

Licensed under the terms of the [Apache-2.0](./LICENSE) license.
