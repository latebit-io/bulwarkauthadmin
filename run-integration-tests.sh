#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}BulwarkAuthAdmin Integration Tests${NC}"
echo -e "${YELLOW}========================================${NC}"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    exit 1
fi

# Start all services (MongoDB, MailHog, BulwarkAuth)
echo -e "\n${YELLOW}1. Starting test infrastructure (MongoDB, MailHog, BulwarkAuth)...${NC}"
docker-compose -f docker-compose.test.yml up -d
sleep 2

# Wait for MongoDB to be ready
echo -e "${YELLOW}2. Waiting for MongoDB to be ready...${NC}"
for i in {1..60}; do
    if docker-compose -f docker-compose.test.yml exec -T mongodb mongosh --eval "rs.status().ok" > /dev/null 2>&1; then
        echo -e "${GREEN}MongoDB replica set is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}MongoDB did not become ready in time${NC}"
        docker-compose -f docker-compose.test.yml logs mongodb
        docker-compose -f docker-compose.test.yml down
        exit 1
    fi
    sleep 1
done

# Wait for BulwarkAuth to be ready
echo -e "${YELLOW}3. Waiting for BulwarkAuth service to be ready...${NC}"
for i in {1..60}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}BulwarkAuth service is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}BulwarkAuth service did not become ready in time${NC}"
        docker-compose -f docker-compose.test.yml logs bulwarkauth
        docker-compose -f docker-compose.test.yml down
        exit 1
    fi
    sleep 1
done

# Build and start BulwarkAuthAdmin service in background
echo -e "\n${YELLOW}4. Building and starting BulwarkAuthAdmin service...${NC}"

# Set environment variables for BulwarkAuthAdmin
export BULWARK_AUTH_URL=http://localhost:8080
export DB_CONNECTION="mongodb://localhost:27017/?directConnection=true"
export DB_NAME_SEED=test
export PORT=8081

go build -o /tmp/bulwark-admin-service cmd/bulwarkauthadmin/main.go cmd/bulwarkauthadmin/config.go 2>&1
if [ ! -f /tmp/bulwark-admin-service ]; then
    echo -e "${RED}Failed to build service${NC}"
    docker-compose -f docker-compose.test.yml down
    exit 1
fi
/tmp/bulwark-admin-service > /tmp/service.log 2>&1 &
SERVICE_PID=$!

# Wait for BulwarkAuthAdmin service to be ready
echo -e "${YELLOW}5. Waiting for BulwarkAuthAdmin service to be ready...${NC}"
for i in {1..60}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo -e "${GREEN}BulwarkAuthAdmin service is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}BulwarkAuthAdmin service did not become ready in time${NC}"
        kill $SERVICE_PID 2>/dev/null || true
        docker-compose -f docker-compose.test.yml down
        echo -e "\n${YELLOW}Service logs:${NC}"
        cat /tmp/service.log
        exit 1
    fi
    sleep 1
done

# Run integration tests
echo -e "\n${YELLOW}6. Running integration tests...${NC}"
export BULWARK_AUTH_URL=http://localhost:8080
export BULWARK_ADMIN_URL=http://localhost:8081
export MAILHOG_URL=http://localhost:8025

if go test -v -tags=integration ./tests/integration/...; then
    TEST_RESULT=0
    echo -e "\n${GREEN}✓ Integration tests passed!${NC}"
else
    TEST_RESULT=1
    echo -e "\n${RED}✗ Integration tests failed${NC}"
fi

# Cleanup
echo -e "\n${YELLOW}7. Cleaning up...${NC}"
kill $SERVICE_PID 2>/dev/null || true
docker-compose -f docker-compose.test.yml down

if [ $TEST_RESULT -eq 0 ]; then
    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}All tests passed!${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo -e "\n${RED}========================================${NC}"
    echo -e "${RED}Tests failed. See output above.${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
