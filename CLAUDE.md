# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. 

Important: The AI agent should always be in ask mode and should never modify the code with out explicit Permission or if asked by the developer. 

## Project Overview

**BulwarkAuthAdmin** is a Go microservice for managing user accounts and authentication. It provides REST API endpoints for account lifecycle operations, social provider management, and role-based access control (RBAC). Built with Echo web framework and MongoDB.

**Current Version:** v0.2.0  
**Go Version:** 1.24.0  
**Module:** github.com/latebit-io/bulwarkauthadmin

## Architecture

This codebase follows a **layered architecture with anemic domain models**:

```
HTTP Layer (api/)
    ↓
Service Layer (internal/*/accounts.go - business logic)
    ↓
Repository Layer (internal/*/mongodb_*_repository.go - data access)
    ↓
MongoDB Database
```

### Key Architectural Decisions

1. **Anemic Models**: Data structures are separate from business logic. Models are plain structs used for data transfer.

2. **Service Layer Naming**: Use business-oriented method names (`RegisterAccount`, `GrantPermissionsToRole`) rather than CRUD terms (`Create`, `Update`).

3. **Repository Layer Naming**: Use technical CRUD terms (`Create`, `Read`, `Update`, `Delete`, `AddPermissions`).

4. **Return Values vs Pointers**: Prefer returning values `(Model, error)` over pointers `(*Model, error)` to avoid nil pointer issues. Only use pointers for truly optional fields within structs.

5. **Natural Keys for RBAC**: Roles and Permissions use their name as the primary key (not ObjectIDs). Names are immutable - renaming means creating a new role/permission. This provides:
   - Human-readable references in logs and database
   - Simpler permission checks (no joins needed)
   - Better API ergonomics
   - Database portability

6. **RBAC Design**: Hybrid model supporting both role-based and direct permission grants:
   - `User → Roles → Permissions` (primary path)
   - `User → Permissions` (direct grants for exceptions)

## API Documentation

### OpenAPI/Swagger Specification

The complete API is documented in OpenAPI 3.0 format:
- **File:** `openapi.yaml`
- **View online:** Use any OpenAPI viewer (Swagger UI, Redoc, etc.)
- **Tools:** `swag init` can generate code from this spec if needed

**Key sections:**
- All endpoints documented with request/response schemas
- Authentication requirements clearly marked
- Authorization rules for tenant admin vs system admin
- RFC 7807 Problem Details for error responses
- Comprehensive component schemas for data models

## Directory Structure

```
api/                      # HTTP handlers and routes
  accounts/               # Account management endpoints
  health/                 # Health check endpoint
  problem/                # RFC 7807 problem details (standardized errors)
  rbac/                   # RBAC endpoints
  tenants/                # Tenant management endpoints

internal/                 # Business logic and data access
  accounts/               # Account domain
    accounts.go           # Service interfaces and implementations
    mongodb_account_repository.go  # Repository implementation
    error.go              # Domain-specific errors
  rbac/                   # RBAC domain
    roles.go              # Role/Permission services and repositories
    permissions.go        # Permission implementation
  tenants/                # Tenant domain
    tenants.go            # Service interfaces and implementations
    mongodb_tenants_repository.go  # Repository implementation
    admin_roles.go        # TenantAdminService for creating default tenant admin roles
    error.go              # Domain-specific errors
  email/                  # Email template management
    emails.go             # Email service
    mongodb_email_repository.go  # Repository
  shared/                 # Common utilities (paging)
  utils/                  # Test utilities (memongo setup)
  version/                # Version info
  middleware/             # JWT and tenant middleware

cmd/bulwarkauthadmin/     # Application entry point
  main.go                 # Service initialization and route registration
  config.go               # Environment configuration
  .env                    # Configuration file

openapi.yaml              # OpenAPI 3.0 specification for the entire API
.github/workflows/        # CI/CD workflows
docker-compose.yml        # Docker Compose for local development
docker-compose.test.yml   # Docker Compose for testing

tests/integration/        # Integration tests
  setup.go                # Test infrastructure and helpers
  accounts/               # Account handler tests
  rbac/                   # RBAC handler tests
  tenants/                # Tenant handler tests
```

## Common Commands

### Development

