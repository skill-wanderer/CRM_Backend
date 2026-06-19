# Spec 0001 — Tenant Management & Multi-Tenancy Foundation

| | |
|---|---|
| **Status** | Published |
| **Author** | Quan Nguyen |
| **Created** | 2026-06-19 |
| **Published** | 2026-06-19 |
| **Feature area** | `admin` domain — Tenant CRUD + multi-tenancy foundation |

---

## 1. Summary

Introduce **multi-tenancy** to the CRM backend and the first feature that exercises it:
**Tenant CRUD**.

The system is split into two API **domains**:

- **`admin`** — back-office APIs used by CRM operators. Authenticated via the
  Keycloak **admin realm**. Tenant CRUD lives here and requires the realm role
  **`CRM`**.
- **`client`** — tenant-facing CRM APIs used by customers (leads, templates,
  etc., to be migrated/added later). Authenticated via the Keycloak
  **client realm**. Tenant access is granted by the CRM through a
  **many-to-many `User ↔ Tenant`** mapping (not by the token), and the active
  tenant is selected per request via the `X-Tenant-ID` header.

This spec covers the **tenant + user + membership data model**, **isolation
strategy**, **authentication/authorization** (including JIT client-user
provisioning and membership-based tenant access), **the admin Tenant CRUD API**,
and the **domain/package layout**. It does **not** implement client-domain CRM
endpoints or membership-management endpoints — those are follow-up specs.

---

## 2. Goals

- Define `Tenant`, `User`, and `UserTenant` (membership) entities in PostgreSQL.
- Model **`User ↔ Tenant` as many-to-many** (one user → many tenants; one
  tenant → many users), with access owned and enforced by the CRM.
- **JIT-provision** client `users` from the validated client-realm token.
- Provide admin-domain CRUD endpoints for tenants.
- Restrict tenant CRUD to users in the **admin realm** holding the **`CRM`**
  realm role.
- Establish the **shared-schema + `tenant_id`** isolation pattern that all
  future client-domain tenant-owned tables will follow.
- Establish the **two-realm Keycloak** authentication model and a reusable
  middleware layer (token validation via OIDC/JWKS, realm checks, role checks,
  JIT user sync, header-based tenant selection + membership check).
- Establish the **`admin` vs `client` domain** package structure.

## 3. Non-Goals

- No Keycloak provisioning side effects on tenant create (tenant CRUD is a
  **DB record only** — see §6.6). Keycloak realms/users/clients are managed
  out-of-band for now.
