# Testing Guide

This document describes how to run unit and integration tests for BulwarkAuthAdmin.

## Quick Start

### Run All Tests (Automated)

```bash
./run-integration-tests.sh
```

This script will:
1. Start MongoDB in Docker
2. Start the BulwarkAuthAdmin service
3. Run all integration tests
4. Clean up everything

### Unit Tests Only

```bash
go test -v ./internal/accounts/...
```

No external dependencies required.

---

## Unit Tests

Unit tests use in-memory MongoDB (memongo) and test the repository layer in isolation.

### Running Unit Tests

```bash
# Run all unit tests
go test -v ./...

# Run account repository tests
go test -v ./internal/accounts/...

# Run with coverage
go test -v -cover ./internal/accounts/...
```

### Test Coverage

**Repository Layer** (`internal/accounts/mongodb_account_repository_test.go`)

- Create account
  - Valid account creation
  - Empty email validation
  - Duplicate email rejection

- Read by ID
  - Account found
  - Account not found

- Read by email
  - Account found
  - Account not found

- Read all accounts
  - Multiple accounts
  - Empty database

- Update account
  - Update email
  - Update deleted flag
  - Update enabled flag
  - Account not found

- Delete account
  - Account deleted
  - Account not found

### Running Specific Tests

```bash
# Run only Create tests
go test -v -run TestMongoDBAccountRepository_Create ./internal/accounts/...

# Run only one subtest
go test -v -run "TestMongoDBAccountRepository_Create/Duplicate" ./internal/accounts/...
```

---

## Integration Tests

Integration tests start the actual service and test the full HTTP API against a real MongoDB instance.

### Architecture

```
Test Process:
  Integration Tests
       ↓ (HTTP Requests)
  BulwarkAuthAdmin Service (localhost:8080)
       ↓ (Database Operations)
  MongoDB (Docker)
```

### Quick Start

**Option 1: Automated (Recommended)**

```bash
./run-integration-tests.sh
```

**Option 2: Manual Steps**

```bash
# Terminal 1: Start MongoDB
docker-compose -f docker-compose.test.yml up -d

# Terminal 2: Start service
go run cmd/bulwarkauthadmin/main.go

# Terminal 3: Run tests
go test -v -tags=integration ./tests/integration/...

# Cleanup when done
docker-compose -f docker-compose.test.yml down
```

### Running Integration Tests

```bash
# Run all integration tests
go test -v -tags=integration ./tests/integration/...

# Run account handler tests only
go test -v -tags=integration ./tests/integration/accounts

# Run specific test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

### Test Coverage

**Account Handlers** (`tests/integration/accounts/account_handler_integration_test.go`)

- RegisterAccount (POST /api/accounts)
  - Valid registration → 201 Created
  - Duplicate email → 409 Conflict

- ListAndGetAccount
  - List accounts → 200 OK
  - Get specific account → 200 OK
  - Account details verification

- ChangeEmail (PUT /api/accounts/email)
  - Change email → 204 No Content
  - Verify persistence

- DeactivateAccount (PUT /api/accounts/deactivate)
  - Deactivate → 204 No Content
  - Verify isDeleted flag

- DisableAccount (PUT /api/accounts/disable)
  - Disable → 204 No Content

- EnableAccount (PUT /api/accounts/enable)
  - Enable → 204 No Content

### Environment Variables

```bash
# Use different MongoDB instance
export DB_CONNECTION="mongodb://user:password@host:27017"

# Default: mongodb://localhost:27017
go test -v -tags=integration ./tests/integration/...
```

### Docker MongoDB

Start MongoDB for testing:

```bash
# Start MongoDB
docker-compose -f docker-compose.test.yml up -d

# Check status
docker-compose -f docker-compose.test.yml ps

# View logs
docker-compose -f docker-compose.test.yml logs mongodb

# Stop MongoDB
docker-compose -f docker-compose.test.yml down
```

---

## Test Organization

### Files

```
bulwarkauthadmin/
├── internal/accounts/
│   ├── mongodb_account_repository_test.go    # Unit tests
│   └── ...
├── tests/integration/
│   ├── setup.go                              # Test setup & helpers
│   ├── README.md                             # Integration test docs
│   └── accounts/
│       └── account_handler_integration_test.go  # Handler tests
├── docker-compose.test.yml                   # MongoDB setup
├── TESTING.md                                # This file
└── run-integration-tests.sh                  # Automated test runner
```

### Test Patterns

All tests follow consistent patterns:

**Unit Tests**
```go
// Table-driven tests
tests := []struct {
    name        string
    input       string
    expected    string
    expectedErr error
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}
```

**Integration Tests**
```go
// Setup service
integration.WaitForService(t, 20)
integration.CleanupDatabase(t)

// Make HTTP request
resp, err := http.Post(...)

// Assert response
assert.Equal(t, http.StatusCreated, resp.StatusCode)
```

---

## Troubleshooting

### Unit Tests

```
Error: "index already exists with a different name"
→ This is normal, memongo handles this gracefully
```

### Integration Tests

**MongoDB not connecting**
```bash
# Check if MongoDB is running
docker-compose -f docker-compose.test.yml logs mongodb

# Restart MongoDB
docker-compose -f docker-compose.test.yml restart
```

**Service not responding**
```bash
# Check if service is running
curl http://localhost:8080/api/accounts

# Check service logs
go run cmd/bulwarkauthadmin/main.go
# Look for: "connecting to mongodb"
```

**Tests timeout**
- Ensure MongoDB is running and healthy
- Ensure service is fully started (wait for output)
- Check that ports 8080 (service) and 27017 (MongoDB) are available

### Clean up stuck containers

```bash
# Stop all Docker containers
docker-compose -f docker-compose.test.yml down -v

# Check processes on port 8080
lsof -i :8080
kill -9 <PID>
```

---

## Best Practices

1. **Run unit tests frequently** - They're fast and require no setup
2. **Run integration tests before committing** - They catch real issues
3. **Use the automation script** - It handles all setup/cleanup
4. **Clean up MongoDB** - Use `docker-compose down -v` to remove volumes
5. **Check logs** - Both service and MongoDB logs help debugging

---

## CI/CD

For CI/CD pipelines:

```bash
# Unit tests only (no external services)
go test -v ./internal/accounts/...

# Both unit and integration tests (Docker required)
./run-integration-tests.sh
```
