# BulwarkAuthAdmin

A Go microservice for managing user accounts and authentication. Provides REST API endpoints for account lifecycle operations, social provider management, and role-based access control (RBAC). Built with Echo web framework and MongoDB.

**Version:** v0.2.0  
**Go Version:** 1.24.0+  
**Module:** github.com/latebit-io/bulwarkauthadmin

## Table of Contents

- [Installation](#installation)
- [Running the Service](#running-the-service)
- [Environment Variables](#environment-variables)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [Development](#development)
- [Testing](#testing)

## Installation

### Prerequisites

- **Go 1.24.0 or later** - [Download Go](https://golang.org/dl/)
- **MongoDB 4.0+** - For database backend
- **Docker** (optional) - For containerized MongoDB

### Clone the Repository

```bash
git clone https://github.com/latebit-io/bulwarkauthadmin.git
cd bulwarkauthadmin
```

### Install Dependencies

```bash
go mod download
go mod verify
```

### Build the Binary

```bash
go build -o bulwarkauthadmin cmd/bulwarkauthadmin/main.go
```

## Running the Service

### Setup MongoDB

#### Option 1: Using Docker Compose (Recommended for Development)

```bash
# Start MongoDB
docker-compose -f docker-compose.test.yml up -d

# Verify MongoDB is running
docker-compose -f docker-compose.test.yml logs mongodb

# Stop MongoDB
docker-compose -f docker-compose.test.yml down

# Remove volumes (clean slate)
docker-compose -f docker-compose.test.yml down -v
```

#### Option 2: Using Local MongoDB

Ensure MongoDB is running on `localhost:27017` (or configure `DB_CONNECTION` accordingly).

### Configure Environment Variables

Create a `.env` file in `cmd/bulwarkauthadmin/` or set environment variables. See [Environment Variables](#environment-variables) for details.

```bash
# Copy the default .env file
cp cmd/bulwarkauthadmin/.env cmd/bulwarkauthadmin/.env.local

# Edit configuration as needed
# nano cmd/bulwarkauthadmin/.env.local
```

### Start the Service

#### Option 1: Using Go Run (Development)

```bash
go run cmd/bulwarkauthadmin/main.go
```

#### Option 2: Using the Built Binary

```bash
./bulwarkauthadmin
```

The service will start and output:
```
⇨ http server started on [::]:8081
```

By default, the service runs on `http://localhost:8081`.

### Verify the Service

```bash
# Health check
curl http://localhost:8081/health
```

## Environment Variables

Configure the service behavior using the following environment variables:

### Core Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `PORT` | int | `8081` | Server port to listen on |
| `BULWARK_AUTH_URL` | string | `http://localhost:8080` | Frontend application URL |

### Database Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DB_CONNECTION` | string | `mongodb://localhost:27017/?connect=direct` | MongoDB connection string |
| `DB_NAME_SEED` | string | `` (empty) | Optional database name suffix (e.g., `test` creates `bulwarkauthtest`) |

### CORS Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CORS_ENABLED` | bool | `false` | Enable Cross-Origin Resource Sharing |
| `ALLOWED_WEB_ORIGINS` | string | `http://localhost:5173` | Comma-separated list of allowed origins (e.g., `http://localhost:5173,https://example.com`) |

### Initial Admin Account Setup

⚠️ **SECURITY WARNING**: These variables are intended only for initial provisioning. After the first successful startup (once the admin account is created or linked), **immediately remove** `ADMIN_ACCOUNT` and `ADMIN_ACCOUNT_PASSWORD` from the environment and any configuration files or deployment manifests.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ADMIN_ACCOUNT` | string | `` (empty) | Email address for the initial admin account |
| `ADMIN_ACCOUNT_PASSWORD` | string | `` (empty) | Password for the initial admin account |

### Email Templates

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `EMAIL_TEMPLATES_DIR` | string | `` (empty) | Directory path containing email templates. When empty (the default), templates are loaded from the current working directory. If a custom path is provided, it must include a trailing slash (e.g., `/path/to/templates/`). In Docker deployments, templates are pre-installed at `/app/`, so this variable typically does not need to be set. |

### Example .env File

```bash
# Server
PORT=8081
BULWARK_AUTH_URL=http://localhost:3000

# Database
DB_CONNECTION=mongodb://localhost:27017/?connect=direct
DB_NAME_SEED=

# CORS
CORS_ENABLED=false
ALLOWED_WEB_ORIGINS=http://localhost:5173,http://localhost:3000

# Initial Admin (REMOVE AFTER FIRST STARTUP)
ADMIN_ACCOUNT=admin@example.com
ADMIN_ACCOUNT_PASSWORD=SecurePassword123!

# Email Templates
EMAIL_TEMPLATES_DIR=/path/to/email/templates/
```

### Setting Environment Variables

#### Using .env File
Place a `.env` file in the working directory when running the service:
```bash
go run cmd/bulwarkauthadmin/main.go
```

#### Using Shell Environment
```bash
export PORT=8080
export DB_CONNECTION=mongodb://mongodb.example.com:27017/?connect=direct
export CORS_ENABLED=true
go run cmd/bulwarkauthadmin/main.go
```

#### Using Command Line
```bash
PORT=8080 DB_CONNECTION=mongodb://localhost:27017 go run cmd/bulwarkauthadmin/main.go
```

## Architecture

This service follows a **layered architecture** with clear separation of concerns:

```
HTTP Layer (api/)
    ↓
Service Layer (internal/*/accounts.go - business logic)
    ↓
Repository Layer (internal/*/mongodb_*_repository.go - data access)
    ↓
MongoDB Database
```

### Key Design Principles

- **Anemic Models**: Data structures are separate from business logic
- **Service Layer**: Business-oriented method names (`RegisterAccount`, `GrantPermissionsToRole`)
- **Repository Layer**: Technical CRUD terms (`Create`, `Read`, `Update`, `Delete`)
- **Natural Keys for RBAC**: Roles and Permissions use names as primary keys (immutable)
- **Soft Deletes**: Accounts use soft delete flag by default

## API Endpoints

All routes are under `/api/v1/` prefix.

### Account Management
Base: `/api/v1/tenant/:tenantid/accounts`

- `POST /accounts` - Register new account
- `GET /accounts` - List accounts (paginated)
- `GET /accounts/:id` - Get account details
- `PUT /accounts/email` - Change email
- `PUT /accounts/disable` - Disable account
- `PUT /accounts/enable` - Enable account
- `PUT /accounts/deactivate` - Soft delete account
- `PUT /accounts/unlink` - Unlink social provider

### RBAC Management
Base: `/api/v1/tenant/:tenantid/rbac`

- `POST /roles` - Create role
- `GET /roles` - List roles
- `GET /roles/:name` - Get role details
- `PUT /roles/:name` - Update role
- `DELETE /roles/:name` - Delete role
- `POST /permissions` - Create permission
- `GET /permissions` - List permissions
- `DELETE /permissions/:name` - Delete permission

### Tenant Management
Base: `/api/v1/admin/tenants` (requires system admin role)

- `POST /tenants` - Create tenant
- `GET /tenants` - List tenants
- `GET /tenants/:id` - Get tenant details
- `PUT /tenants/:id` - Update tenant
- `DELETE /tenants/:id` - Delete tenant

### Health Check

- `GET /health` - Health check endpoint

## Development

### Project Structure

```
api/                          # HTTP handlers and routes
  accounts/                   # Account management endpoints
  health/                     # Health check endpoint
  problem/                    # RFC 7807 problem details (errors)
  rbac/                       # RBAC endpoints
  tenants/                    # Tenant management endpoints

internal/                     # Business logic and data access
  accounts/                   # Account domain
  rbac/                       # RBAC domain
  tenants/                    # Tenant domain
  email/                      # Email template management
  shared/                     # Common utilities
  utils/                      # Test utilities
  version/                    # Version info
  middleware/                 # JWT and tenant middleware

cmd/bulwarkauthadmin/         # Application entry point
  main.go                     # Service initialization
  config.go                   # Configuration management
  .env                        # Default environment variables

tests/integration/            # Integration tests
```

## Testing

### Unit Tests

Run all unit tests (no external dependencies):

```bash
go test -v ./internal/...
go test -v -cover ./internal/...
```

### Integration Tests

Integration tests require MongoDB running:

```bash
# Start MongoDB
docker-compose -f docker-compose.test.yml up -d

# Run all integration tests
./run-integration-tests.sh

# Run specific domain tests
go test -v -tags=integration ./tests/integration/accounts
go test -v -tags=integration ./tests/integration/rbac
go test -v -tags=integration ./tests/integration/tenants

# Run specific test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

### Linting

```bash
golangci-lint run
```

## Build and Release

### Build Binary

```bash
go build -o bulwarkauthadmin cmd/bulwarkauthadmin/main.go
```

### Build with GoReleaser

Requires Docker:

```bash
goreleaser release --snapshot --clean
```

## Contributing

See the project's CLAUDE.md for architectural guidelines and coding conventions.

## License

Proprietary - Bulwark Auth Admin
