# Integration Tests

Integration tests for the BulwarkAuthAdmin microservice test the full application stack against real MongoDB, BulwarkAuth, and MailHog instances.

## Quick Start

The easiest way to run integration tests is with the provided script:

```bash
./run-integration-tests.sh
```

This script:
1. Starts MongoDB, MailHog, and BulwarkAuth services via Docker Compose
2. Builds the BulwarkAuthAdmin service
3. Runs all integration tests
4. Cleans up containers

## Manual Setup (Advanced)

If you prefer manual setup:

### 1. Start Test Infrastructure

```bash
docker-compose -f docker-compose.test.yml up -d
```

This starts:
- **MongoDB** on `localhost:27017` (replica set mode)
- **MailHog** on `localhost:8025` (email capture)
- **BulwarkAuth** on `localhost:8080` (authentication service)

Verify services are running:
```bash
docker-compose -f docker-compose.test.yml ps
```

### 2. Set Environment Variables

```bash
export BULWARK_AUTH_URL=http://localhost:8080
export BULWARK_ADMIN_URL=http://localhost:8081
export MAILHOG_URL=http://localhost:8025
export DB_CONNECTION="mongodb://localhost:27017/?directConnection=true"
export ADMIN_ACCOUNT=admin@test.example.com
export ADMIN_ACCOUNT_PASSWORD=TestAdminPassword123!
```

### 3. Build and Start the Service

```bash
go build -o /tmp/bulwark-admin-service cmd/bulwarkauthadmin/main.go cmd/bulwarkauthadmin/config.go
/tmp/bulwark-admin-service
```

The service will start on `http://localhost:8081`.

### 4. Run Integration Tests

In another terminal:

```bash
# Run all integration tests
go test -v -tags=integration ./tests/integration/...

# Run specific domain tests
go test -v -tags=integration ./tests/integration/accounts
go test -v -tags=integration ./tests/integration/rbac
go test -v -tags=integration ./tests/integration/tenants

# Run specific test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

## Cleanup

### Automated (with script)
The `run-integration-tests.sh` script handles cleanup automatically.

### Manual
```bash
# Stop containers
docker-compose -f docker-compose.test.yml down