```bash
# Run the service
go run cmd/bulwarkauthadmin/main.go

# Run with custom .env file
cp cmd/bulwarkauthadmin/.env cmd/bulwarkauthadmin/.env.local
# Edit .env.local, then:
go run cmd/bulwarkauthadmin/main.go
```

### Testing

```bash
# Unit tests (no external dependencies)
go test -v ./internal/...
go test -v -cover ./internal/...

# Integration tests (requires MongoDB + running service)
./run-integration-tests.sh

# Run specific integration test package
go test -v -tags=integration ./tests/integration/accounts
go test -v -tags=integration ./tests/integration/rbac
go test -v -tags=integration ./tests/integration/tenants

# Run specific test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

### Database

```bash
# Start MongoDB for development/testing
docker-compose -f docker-compose.test.yml up -d

# View MongoDB logs
docker-compose -f docker-compose.test.yml logs mongodb

# Stop MongoDB
docker-compose -f docker-compose.test.yml down

# Remove volumes (clean slate)
docker-compose -f docker-compose.test.yml down -v
```

### Build and Release

```bash
# Build binary
go build -o bulwarkauthadmin cmd/bulwarkauthadmin/main.go

# Run linting (if golangci-lint installed)
golangci-lint run

# Test GoReleaser config (requires Docker)
goreleaser release --snapshot --clean
```

## Data Models

### Tenant Structure

```go
type Tenant struct {
    ID          string    // UUID
    Name        string    // Unique, immutable
    Description string
    Domain      string
    Created     time.Time
    Modified    time.Time
}
```

**Important Notes:**
- Tenant name has unique index at database level
- System tenant created on startup with ID `00000000-0000-0000-0000-000000000000`

### Account Structure

```go
type Account struct {
    ID                 string            // UUID
    TenantID           string            // Reference to tenant
    Email              string            // Unique per tenant
    IsVerified         bool
    VerificationToken  string            // UUID
    IsEnabled          bool              // Account active status
    IsDeleted          bool              // Soft delete flag
    SocialProviders    []SocialProvider
    Roles              []string          // Role names (natural keys)
    Permissions        []string          // Permission names (natural keys)
    Created            time.Time
    Modified           time.Time
}
```

**Important Notes:**
- Accounts use soft deletes (`IsDeleted` flag) by default
- `PurgeAccount` performs hard deletion
- Email has unique index per tenant
- Roles and Permissions are stored as string arrays (names, not ObjectIDs)

### RBAC Models

```go
type Role struct {
    Name        string    `bson:"_id"`  // Natural key (immutable)
    Description string
    Permissions []string  // Permission names
    Created     time.Time
    Modified    time.Time
}

type Permission struct {
    Name        string    `bson:"_id"`  // Natural key (immutable)
    Description string
    Resource    string    // e.g., "users"
    Action      string    // e.g., "delete"
}
```

## API Endpoints

All routes are under `/api/v1/` prefix.

### Account Management (Tenant-scoped: `/api/v1/tenant/:tenantid/accounts`)
**Requires:** Tenant admin or system admin role in the tenant
- `POST /accounts` - Register new account
- `GET /accounts` - List all accounts (paginated)
- `GET /accounts/:id` - Get account details
- `PUT /accounts/email` - Change account email
- `PUT /accounts/disable` - Disable account
- `PUT /accounts/enable` - Enable account
- `PUT /accounts/deactivate` - Soft delete account
- `PUT /accounts/unlink` - Unlink social provider

### RBAC Management (Tenant-scoped: `/api/v1/tenant/:tenantid/rbac`)
**Requires:** Tenant admin or system admin role in the tenant
- `POST /roles` - Create role
- `GET /roles` - List roles
- `GET /roles/:name` - Get role details
- `PUT /roles/:name` - Update role
- `DELETE /roles/:name` - Delete role
- `POST /permissions` - Create permission
- `GET /permissions` - List permissions
- `DELETE /permissions/:name` - Delete permission

### Account RBAC (Tenant-scoped: `/api/v1/tenant/:tenantid/accounts/rbac`)
**Requires:** Tenant admin or system admin role in the tenant
- `POST /roles` - Assign role to account
- `DELETE /roles` - Remove role from account
- `POST /permissions` - Assign permission to account
- `DELETE /permissions` - Remove permission from account

### Tenant Management (Admin-scoped: `/api/v1/admin/tenants`)
**Requires:** System admin role
- `POST /tenants` - Create tenant
- `GET /tenants` - List all tenants
- `GET /tenants/:id` - Get tenant details
- `PUT /tenants/:id` - Update tenant
- `DELETE /tenants/:id` - Delete tenant

### Health
- `GET /health` - Health check (no authentication required)

## Environment Configuration

Key environment variables (see `cmd/bulwarkauthadmin/.env`):

```bash
PORT=8080                                    # Server port
CORS_ENABLED=false                          # Enable CORS
ALLOWED_WEB_ORIGINS=http://localhost:5173  # CORS origins (comma-separated)

