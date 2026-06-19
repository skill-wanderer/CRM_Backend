# Tenant Management and Multi-Tenancy Foundation

This document describes the implementation of
[Spec 0001](specs/0001-tenant-management.md).

## What Is Implemented

- Shared-schema multi-tenancy foundation using a global `tenants` table and a
  `user_tenants` membership table.
- Keycloak/OIDC bearer-token validation for two realms:
  - admin realm for back-office APIs.
  - client realm for future tenant-facing APIs.
- Admin Tenant CRUD at `/api/admin/tenants`.
- Realm-role authorization for tenant administration using
  `realm_access.roles` and the configured `KEYCLOAK_ADMIN_REQUIRED_ROLE`
  value, defaulting to `CRM`.
- JIT client-user sync from client-realm tokens into the `users` table.
- Client tenant scoping middleware that reads the configured tenant header
  (`X-Tenant-ID` by default), checks tenant status, verifies membership, and
  stores the tenant ID in request context.

Client-domain business endpoints and membership-management endpoints are still
future work, as defined by Spec 0001.

## Package Layout

```text
internal/
├── admin/
│   ├── handlers/        # Tenant HTTP handlers
│   ├── repositories/    # Tenant persistence
│   ├── services/        # Tenant validation and business rules
│   └── routes.go        # /api/admin/tenants route registration
├── auth/                # OIDC claims and realm verifier
├── client/              # Client-domain middleware chain registration
├── config/              # Typed environment config
├── middleware/          # Auth, RBAC, user sync, tenant scope, CORS
├── models/              # Tenant, User, UserTenant, existing placeholders
└── tenancy/             # Tenant context helpers
```

## Configuration

Configuration is loaded once at startup from environment variables. In local
development, `.env` is loaded automatically when present.

Required values:

- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `KEYCLOAK_BASE_URL`
- `KEYCLOAK_ADMIN_REALM`
- `KEYCLOAK_CLIENT_REALM`

Common optional values:

- `KEYCLOAK_ADMIN_REQUIRED_ROLE` defaults to `CRM`.
- `KEYCLOAK_ADMIN_AUDIENCE` and `KEYCLOAK_CLIENT_AUDIENCE` are blank by
  default, which skips `aud`/`azp` checks.
- `KEYCLOAK_ADMIN_ISSUER` and `KEYCLOAK_CLIENT_ISSUER` default to
  `{KEYCLOAK_BASE_URL}/realms/{REALM}`.
- `TENANT_HEADER` defaults to `X-Tenant-ID`.
- `DB_AUTO_MIGRATE` defaults to `true`.

See [../.env.example](../.env.example) for the full list.

## Database

Startup migration uses GORM AutoMigrate and ensures the PostgreSQL `pgcrypto`
extension exists for `gen_random_uuid()`.

New tables:

- `tenants`
  - UUID primary key.
  - soft delete via `deleted_at`.
  - live-row unique slug index.
  - status values: `active`, `suspended`.
- `users`
  - UUID primary key.
  - unique `keycloak_sub`.
  - synced `email`, `name`, and `last_login_at`.
- `user_tenants`
  - composite primary key `(user_id, tenant_id)`.
  - reverse lookup index on `tenant_id`.

Existing lead/template models remain placeholders until the client-domain CRM
spec rebuilds them with tenant ownership.

## Admin Tenant API

All routes require:

```http
Authorization: Bearer <admin-realm-token-with-CRM-role>
```

Base path:

```text
/api/admin/tenants
```

Routes:

```text
POST   /api/admin/tenants
GET    /api/admin/tenants?page=1&pageSize=20&status=active&q=acme
GET    /api/admin/tenants/:id
PUT    /api/admin/tenants/:id
DELETE /api/admin/tenants/:id
```

Create body:

```json
{
  "name": "Acme Corp",
  "slug": "acme",
  "description": "Pilot customer"
}
```

Update body:

```json
{
  "name": "Acme Corporation",
  "description": "Upgraded account",
  "status": "suspended"
}
```

Rules:

- `name` is required and limited to 120 characters.
- `slug` is optional on create; when omitted, it is derived from `name`.
- `slug` is immutable after create.
- `description` is optional and limited to 1000 characters.
- `status` must be `active` or `suspended`.
- Unknown JSON fields are rejected.
- Delete is a soft delete.

Errors use this envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "name is required"
  }
}
```

Important status mappings:

- `401 UNAUTHENTICATED`: missing/invalid/wrong-realm token.
- `403 FORBIDDEN`: valid admin token missing the CRM role.
- `400 VALIDATION_ERROR`: invalid JSON/body/query/path input.
- `404 NOT_FOUND`: tenant does not exist.
- `409 CONFLICT`: duplicate live slug.

## Client Tenant Scope Foundation

The client-domain middleware chain is registered for `/api/client`:

```text
Auth(client realm) -> UserSync -> TenantScope
```

`UserSync` creates or updates a row in `users` from token claims:

- `sub` -> `keycloak_sub`
- `email` -> `email`
- `name` or `preferred_username` -> `name`
- current time -> `last_login_at`

`TenantScope` requires:

```http
Authorization: Bearer <client-realm-token>
X-Tenant-ID: <tenant-uuid>
```

It rejects missing or malformed tenant headers with `400`, unavailable or
suspended tenants with `403`, and non-members with `403`.

No client business endpoints are implemented yet.

## Local Checks

Run:

```bash
go test ./...
```

The middleware tests cover:

- admin realm-role enforcement.
- JIT user create/update behavior.
- tenant header and membership checks.