- No client-domain CRM endpoints (leads/templates migration) — separate spec.
- No membership-management endpoints (grant/revoke a user's tenant access) —
  the schema + read path are defined here; the admin API for it is a follow-up.
- No per-tenant roles/permissions on memberships (plain membership only).
- No per-tenant schema or per-tenant database. (Decision: shared schema.)
- No tenant self-service signup / onboarding UI.
- No billing, quotas, or usage metering.

---

## 4. Decisions (resolved)

| Topic | Decision |
|---|---|
| **Data isolation** | Shared database, shared schema, `tenant_id` discriminator column on every tenant-owned table. App-layer enforcement via middleware; Postgres Row-Level Security (RLS) is an optional later hardening. |
| **Keycloak realms** | Exactly **2 realms**: `admin` realm (CRM operators) and a single shared `client` realm (all tenant users). The token does **not** carry a tenant claim — Keycloak only proves identity. |
| **Tenant access model** | The **CRM owns tenant access**, not Keycloak. A **many-to-many** `User ↔ Tenant` relationship (join table) determines which tenants a client user may access. One user → many tenants; one tenant → many users. |
| **Client `User` table** | A `users` table mirrors client-realm identities. On each client request the user is **provisioned/synced just-in-time (JIT)**: if no row exists for the token's `sub`, insert one; if it exists, update `email`/`name`/`last_login_at` from the token so CRM data stays in sync with Keycloak. |
| **Tenant selection** | A client request names the target tenant via the **`X-Tenant-ID` header** (tenant UUID). The backend authorizes by checking the caller's membership in that tenant. |
| **Tenant provisioning** | Creating a tenant writes a **DB row only**. No Keycloak Admin API calls in this iteration. |
| **`CRM` role** | A **realm role** in the admin realm, read from `realm_access.roles`. |
| **Database** | PostgreSQL (existing GORM setup). |

---

## 5. Architecture & Domain Layout

### 5.1 Domain separation

Two top-level domains under `internal/`. Shared building blocks
(auth, tenancy, database, config) stay in cross-cutting packages.

```text
internal/
├── admin/                  # ADMIN DOMAIN (admin realm, back-office)
│   ├── handlers/           # e.g. tenant_handler.go
│   ├── services/           # e.g. tenant_service.go
│   ├── repositories/       # e.g. tenant_repo.go
│   └── routes.go           # registers /api/admin/... routes
│
├── client/                 # CLIENT DOMAIN (client realm, tenant-scoped)
│   ├── handlers/           # leads/templates (future specs)
│   ├── services/
│   ├── repositories/
│   └── routes.go           # registers /api/client/... routes
│
├── auth/                   # Keycloak/OIDC: token validation, JWKS cache, claims
│   ├── verifier.go         # validates RS256 tokens against realm JWKS
│   └── claims.go           # parsed claims (sub, email, name, realm_access.roles)
│
├── tenancy/                # tenant context helpers (resolve/set/get tenant_id)
│   └── context.go
│
├── middleware/             # gin middleware (see §7)
│   ├── auth.go             # realm-aware bearer auth
│   ├── rbac.go             # realm-role check (e.g. requires "CRM")
│   ├── user_sync.go        # JIT upsert of client user from token (keeps in sync)
│   └── tenant_scope.go     # reads X-Tenant-ID, checks membership, sets context
│
├── models/                 # shared GORM models (Tenant added here)
├── database/               # connection + migrations
├── config/                 # env config (realms, issuer URLs, etc.)
└── utils/
```

> The existing local-JWT `AuthMiddleware`/`RBACMiddleware` and the password-based
> `User` model/auth flow are **removed** and fully replaced by Keycloak. All
> authentication is OIDC against the two realms — see §12.

### 5.2 Request routing

```text
/api
├── /admin                  # admin realm token required
│   └── /tenants            # requires realm role "CRM"
│       ├── POST   ""        create
│       ├── GET    ""        list (paginated)
│       ├── GET    "/:id"    get one
│       ├── PUT    "/:id"    update
│       └── DELETE "/:id"    delete (soft)
│
└── /client                 # client realm token required; tenant-scoped (future)
    └── ...                  # leads, templates — later specs
```

### 5.3 Configuration (environment-driven)

**Principle: nothing environment-specific is hard-coded.** Everything that can
differ between local / staging / prod — ports, DB connection, Keycloak base URL,
realm names, role name, the tenant header, cache TTLs — is read from environment
variables (loaded from `.env` via `godotenv` in dev, real env vars in
containers/CI). This keeps a single binary deployable anywhere by config alone.

Conventions:

- Config is parsed **once at startup** into a typed `config.Config` struct
  (`internal/config`); the rest of the code depends on that struct, not on
  `os.Getenv` scattered around.
- **Fail fast:** required vars missing/invalid → the process exits at boot with a
  clear error (don't start half-configured).
- Sensible **defaults** for non-secret knobs (e.g. `PORT=8080`,
  `JWKS_CACHE_TTL=15m`); secrets have no default.
- Issuer URLs are **derived** (`{KEYCLOAK_BASE_URL}/realms/{REALM}`) unless
  explicitly overridden, so most deployments set only the base URL + realm names.
- A committed **`.env.example`** documents every variable; the real **`.env`** is
  git-ignored (secrets never committed).

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port. |
| `GIN_MODE` | `debug` | `debug` \| `release` \| `test`. |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed origins. |
| `DB_HOST` / `DB_PORT` | `localhost` / `5432` | PostgreSQL location. |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | — (required) | PostgreSQL credentials/db. |
| `DB_SSLMODE` | `disable` | `disable`\|`require`\|`verify-ca`\|`verify-full`. |
| `DB_TIMEZONE` | `UTC` | Session timezone. |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME` | `25` / `5` / `1h` | Connection-pool tuning. |
| `DB_AUTO_MIGRATE` | `true` | Run GORM AutoMigrate on startup. |
| `KEYCLOAK_BASE_URL` | — (required) | Keycloak server base URL. |
| `KEYCLOAK_ADMIN_REALM` | — (required) | Admin realm name. |
| `KEYCLOAK_CLIENT_REALM` | — (required) | Client realm name. |
| `KEYCLOAK_ADMIN_REQUIRED_ROLE` | `CRM` | Realm role required for tenant admin. |
| `KEYCLOAK_ADMIN_AUDIENCE` / `KEYCLOAK_CLIENT_AUDIENCE` | _(blank = skip)_ | Expected `aud`/`azp`. |
| `KEYCLOAK_ADMIN_ISSUER` / `KEYCLOAK_CLIENT_ISSUER` | _(derived)_ | Issuer override (proxy/custom host). |
| `JWKS_CACHE_TTL` | `15m` | How long realm signing keys are cached. |
| `TOKEN_CLOCK_SKEW` | `1m` | Allowed skew on `exp`/`nbf`. |
| `TENANT_HEADER` | `X-Tenant-ID` | Header naming the active tenant (client domain). |

See [`.env.example`](../../.env.example) for the authoritative, commented list.

---

## 6. Data Model

### 6.1 `Tenant` entity

GORM model (added to `internal/models`):

```go
type Tenant struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    Name        string         `gorm:"not null" json:"name"`
    Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
    Description  string         `gorm:"type:text" json:"description"`
    Status      TenantStatus   `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
    CreatedAt   time.Time      `json:"createdAt"`
    UpdatedAt   time.Time      `json:"updatedAt"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // soft delete
}

type TenantStatus string

const (
    TenantStatusActive    TenantStatus = "active"
    TenantStatusSuspended TenantStatus = "suspended"
)
```

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key. UUID (not auto-inc) so tenant IDs are stable, non-guessable, and safe to embed in tokens. |
| `name` | string | Human-readable display name. Required. |
| `slug` | string | URL/identifier-safe unique key (e.g. `acme`). Lowercase, `[a-z0-9-]`, unique. Human-readable stable reference (logs, admin UI, future URLs). |
| `description` | text | Optional free-text notes about the tenant (purpose, owner, internal context). Not used for any logic. |
| `status` | enum | `active` \| `suspended`. Suspended tenants are rejected at client-domain auth. |
| `created_at` / `updated_at` | timestamp | Managed by GORM. |
| `deleted_at` | timestamp (nullable) | Soft delete; preserves historical data and avoids orphaning tenant-owned rows. |

### 6.2 `User` entity (client-realm identities)

Mirrors a Keycloak **client-realm** user. A record is created just-in-time on
first appearance of the token and kept in sync with the token's claims on later
requests (§7.3) — the CRM never stores passwords.

```go
type User struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    KeycloakSub string         `gorm:"uniqueIndex;not null" json:"-"`   // token `sub`
    Email       string         `gorm:"index" json:"email"`
    Name        string         `json:"name"`
    LastLoginAt *time.Time     `json:"lastLoginAt,omitempty"`
    CreatedAt   time.Time      `json:"createdAt"`
    UpdatedAt   time.Time      `json:"updatedAt"`

    // Association (many-to-many via user_tenants)
    Tenants     []Tenant       `gorm:"many2many:user_tenants;" json:"tenants,omitempty"`
}
```

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Internal primary key. |
| `keycloak_sub` | string | The token `sub` claim. Unique. The link to the Keycloak identity and the upsert key. |
| `email` | string | Synced from the token on each request. |
| `name` | string | Display name synced from token (`name`/`preferred_username`). |
| `last_login_at` | timestamp (nullable) | Refreshed to now on every authenticated request. |

> Admin-realm operators are **not** stored here; admin authz relies solely on the
> `CRM` realm role in the token (§7.2).

### 6.3 `UserTenant` membership (join table)

The many-to-many link that the CRM uses to authorize tenant access. One user can
belong to many tenants; one tenant can have many users.

```go
type UserTenant struct {
    UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"userId"`
    TenantID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"tenantId"`
    CreatedAt time.Time `json:"createdAt"`
}
```

| Field | Type | Notes |
|---|---|---|
| `user_id` | UUID | FK → `users.id`. Part of composite PK. |
| `tenant_id` | UUID | FK → `tenants.id`. Part of composite PK. |
| `created_at` | timestamp | When the membership was granted. |

- Composite primary key `(user_id, tenant_id)` makes a membership unique and
  idempotent to grant.
- **No per-tenant role in this iteration** (decision §4). A `role` column can be
  added later without breaking the PK.
- Managing memberships (grant/revoke) is **CRM-owned**; the admin endpoints for
  it are a follow-up (see §14). This spec defines the schema + the read path
  used during client auth.

### 6.4 Isolation pattern for tenant-owned tables

Every **client-domain** (tenant-owned) table MUST include:

```go
TenantID uuid.UUID `gorm:"type:uuid;index;not null" json:"-"`
```

- All client-domain queries are filtered by the resolved `tenant_id` from the
  request context (§7.3). This filtering is centralized so individual handlers
  cannot forget it.
- Future hardening (optional): Postgres **RLS** policies keyed on a session
  variable set per request (`SET app.current_tenant = ...`).
- The `Tenant` table itself is **global** (admin-owned) and has no `tenant_id`.

### 6.5 Migrations

- Use GORM `AutoMigrate` for `Tenant`, `User`, `UserTenant` (consistent with
  current codebase), OR introduce versioned SQL migrations. **OQ-2** in §11.
- Requires the `pgcrypto` (or `uuid-ossp`) extension for `gen_random_uuid()`.
  Migration must `CREATE EXTENSION IF NOT EXISTS pgcrypto;`.

### 6.6 Provisioning behavior

Creating a tenant performs **only** a DB insert. No Keycloak Admin API calls.
Client `users` rows are created just-in-time from the token (§7.3); membership
in `user_tenants` is granted by CRM admins (endpoints are a follow-up, §14).

### 6.7 Database design

#### 6.7.1 Entity-relationship overview

```text
   ┌──────────────────────────┐                ┌─────────────────────────────┐
   │           users          │                │           tenants           │
   │   (client-realm mirror)  │                │     (global / admin-owned)  │
   ├──────────────────────────┤                ├─────────────────────────────┤
   │ id            uuid PK     │                │ id          uuid  PK         │
   │ keycloak_sub  varchar UQ  │                │ name        varchar(120)     │
   │ email         varchar     │                │ slug        varchar(63) UQ   │
   │ name          varchar     │                │ description text             │
   │ last_login_at timestamptz │                │ status      varchar(20)      │
   │ created_at    timestamptz │                │ created_at  timestamptz      │
   │ updated_at    timestamptz │                │ updated_at  timestamptz      │
   └────────────┬─────────────┘                │ deleted_at  timestamptz NULL │
                │ 1                             └──────────────┬──────────────┘
                │                                              │ 1
                │ N         ┌────────────────────────┐      N │
                └──────────►│      user_tenants      │◄───────┘
                            │     (M:N join table)   │
                            ├────────────────────────┤
                            │ user_id   uuid PK,FK    │
                            │ tenant_id uuid PK,FK    │
                            │ created_at timestamptz  │
                            └────────────────────────┘
                                          │ 1 (tenant)
                                          │
                  ┌───────────────────────┼───────────────────────┐
                  │ N                      │ N                     │ N
        ┌─────────▼────────┐    ┌──────────▼───────┐    ┌──────────▼───────┐
        │  lead_templates  │    │      leads       │    │   <future...>    │
        │  (tenant-owned)  │    │  (tenant-owned)  │    │  (tenant-owned)  │
        ├──────────────────┤    ├──────────────────┤    ├──────────────────┤
        │ id        PK     │    │ id        PK     │    │ ...              │
        │ tenant_id FK ────┼────┼ tenant_id FK ────┼────┼ tenant_id FK     │
        │ ...              │    │ ...              │    │ ...              │
        └──────────────────┘    └──────────────────┘    └──────────────────┘
