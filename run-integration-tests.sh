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

# Start MongoDB
echo -e "\n${YELLOW}1. Starting MongoDB...${NC}"
docker-compose -f docker-compose.test.yml up -d
sleep 2

# Wait for MongoDB to be ready
echo -e "${YELLOW}2. Waiting for MongoDB to be ready...${NC}"
for i in {1..60}; do
    if docker-compose -f docker-compose.test.yml exec -T mongodb mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
        echo -e "${GREEN}MongoDB is ready!${NC}"
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

# Build and start the service in background
echo -e "\n${YELLOW}3. Building and starting BulwarkAuthAdmin service...${NC}"
go build -o /tmp/bulwark-service cmd/bulwarkauthadmin/main.go cmd/bulwarkauthadmin/config.go 2>&1
if [ ! -f /tmp/bulwark-service ]; then
    echo -e "${RED}Failed to build service${NC}"
    docker-compose -f docker-compose.test.yml down
    exit 1
fi
/tmp/bulwark-service > /tmp/service.log 2>&1 &
SERVICE_PID=$!

# Wait for service to be ready
echo -e "${YELLOW}4. Waiting for service to be ready...${NC}"
for i in {1..60}; do
    if curl -s http://localhost:8080/api/accounts > /dev/null 2>&1; then
        echo -e "${GREEN}Service is ready!${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}Service did not become ready in time${NC}"
        kill $SERVICE_PID 2>/dev/null || true
        docker-compose -f docker-compose.test.yml down
        echo -e "\n${YELLOW}Service logs:${NC}"
        cat /tmp/service.log
        exit 1
    fi
    sleep 1
done

# Run integration tests
echo -e "\n${YELLOW}5. Running integration tests...${NC}"
if go test -v -tags=integration ./tests/integration/...; then
    TEST_RESULT=0
    echo -e "\n${GREEN}✓ Integration tests passed!${NC}"
else
    TEST_RESULT=1
    echo -e "\n${RED}✗ Integration tests failed${NC}"
fi

# Cleanup
echo -e "\n${YELLOW}6. Cleaning up...${NC}"
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