DB_CONNECTION=mongodb://localhost:27017/?connect=direct
DB_NAME_SEED=""                             # Database suffix (e.g., "test")

BULWARK_AUTH_URL=http://localhost:5173     # Frontend URL
```

## Testing Strategy

### Unit Tests
- Use in-memory MongoDB (memongo)
- Test repository layer in isolation
- No external dependencies required
- Table-driven test patterns
- Located alongside code (`*_test.go`)

### Integration Tests
- Test full HTTP API against real MongoDB
- Build tag: `//go:build integration`
- Located in `tests/integration/`
- Require MongoDB running on localhost:27017
- Use helper functions in `tests/integration/setup.go`

## Code Style and Patterns

### Error Handling

Use RFC 7807 Problem Details for HTTP responses:
```go
problem.InternalServerError(c, "failed to create account", err)
problem.Conflict(c, "account already exists", err)
problem.BadRequest(c, "invalid account id", err)
```

### Repository Pattern

```go
type AccountRepository interface {
    Create(ctx context.Context, account Account) (Account, error)
    Read(ctx context.Context, id string) (Account, error)
    ReadByEmail(ctx context.Context, email string) (Account, error)
    ReadAll(ctx context.Context, opts shared.PagingOptions) ([]Account, error)
    Update(ctx context.Context, account Account) error
    Delete(ctx context.Context, id string) error
}
```

### Service Pattern

```go
type AccountManagementService interface {
    RegisterAccount(ctx context.Context, email string) (Account, error)
    GetAccountDetails(ctx context.Context, id string) (AccountDetails, error)
    ChangeAccountEmail(ctx context.Context, id, newEmail string) error
    DisableAccount(ctx context.Context, id string) error
    EnableAccount(ctx context.Context, id string) error
    DeactivateAccount(ctx context.Context, id string) error
    UnlinkSocialProvider(ctx context.Context, accountID, providerName string) error
}
```

### MongoDB Patterns

- Use `primitive.ObjectID` for MongoDB `_id` fields
- Use context for all database operations
- Create unique indexes in repository constructors
- Handle `mongo.ErrNoDocuments` for not found cases

## CI/CD

### GitHub Actions Workflow

`.github/workflows/publish.yml` handles:
1. **Semantic Versioning** - Auto-calculates version from commit messages:
   - `BREAKING CHANGE:` → major version bump
   - `feat:` → minor version bump
   - `fix:` → patch version bump
2. **Tag Creation** - Automatic git tags on main branch
3. **GoReleaser** - Cross-platform builds (Linux, macOS, Windows on amd64/arm64)
4. **Docker Images** - Published to GitHub Container Registry

### Commit Message Format

Use conventional commits:
```
feat: add new feature
fix: bug fix
BREAKING CHANGE: breaking API change
```

## Authorization & RBAC

### Role Hierarchy

The system supports a hierarchical authorization model:

1. **System Admin (`bulwark_admin`)**
   - Located in the system tenant (UUID nil: `00000000-0000-0000-0000-000000000000`)
   - Can perform operations across all tenants
   - Implicitly has tenant admin permissions for all tenants
   - Routes: `/api/v1/admin/*`

2. **Tenant Admin (`tenant_admin`)**
   - Located in their respective tenant
   - Can manage accounts and RBAC within their tenant only
   - Created automatically when a tenant is created
   - Cannot access other tenants

