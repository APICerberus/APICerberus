#!/bin/bash
# Test coverage script for API Cerberus

set -e

echo "API Cerberus Test Coverage Report"
echo "================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run tests with coverage for all packages
echo "Running tests with coverage..."
echo ""

PACKAGES=(
  "./internal/auth/..."
  "./internal/balancer/..."
  "./internal/cache/..."
  "./internal/config/..."
  "./internal/federation/..."
  "./internal/grpc/..."
  "./internal/graphql/..."
  "./internal/loadbalancer/..."
  "./internal/mcp/..."
  "./internal/metrics/..."
  "./internal/observability/..."
  "./internal/proxy/..."
  "./internal/raft/..."
  "./internal/router/..."
  "./internal/server/..."
  "./internal/storage/..."
  "./internal/upstream/..."
)

TOTAL_COVERAGE=0
PACKAGE_COUNT=0

for pkg in "${PACKAGES[@]}"; do
  echo -n "Testing $pkg... "

  # Run tests with coverage
  if go test -coverprofile=coverage.out "$pkg" > /dev/null 2>&1; then
    # Get coverage percentage
    COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//')

    if [ -z "$COVERAGE" ]; then
      COVERAGE=0
    fi

    # Check if coverage meets threshold
    if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
      echo -e "${GREEN}${COVERAGE}%${NC}"
    elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then
      echo -e "${YELLOW}${COVERAGE}%${NC}"
    else
      echo -e "${RED}${COVERAGE}%${NC}"
    fi

    TOTAL_COVERAGE=$(echo "$TOTAL_COVERAGE + $COVERAGE" | bc)
    ((PACKAGE_COUNT++))
  else
    echo -e "${RED}FAILED${NC}"
  fi
done

# Clean up
rm -f coverage.out

# Calculate average
if [ $PACKAGE_COUNT -gt 0 ]; then
  AVERAGE_COVERAGE=$(echo "scale=2; $TOTAL_COVERAGE / $PACKAGE_COUNT" | bc)
else
  AVERAGE_COVERAGE=0
fi

echo ""
echo "================================="
echo "Average Coverage: ${AVERAGE_COVERAGE}%"
echo ""

# Check if we meet the 80% target
if (( $(echo "$AVERAGE_COVERAGE >= 80" | bc -l) )); then
  echo -e "${GREEN}✅ PASS: Coverage meets 80% target${NC}"
  exit 0
else
  echo -e "${RED}❌ FAIL: Coverage below 80% target${NC}"
  exit 1
fi