```

- `users` ↔ `tenants` is **many-to-many** via `user_tenants`.
- `tenants` is **global** (no `tenant_id`); `users` is also global (a user can
  span tenants).
- Every tenant-owned table (`lead_templates`/`leads`/future) carries
  `tenant_id uuid NOT NULL` → `tenants(id)`. Those tables are migrated in a
  later spec, shown here only for context.

#### 6.7.2 `tenants` table DDL

```sql
-- Required for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar(120) NOT NULL,
    slug        varchar(63)  NOT NULL,
    description text         NULL,
    status      varchar(20)  NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'suspended')),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    deleted_at  timestamptz  NULL
);

-- Slug is unique only among live (non-soft-deleted) tenants, so a slug can be
-- reused after a tenant is deleted.
CREATE UNIQUE INDEX ux_tenants_slug_live
    ON tenants (slug)
    WHERE deleted_at IS NULL;

-- Common list/filter access paths.
CREATE INDEX ix_tenants_status     ON tenants (status) WHERE deleted_at IS NULL;
CREATE INDEX ix_tenants_deleted_at ON tenants (deleted_at);
```

> **Note on uniqueness vs soft delete:** a plain `UNIQUE` constraint on `slug`
> would prevent reusing a slug after deletion. The **partial** unique index
> (`WHERE deleted_at IS NULL`) enforces uniqueness only across active rows. If
> slug reuse should be forbidden forever, replace it with a plain
> `UNIQUE (slug)` — see **OQ-7** (§11).

#### 6.7.3 `users` table DDL

```sql
CREATE TABLE users (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_sub  varchar(255) NOT NULL,
    email         varchar(255) NULL,
    name          varchar(255) NULL,
    last_login_at timestamptz  NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);