3. **Regular User**
   - Can perform read-only operations or operations assigned via direct permissions
   - Cannot access tenant management endpoints

### Tenant Admin Functionality

Tenant admins can:
- List, create, and manage accounts in their tenant
- Create and manage roles and permissions
- Assign roles (including `tenant_admin`) to other accounts
- View and manage RBAC configurations

**Default Permissions Created per Tenant:**
- `accounts:manage` - For managing accounts
- `rbac:manage` - For managing roles and permissions

**Default Role Created per Tenant:**
- `tenant_admin` - Has both `accounts:manage` and `rbac:manage` permissions

### Middleware Implementation

- **JWT Validation** - `JwtMiddleware` validates tokens and extracts claims
- **Tenant Extraction** - `ExtractAndAuthorizeTenant` validates tenant access
- **Tenant Admin Check** - `RequireTenantAdminOrSystemAdmin` enforces admin role requirement
- **System Admin Check** - `RequireSystemAdmin` enforces system admin role requirement

All tenant-scoped routes automatically require either `tenant_admin` or `bulwark_admin` role.

## Current Development Status

### Fully Implemented
- **Multi-tenant architecture** - System tenant + user-managed tenants
- **Account management** - Registration, lifecycle, social provider linking
- **RBAC system** - Roles, permissions, role-based and direct permission grants
- **Tenant management** - Create, read, update, delete tenants (system admin only)
- **Tenant Admin Authorization** - Tenant admins can manage their tenant's accounts and RBAC
- **Email templates** - Per-tenant email template management
- **Authentication** - JWT validation with tenant context extraction
- **CORS middleware** - Configurable cross-origin support
- **Comprehensive testing** - Unit tests + integration tests for all domains
  - Account tests: 3 unit tests + 7 integration tests
  - RBAC tests: 2 unit tests + 10 integration tests
  - Tenant tests: 8 unit tests + 12 integration tests
  - Middleware tests: 6 unit tests for authorization helpers
  - Tenant admin tests: 6 integration tests
- **CI/CD** - Automated versioning, building, and Docker image publishing

### Active Branch
- Main branch: `main`
- Current feature branch: `feat-admin-logon`

## MongoDB Collections

- **tenants** - Multi-tenant configuration with unique name index
- **accounts** - User accounts (per tenant) with unique email index per tenant
- **roles** - RBAC roles (per tenant)
- **permissions** - RBAC permissions (per tenant)
- **email_templates** - Email templates (per tenant)

Database name: `bulwarkauth{DB_NAME_SEED}` (e.g., `bulwarkauth`, `bulwarkauthtest`)

## Important Conventions

1. **Do not rename roles or permissions** - Names are immutable identifiers. Create new ones instead.

2. **Use value returns, not pointers** - Return `(Model, error)` rather than `(*Model, error)` for safety.

3. **Service methods use business language** - `AssignRoleToUser`, not `CreateUserRole`.

4. **Repository methods use CRUD language** - `Create`, `Read`, `Update`, `Delete`.

5. **Soft deletes by default** - Use `IsDeleted` flag. Only use `PurgeAccount` when explicitly requested.

6. **MongoDB natural keys for RBAC** - Use name strings as `_id` for roles and permissions.

7. **Always read before write** - When modifying existing files, read them first to understand the context.

8. **Avoid over-engineering** - Only implement what's requested. Don't add extra features, error handling for impossible scenarios, or premature abstractions.

9. **Tenant Admin Role is Automatic** - The `tenant_admin` role and its permissions are created automatically:
   - When the service starts (for the system tenant)
   - When a new tenant is created via `TenantAdminService.CreateTenantAdminRole()`
   - Idempotent design - safe to call multiple times

10. **System Admins Implicitly Have Tenant Admin Access** - Use `IsTenantAdminOrSystemAdmin()` for permission checks on tenant-scoped routes to allow system admins to bypass tenant admin checks and access any tenant.

11. **Middleware Ordering Matters** - Apply middleware in this order for tenant-scoped routes:
    1. JWT validation (`jwt.Jwt`)
    2. Tenant extraction and authorization (`tenantMiddleware.ExtractAndAuthorizeTenant`)
    3. Tenant admin role check (`tenantMiddleware.RequireTenantAdminOrSystemAdmin`)
