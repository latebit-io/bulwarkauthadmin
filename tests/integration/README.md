# Integration Tests

Integration tests for the BulwarkAuthAdmin microservice test the full application stack against a real MongoDB instance.

## Quick Start

### 1. Start MongoDB with Docker

```bash
docker-compose -f docker-compose.test.yml up -d
```

Verify MongoDB is running:
```bash
docker-compose -f docker-compose.test.yml ps
```

### 2. Start the Application

In a separate terminal, start the service:

```bash
go run cmd/bulwarkauthadmin/main.go
```

The service will start on `http://localhost:8080`.

### 3. Run Integration Tests

In another terminal, run the tests:

```bash
go test -v -tags=integration ./tests/integration/...
```

Or run specific tests:

```bash
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

## Cleanup

Stop the MongoDB container after tests:

```bash
docker-compose -f docker-compose.test.yml down
```

## What Gets Tested

Integration tests verify the complete request/response cycle:

1. **RegisterAccount** - POST /api/accounts
   - Create new account
   - Duplicate email rejection
   
2. **ListAndGetAccount** - GET /api/accounts and GET /api/accounts/:id
   - List all accounts
   - Retrieve specific account
   
3. **ChangeEmail** - PUT /api/accounts/email
   - Update email successfully
   - Verify persistence
   
4. **DeactivateAccount** - PUT /api/accounts/deactivate
   - Soft delete account
   - Verify isDeleted flag
   
5. **DisableAccount** - PUT /api/accounts/disable
   - Disable account
   
6. **EnableAccount** - PUT /api/accounts/enable
   - Enable account

## Architecture

```
┌─────────────────────────────────────────────────────┐
│         Integration Test Process                    │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Tests (tests/integration/accounts/*.go)           │
│         │                                           │
│         │ HTTP Requests                            │
│         ▼                                           │
│  Service Running on localhost:8080                 │
│  (started manually with: go run main.go)           │
│         │                                           │
│         │ Database Operations                      │
│         ▼                                           │
│  MongoDB (running in docker-compose)               │
│         │                                           │
│         │ Response                                 │
│         ▼                                           │
│  Tests verify response and state                   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## Test Flow

1. `TestMain()` waits for MongoDB to be available
2. Each test:
   - Waits for service to be available
   - Cleans database
   - Makes HTTP requests to running service
   - Verifies responses
   - Verifies data persistence in MongoDB

## Requirements

- Go 1.24+
- Docker & Docker Compose
- MongoDB 7.0.0+ (via docker-compose)
- Running BulwarkAuthAdmin service on localhost:8080

## Troubleshooting

### MongoDB not connecting

```bash
# Check if MongoDB is running
docker-compose -f docker-compose.test.yml logs mongodb

# Restart MongoDB
docker-compose -f docker-compose.test.yml down
docker-compose -f docker-compose.test.yml up -d
```

### Service not responding

```bash
# Check if service is running
curl http://localhost:8080/api/accounts

# Start service if not running
go run cmd/bulwarkauthadmin/main.go
```

### Tests timeout waiting for service

Make sure the service is fully started before running tests. Service startup may take a few seconds to:
1. Connect to MongoDB
2. Create indexes
3. Start HTTP server

Wait for this output in service logs:
```
Bulwark Auth Admin Service Started on :8080
```

## Environment Variables

To use a different MongoDB instance, set `DB_CONNECTION`:

```bash
export DB_CONNECTION="mongodb://user:password@host:27017"
go test -v -tags=integration ./tests/integration/...
```

Default: `mongodb://localhost:27017`