# Remove volumes (clean slate)
docker-compose -f docker-compose.test.yml down -v
```

## What Gets Tested

### Account Management Tests
- **RegisterAccount** - Create new account
- **ListAccounts** - List all accounts (paginated)
- **GetAccount** - Retrieve specific account
- **ChangeEmail** - Update account email
- **DeactivateAccount** - Soft delete account
- **DisableAccount** - Disable account
- **EnableAccount** - Enable account

### API Key Tests
- **CreateApiKey** - Generate new API key, verify plaintext format
- **CreateApiKey_DuplicateName** - Duplicate name returns 409 Conflict
- **CreateApiKey_MissingAccountID** - Missing account ID rejected
- **ListApiKeys** - List all API keys for an account
- **GetApiKey** - Retrieve specific API key details
- **SuspendApiKey** - Suspend API key, verify isEnabled is false
- **EnableApiKey** - Re-enable suspended key, verify isEnabled is true
- **RevokeApiKey** - Delete API key, verify removal from list

### RBAC Tests
- **CreateRole** - Create new role with description
- **ListRoles** - List all roles
- **GetRole** - Retrieve specific role
- **UpdateRole** - Update role description
- **DeleteRole** - Delete role
- **CreatePermission** - Create new permission
- **ListPermissions** - List all permissions
- **DeletePermission** - Delete permission
- **RolePermissionFlow** - Add/remove permissions to/from roles

### Account RBAC Tests
- **AssignRole** - Assign role to account
- **RemoveRole** - Remove role from account
- **AssignPermission** - Assign permission to account
- **RemovePermission** - Remove permission from account

### Tenant Management Tests
- **CreateTenant** - Create new tenant
- **ListTenants** - List all tenants
- **GetTenant** - Retrieve specific tenant
- **UpdateTenant** - Update tenant details
- **DeleteTenant** - Delete tenant

### Tenant Admin Authorization Tests
- **TenantAdminMiddlewareRequiresAuth** - JWT validation
- **TenantAdminMiddlewareRequiresTenantAdminRole** - Role checking
- **TenantAdminCanAccessTenantEndpoints** - Tenant admin access
- **TenantAdminCannotAccessOtherTenants** - Tenant isolation
- **SystemAdminCanAccessAnyTenant** - System admin privileges
- **TenantAdminRoleCreatedAutomatically** - Auto role creation

## Test Infrastructure Architecture

```
┌──────────────────────────────────────────────────────────┐
│         Integration Test Architecture                    │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  Integration Tests (tests/integration/*_test.go)        │
│         │                                                │
│         │ HTTP Requests (Bearer JWT tokens)             │
│         ▼                                                │
│  BulwarkAuthAdmin Service (localhost:8081)              │
│  - Account Management                                   │
│  - API Key Management                                   │
│  - RBAC Management                                      │
│  - Tenant Management                                    │
│         │                                                │
│         │ Database Operations                           │
│         ▼                                                │
│  MongoDB Replica Set (localhost:27017)                  │
│  (Shared database with BulwarkAuth)                     │
│         │                                                │
│         └────────────────────┐                          │
│                              │                          │
│  BulwarkAuth Service (localhost:8080)                   │
│  - Authentication                                       │
│  - Account verification                                │
│  - JWT issuance                                         │
│                              │                          │
│                              ▼                          │
│  MailHog (localhost:8025)                               │
│  - Email capture for verification tokens                │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

## Key Test Helpers

The `tests/integration/setup.go` file provides helpers:

- **`NewTestContext(t)`** - Create authenticated test context
- **`SetupTestTenant(t)`** - Setup test tenant and user
- **`SetupSystemAdminContext(t)`** - Create system admin context
- **`MakeAuthenticatedRequest()`** - Make HTTP request with JWT
- **`GetVerificationTokenFromEmail()`** - Extract token from MailHog
- **`ExtractTokenFromEmailBody()`** - Parse email body for token

## Requirements

- Go 1.24+
- Docker & Docker Compose
- MongoDB 7.0.0+ (via docker-compose)
- BulwarkAuth service (via docker-compose)
- MailHog service (via docker-compose)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BULWARK_AUTH_URL` | `http://localhost:8080` | BulwarkAuth service URL |
| `BULWARK_ADMIN_URL` | `http://localhost:8081` | BulwarkAuthAdmin service URL |
| `MAILHOG_URL` | `http://localhost:8025` | MailHog API URL |
| `DB_CONNECTION` | `mongodb://localhost:27017/?directConnection=true` | MongoDB connection string |
| `DB_NAME_SEED` | `` (empty) | Database name suffix for test isolation |
| `ADMIN_ACCOUNT` | `admin@test.example.com` | System admin email |
| `ADMIN_ACCOUNT_PASSWORD` | `TestAdminPassword123!` | System admin password |

## Troubleshooting

### Tests fail with "MongoDB not available"
```bash
# Check if MongoDB is running
docker-compose -f docker-compose.test.yml logs mongodb

# Restart containers
docker-compose -f docker-compose.test.yml down
docker-compose -f docker-compose.test.yml up -d
```

### Tests fail with "BulwarkAuth service not ready"
```bash
# Check BulwarkAuth logs
docker-compose -f docker-compose.test.yml logs bulwarkauth

# Ensure BulwarkAuth is running
curl http://localhost:8080/health
```

### Tests fail with "service did not become available"
```bash
# Check BulwarkAuthAdmin logs
cat /tmp/service.log

# Start the service manually
go run cmd/bulwarkauthadmin/main.go
```

### Docker platform mismatch warning
You may see warnings like:
```
The requested image's platform (linux/amd64) does not match the detected host platform (linux/arm64/v8)
```

This is normal when running on Apple Silicon (ARM64) with amd64 images. The containers will still work via emulation.

## Debugging Tests

### Run single test with output
```bash
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts -count=1
```

### Check service logs
```bash
tail -f /tmp/service.log
```

### Query MongoDB directly
```bash
docker-compose -f docker-compose.test.yml exec mongodb mongosh bulwarkauth
```

### View emails in MailHog
```
http://localhost:8025
```

## Continuous Integration

The tests are run automatically via GitHub Actions in `.github/workflows/publish.yml`:

```yaml
- name: Run integration tests
  run: bash run-integration-tests.sh
```

All tests must pass before releases are created.
