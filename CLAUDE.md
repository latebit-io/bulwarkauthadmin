# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

## Directory Structure

```
api/                      # HTTP handlers and routes
  accounts/               # Account management endpoints
  health/                 # Health check endpoint
  problem/                # RFC 7807 problem details (standardized errors)

internal/                 # Business logic and data access
  accounts/               # Account domain
    accounts.go           # Service interfaces and implementations
    mongodb_account_repository.go  # Repository implementation
    error.go              # Domain-specific errors
  rbac/                   # RBAC (scaffolded - interfaces only)
    roles.go              # Role/Permission interfaces
  shared/                 # Common utilities (paging)
  utils/                  # Test utilities (memongo setup)
  version/                # Version info

cmd/bulwarkauthadmin/     # Application entry point
  main.go                 # Service initialization
  config.go               # Environment configuration
  .env                    # Configuration file

tests/integration/        # Integration tests
  setup.go                # Test infrastructure
  accounts/               # Account handler tests
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
go test -v ./internal/accounts/...
go test -v -cover ./internal/accounts/...

# Integration tests (requires MongoDB + running service)
./run-integration-tests.sh

# Manual integration tests
docker-compose -f docker-compose.test.yml up -d
go run cmd/bulwarkauthadmin/main.go  # In another terminal
go test -v -tags=integration ./tests/integration/...

# Run specific integration test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts

# Cleanup
docker-compose -f docker-compose.test.yml down
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

### Account Structure

```go
type Account struct {
    ID                 string            // UUID
    Email              string            // Unique, required
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
- Email has unique index at database level
- Roles and Permissions are stored as string arrays (names, not ObjectIDs)

### RBAC Models (scaffolded)

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

### Account Management
- `POST /api/accounts` - Register new account
- `GET /api/accounts` - List all accounts (paginated)
- `GET /api/accounts/:id` - Get account details
- `PUT /api/accounts/email` - Change account email
- `PUT /api/accounts/disable` - Disable account
- `PUT /api/accounts/enable` - Enable account
- `PUT /api/accounts/deactivate` - Soft delete account
- `PUT /api/accounts/unlink` - Unlink social provider

### Health
- `GET /health` - Health check

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

## Current Development Status

### Implemented
- Account management (CRUD operations)
- Social provider linking/unlinking
- Soft delete and hard delete (purge)
- Email uniqueness enforcement
- CORS middleware (configurable)
- Unit and integration testing infrastructure
- CI/CD with automated versioning

### Scaffolded (Interfaces Only)
- RBAC (Role and Permission repositories and services)
- JWT token management
- Magic code management

### Active Branch
- `feat-add-cors` - CORS middleware implementation
- Main branch: `main`

## MongoDB Collections

- **accounts** - User accounts with unique email index
- **roles** - (Scaffolded) RBAC roles
- **permissions** - (Scaffolded) RBAC permissions

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
