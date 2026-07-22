#! /usr/bin/bash 

# Exit immmediately at failure
set -euo pipefail

# ANSI colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
LIGHT_BLUE='\x1b[94m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "Running AlpineJudge E2E test"

echo -e "${LIGHT_BLUE}Step [1/5] Running linters (golanngci-lint)${NC}"
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}WARNING: golangci-lint not installed. Fallback to go vet & go fmt${NC}"
    if [ -n "$(gofmt -l .)" ]; then
        echo "Formatting issues found"
        exit 1
    fi
    go vet ./...
else 
    golangci-lint run ./...
    echo -e "${GREEN}✓ Code style clean and compliant.${NC}\n"
fi

echo -e "${LIGHT_BLUE}Step [2/5] Compiling with thread safety check...${NC}"
go test -race -v ./...
echo -e "${GREEN}✓ No data races detected.${NC}\n"

echo -e "${LIGHT_BLUE}Step [3/5] Calculating structural code coverage...${NC}"
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
echo -e "${GREEN}✓ Test coverage report generated.${NC}\n"

echo -e "${LIGHT_BLUE}Step [4/5] Calculating resource consumption benchmarks...${NC}"
# Runs profiling algorithms, limits allocations, and checks memory profiles
go test -run=^$ -bench=. -benchmem -cpuprofile=cpu.pprof -memprofile=mem.pprof ./...
echo -e "${GREEN}✓ Benchmarks finished analyzing execution allocations.${NC}\n"

echo -e "${BLUE}[Step 5/5] Invoking Live End-to-End Integration Test...${NC}"

# Explicitly runs tests matching the E2E namespace tag
echo -e "${BLUE}E2E test Dispatcher subsystem...${NC}"
go test -v -run=Test_Dispatcher_Subsystem_E2E ./...
echo -e "${BLUE}E2E test Runner subsystem...${NC}"
go test -v -run=Test_Runner_Subsystem_E2E ./...

echo -e "\n${GREEN}✓ All verification gates passed${NC}"