-- Upsert key: one CRM user row per Keycloak identity.
CREATE UNIQUE INDEX ux_users_keycloak_sub ON users (keycloak_sub);
CREATE INDEX        ix_users_email        ON users (email);
```

#### 6.7.4 `user_tenants` join table DDL

```sql
CREATE TABLE user_tenants (
    user_id    uuid        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    tenant_id  uuid        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, tenant_id)
);

-- Reverse lookup: "which users belong to this tenant?"
CREATE INDEX ix_user_tenants_tenant_id ON user_tenants (tenant_id);
```

- Composite PK `(user_id, tenant_id)` guarantees one membership per pair and
  makes grants idempotent. The PK index already serves "tenants for a user"; the
  extra index serves "users for a tenant".
- `ON DELETE CASCADE` here removes the **membership** when either side is hard
  deleted; it does not delete the user or tenant. (Tenants are soft-deleted in
  practice, so cascade rarely fires.)

#### 6.7.5 Pattern for tenant-owned tables (reference)

Applied when client-domain tables are introduced (future spec):

```sql
CREATE TABLE <tenant_owned_table> (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    -- ... domain columns ...
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Every tenant-scoped query filters by tenant_id; index it first.
CREATE INDEX ix_<table>_tenant_id ON <tenant_owned_table> (tenant_id);

-- Composite indexes for tenant-scoped lookups put tenant_id first, e.g.:
-- CREATE INDEX ix_<table>_tenant_created ON <tenant_owned_table> (tenant_id, created_at DESC);
```

Conventions:

- `tenant_id` is the **leading column** of every secondary index on a
  tenant-owned table, so all tenant-scoped queries are index-covered.
- FK uses `ON DELETE RESTRICT` to align with soft-delete semantics (a tenant
  with live data cannot be hard-deleted out from under its rows). See **OQ-3**.
- Unique constraints on tenant-owned tables must be **scoped by tenant**, e.g.
  `UNIQUE (tenant_id, name)`, never a global `UNIQUE (name)`.

#### 6.7.6 Type & convention notes

- **Timestamps:** `timestamptz` (UTC) everywhere. GORM `time.Time` maps to this.
- **Primary keys:** `uuid` via `gen_random_uuid()` (pgcrypto). Non-enumerable
  and safe to embed in tokens/URLs.
- **Enums:** modeled as `varchar` + `CHECK` rather than native Postgres `ENUM`
  types, to keep additive changes migration-friendly.
- **Soft delete:** `deleted_at timestamptz NULL`; GORM `gorm.DeletedAt` filters
  it automatically. All "live" indexes are partial (`WHERE deleted_at IS NULL`).

---

## 7. Authentication & Authorization

### 7.1 Keycloak / OIDC model

- Tokens are **RS256 JWTs** issued by Keycloak, validated against each realm's
  **JWKS** (fetched from the realm's OIDC discovery document and cached with
  periodic refresh). No shared HMAC secret.
- Validation checks: signature, `exp`/`nbf`, `iss` (must match the expected
  realm issuer for the domain), and `aud`/`azp` as configured.

All Keycloak settings are **environment-driven** (realm names, base URL, issuer
overrides, audiences, JWKS TTL, clock skew) — see the full table in **§5.3** and
[`.env.example`](../../.env.example). Nothing realm- or host-specific is
hard-coded, so the same binary points at any Keycloak by config alone.

### 7.2 Admin domain authz

1. `middleware.Auth(adminRealm)` — validates the token was issued by the
   **admin** realm issuer.
2. `middleware.RequireRealmRole("CRM")` — checks `realm_access.roles` contains
   `CRM`. Returns `403` otherwise.

Both are applied to the entire `/api/admin/tenants` group.

### 7.3 Client domain authz (foundation only)

The token proves **identity only** — it carries no tenant. Tenant access is
decided by the CRM via the `user_tenants` membership table. The middleware chain:

1. `middleware.Auth(clientRealm)` — validates the token came from the **client**
   realm (signature, issuer, expiry).
2. `middleware.UserSync()` — **upsert + keep in sync**. Match the `users` row by
   `keycloak_sub` (the token `sub`):
   - **not found** → insert a new row from the token claims (`sub`, `email`,
     `name`); set `last_login_at`/`created_at` to now.
   - **found** → **update** `email`/`name` if the token now carries different
     values (so CRM data stays in sync with Keycloak), and refresh
     `last_login_at`. Unchanged values are written back harmlessly.

   Either way, put the internal `user_id` in the request context for downstream
   handlers. First-time callers are created automatically (JIT provisioning).

   This is a single race-safe upsert keyed on `keycloak_sub`:

   ```sql
   INSERT INTO users (keycloak_sub, email, name, last_login_at)
   VALUES ($sub, $email, $name, now())
   ON CONFLICT (keycloak_sub) DO UPDATE
   SET email         = EXCLUDED.email,
       name          = EXCLUDED.name,
       last_login_at = now(),
       updated_at    = now()
   RETURNING id;
   ```

   > `email`/`name`/`last_login_at` always reflect the **latest token**, so a
   > change made in Keycloak propagates to the CRM on the user's next request.
   > `keycloak_sub` is immutable and is never changed by the update.
3. `middleware.TenantScope()`:
   - reads the **`X-Tenant-ID`** header (tenant UUID); missing/malformed → `400`.
   - loads the tenant; not found or not `active` → `403`.
   - checks a `user_tenants` row exists for `(user_id, tenant_id)`; if not, the
     caller is not a member → `403`.
   - stores the resolved `tenant_id` in the request context for downstream
     repositories.

```text
Client request:
  Authorization: Bearer <client-realm token>
  X-Tenant-ID: 7c9e6679-7425-40de-944b-e07fc1f90ae7

  Auth ──► UserSync (upsert user) ──► TenantScope (membership check) ──► handler
```

(No client business endpoints are wired in this spec; the middleware is defined
and unit-tested. A `GET /api/client/me/tenants` discovery endpoint is noted as a
follow-up in §14.)

### 7.4 Error responses

Consistent JSON envelope:

```json
{ "error": { "code": "FORBIDDEN", "message": "missing required role: CRM" } }
```

| Situation | HTTP | code |
|---|---|---|
| No/blank bearer token | 401 | `UNAUTHENTICATED` |
| Bad signature / expired / wrong issuer | 401 | `UNAUTHENTICATED` |
| Authenticated but missing `CRM` role | 403 | `FORBIDDEN` |
| Client request missing/malformed `X-Tenant-ID` | 400 | `VALIDATION_ERROR` |
| Client token, tenant unknown/suspended | 403 | `FORBIDDEN` |
| Client token, caller not a member of tenant | 403 | `FORBIDDEN` |
| Validation error on body | 400 | `VALIDATION_ERROR` |
| Tenant not found | 404 | `NOT_FOUND` |
| Duplicate slug | 409 | `CONFLICT` |

---

## 8. Admin API — Tenant CRUD

Base path: `/api/admin/tenants`
Auth: admin-realm token **+** realm role `CRM` (all routes).

### 8.1 Create

`POST /api/admin/tenants`

```json
// request
{ "name": "Acme Corp", "slug": "acme", "description": "Pilot customer, EU region" }
```

- `name`: required, 1–120 chars.
- `slug`: optional; if omitted, derived from `name` (lowercased, non-alnum →
  `-`). Must be unique → `409 CONFLICT` on collision.
- `description`: optional free text, ≤1000 chars.
- `status` defaults to `active`.

```json
// 201 response
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "name": "Acme Corp",
  "slug": "acme",
  "description": "Pilot customer, EU region",
  "status": "active",
  "createdAt": "2026-06-19T10:00:00Z",
  "updatedAt": "2026-06-19T10:00:00Z"
}
```

### 8.2 List

`GET /api/admin/tenants?page=1&pageSize=20&status=active&q=acme`

- Pagination: `page` (default 1), `pageSize` (default 20, max 100).
- Filters: `status`, `q` (matches name/slug, case-insensitive).
- Excludes soft-deleted tenants.

```json
// 200 response
{
  "data": [ { "id": "...", "name": "Acme Corp", "slug": "acme", "status": "active", "createdAt": "...", "updatedAt": "..." } ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### 8.3 Get one

`GET /api/admin/tenants/:id` → `200` tenant object, or `404`.

### 8.4 Update

`PUT /api/admin/tenants/:id`

```json
{ "name": "Acme Corporation", "description": "Upgraded to paid plan", "status": "suspended" }
```

- Updatable: `name`, `description`, `status`. **`slug` is immutable** (it
  correlates with issued tokens). `404` if not found; `400` on invalid `status`.

### 8.5 Delete

`DELETE /api/admin/tenants/:id` → `204`. **Soft delete** (sets `deleted_at`).
A suspended/deleted tenant's client users are rejected at client-domain auth.

> **OQ-3:** Should delete be blocked (or cascade) when tenant-owned data exists?
> Spec assumes soft delete only, with no cascade for now.

---

## 9. Validation Rules

- `slug`: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, length 2–63, unique.
- `name`: trimmed, non-empty, ≤120 chars.
- `description`: optional, ≤1000 chars.
- `status`: one of `active`, `suspended`.
- Unknown JSON fields rejected (or ignored — **OQ-4**).

---

## 10. Security Considerations

- **No tenant data crosses tenant boundaries.** All client-domain reads/writes
  are filtered by context `tenant_id`; this is enforced centrally, not per
  handler.
- **Tenant access is authorized server-side.** The `X-Tenant-ID` header is
  *untrusted input*: access is granted only after confirming a `user_tenants`
  membership for the authenticated user. A user passing a tenant they don't
  belong to gets `403` — the header alone grants nothing.
- **Admin realm ≠ client realm.** A client-realm token can never reach admin
  endpoints (issuer mismatch → 401), and vice versa.
- **`CRM` role gate** on all tenant mutations + reads.
- JWKS keys cached with refresh; tolerate Keycloak key rotation.
- Tenant IDs are UUIDs (non-enumerable).
- Validate `iss`/`aud` to prevent token reuse across realms/clients.

---

## 11. Open Questions

| ID | Question | Spec assumption |
|---|---|---|
| OQ-2 | GORM `AutoMigrate` vs versioned SQL migrations? | AutoMigrate (match current code) |
| OQ-3 | On tenant delete, block/cascade when tenant data exists? | Soft delete, no cascade |
| OQ-4 | Reject unknown JSON fields, or ignore them? | Reject |
| OQ-6 | Is the `CRM` realm role name exactly `CRM` (case-sensitive)? | Yes, `CRM` |
| OQ-7 | Can a `slug` be reused after a tenant is soft-deleted? | Yes (partial unique index on live rows) |
| OQ-8 | When a tenant is soft-deleted, also remove its `user_tenants` rows? | Keep rows; access blocked via tenant status check |
| OQ-9 | Which claim is the display name — `name` or `preferred_username`? | Prefer `name`, fall back to `preferred_username` |
| OQ-10 | Refresh `email`/`name`/`last_login_at` on later sign-ins? | Yes — upsert keeps them in sync with the token |

---

## 12. Migration / Impact on Existing Code

- `internal/middleware/auth.go` (HMAC JWT) is **removed** and replaced by
  `auth/` + the realm-aware middleware. Authentication is OIDC-only.
- `internal/models.User` (password login, auto-inc ID, `role` column) is
  **replaced** by the new Keycloak-backed `User` (UUID PK, `keycloak_sub`, no
  password). `auth_service`/`auth_handler` and the `/api/auth`
  register/login routes are **removed**; `JWT_SECRET` is dropped.
- The current `leads`/`templates` models and endpoints are **placeholders**,
  left untouched by this spec. They will be completely rebuilt in the **client**
  domain under **Spec 0002** — not migrated in place.
- New deps likely: `github.com/google/uuid`, an OIDC/JWKS lib
  (e.g. `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`, or
  `github.com/MicahParks/keyfunc` for JWKS).

---

## 13. Acceptance Criteria

- [ ] `Tenant`, `User`, `UserTenant` models migrated into PostgreSQL (UUID PKs,
      unique slug, unique `keycloak_sub`, composite PK on `user_tenants`, soft
      delete on tenants).
- [ ] `POST/GET/GET:id/PUT/DELETE /api/admin/tenants` implemented per §8.
- [ ] All tenant routes reject non-admin-realm tokens (401) and admin-realm
      tokens lacking the `CRM` role (403).
- [ ] Realm-aware token validation against Keycloak JWKS for both realms.
- [ ] `UserSync` middleware upserts the client `users` row from token claims
      (insert if absent; on existing, sync `email`/`name`/`last_login_at`),
      race-safe via `ON CONFLICT (keycloak_sub) DO UPDATE`, and exposes the
      internal `user_id` (unit-tested).
- [ ] `TenantScope` middleware reads `X-Tenant-ID`, rejects missing/malformed
      header (400), unknown/suspended tenant (403), and non-members (403);
      sets `tenant_id` context on success (unit-tested).
- [ ] Duplicate slug → 409; not found → 404; bad body → 400.
- [ ] `admin` and `client` domain packages exist with documented boundaries.

---

## 14. Future Specs (out of scope here)

1. **Membership management** admin endpoints (grant/revoke a user's access to a
   tenant: `POST/DELETE /api/admin/tenants/:id/users`) and listing a tenant's
   users.
2. **`GET /api/client/me/tenants`** so a signed-in client user can discover the
   tenants they belong to (to populate the `X-Tenant-ID` selector).
3. **Spec 0002 — Templates & Leads (client domain).** The current
   `LeadTemplate`/`LeadField`/`Lead`/`LeadValue` models and handlers are
   throwaway placeholders; they will be completely rebuilt in the **client**
   domain with `tenant_id` scoping. Out of scope here — deferred to Spec 0002.
4. Per-tenant **roles/permissions** on `user_tenants` (e.g. owner/member/viewer).
5. Optional Postgres RLS hardening.
6. Optional Keycloak Admin API provisioning on tenant create.
7. Tenant-scoped settings, quotas, and audit logging.
