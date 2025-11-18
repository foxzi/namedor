#!/bin/bash
# Test script for GeoDNS functionality
# Automatically starts server, runs tests, and stops server

set -e

API_BASE="http://127.0.0.1:7070"
DNS_SERVER="127.0.0.1"
DNS_PORT="5353"
ZONE_NAME="geodns.test"
CONFIG_FILE="test/integration/test-config.yaml"
DB_FILE="test/integration/test.db"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
SERVER_PID=""

echo "========================================"
echo "GeoDNS Test Suite"
echo "========================================"
echo ""

# Cleanup function
cleanup() {
  if [ -n "$SERVER_PID" ]; then
    echo ""
    echo "Stopping server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    echo -e "${GREEN}✓ Server stopped${NC}"
  fi

  # Clean up test database
  if [ -f "$DB_FILE" ]; then
    rm -f "$DB_FILE"
    echo -e "${GREEN}✓ Test database cleaned${NC}"
  fi
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Check if dig is available
if ! command -v dig &> /dev/null; then
  echo -e "${RED}ERROR: dig command not found${NC}"
  echo "Install it with: sudo apt-get install dnsutils"
  exit 1
fi

# Check if namedot binary exists
if [ ! -f "./namedot" ]; then
  echo -e "${RED}ERROR: namedot binary not found${NC}"
  echo "Build it with: make build"
  exit 1
fi

# Start server
echo "Starting namedot server..."
./namedot -c $CONFIG_FILE > /tmp/namedot-test.log 2>&1 &
SERVER_PID=$!
echo "Server started with PID: $SERVER_PID"

# Wait for server to start
echo -n "Waiting for server to start"
for i in {1..30}; do
  if dig @$DNS_SERVER -p $DNS_PORT version.bind txt chaos +short &> /dev/null; then
    echo ""
    echo -e "${GREEN}✓ DNS server is running${NC}"
    break
  fi
  echo -n "."
  sleep 0.5
done

# Check if server started successfully
if ! dig @$DNS_SERVER -p $DNS_PORT version.bind txt chaos +short &> /dev/null; then
  echo ""
  echo -e "${RED}ERROR: Failed to start DNS server${NC}"
  echo "Server log:"
  cat /tmp/namedot-test.log
  exit 1
fi

# Wait for API to start
echo -n "Waiting for API server to start"
for i in {1..30}; do
  if curl -s "$API_BASE/health" > /dev/null 2>&1; then
    echo ""
    echo -e "${GREEN}✓ API server is running${NC}"
    break
  fi
  echo -n "."
  sleep 0.5
done

# Check if API started successfully
if ! curl -s "$API_BASE/health" > /dev/null 2>&1; then
  echo ""
  echo -e "${RED}ERROR: Failed to start API server${NC}"
  exit 1
fi

echo ""

# Setup test data
echo "Setting up test data..."

# Create zone
TOKEN="test-token-12345"
echo -n "Creating zone: $ZONE_NAME... "
RESPONSE=$(curl -s -X POST "$API_BASE/zones" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$ZONE_NAME\"}")

ZONE_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')

if [ -z "$ZONE_ID" ]; then
  echo -e "${RED}FAIL${NC}"
  echo "API Response: $RESPONSE"
  exit 1
fi
echo -e "${GREEN}OK${NC} (ID: $ZONE_ID)"

# Create A record with geolocation routing
echo -n "Creating www.$ZONE_NAME (country-based)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"www\",
    \"type\": \"A\",
    \"ttl\": 60,
    \"records\": [
      {\"data\": \"1.1.1.1\", \"country\": \"RU\"},
      {\"data\": \"2.2.2.2\", \"country\": \"GB\"},
      {\"data\": \"3.3.3.3\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create another test record for ASN-based routing
echo -n "Creating asn.$ZONE_NAME (ASN-based)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"asn\",
    \"type\": \"A\",
    \"ttl\": 60,
    \"records\": [
      {\"data\": \"10.1.1.1\", \"asn\": 65001},
      {\"data\": \"10.2.2.2\", \"asn\": 65002},
      {\"data\": \"10.3.3.3\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

echo ""

# Function to test DNS query
test_query() {
  local source_ip=$1
  local query_name=$2
  local expected_ip=$3
  local test_description=$4

  echo -n "Testing: $test_description... "

  # Execute dig query with source IP
  result=$(dig @$DNS_SERVER -p $DNS_PORT -b $source_ip $query_name A +short | head -1)

  if [ "$result" == "$expected_ip" ]; then
    echo -e "${GREEN}PASS${NC} (got: $result)"
    PASSED=$((PASSED + 1))
  else
    echo -e "${RED}FAIL${NC} (expected: $expected_ip, got: $result)"
    FAILED=$((FAILED + 1))
  fi
}

echo "========================================"
echo "Country-based GeoDNS Tests"
echo "========================================"
echo ""

# Test country-based routing for www.geodns.test
test_query "127.0.1.10" "www.$ZONE_NAME" "1.1.1.1" "RU (127.0.1.10) → 1.1.1.1"
test_query "127.0.1.50" "www.$ZONE_NAME" "1.1.1.1" "RU (127.0.1.50) → 1.1.1.1"
test_query "127.0.2.10" "www.$ZONE_NAME" "2.2.2.2" "GB (127.0.2.10) → 2.2.2.2"
test_query "127.0.2.50" "www.$ZONE_NAME" "2.2.2.2" "GB (127.0.2.50) → 2.2.2.2"
test_query "127.0.3.10" "www.$ZONE_NAME" "3.3.3.3" "Other (127.0.3.10) → 3.3.3.3"
test_query "127.0.0.1" "www.$ZONE_NAME" "3.3.3.3" "Other (127.0.0.1) → 3.3.3.3"

echo ""
echo "========================================"
echo "ASN-based GeoDNS Tests"
echo "========================================"
echo ""

# Test ASN-based routing for asn.geodns.test
test_query "127.0.1.10" "asn.$ZONE_NAME" "10.1.1.1" "AS65001 (127.0.1.10) → 10.1.1.1"
test_query "127.0.1.50" "asn.$ZONE_NAME" "10.1.1.1" "AS65001 (127.0.1.50) → 10.1.1.1"
test_query "127.0.2.10" "asn.$ZONE_NAME" "10.2.2.2" "AS65002 (127.0.2.10) → 10.2.2.2"
test_query "127.0.2.50" "asn.$ZONE_NAME" "10.2.2.2" "AS65002 (127.0.2.50) → 10.2.2.2"
test_query "127.0.3.10" "asn.$ZONE_NAME" "10.3.3.3" "Other (127.0.3.10) → 10.3.3.3"
test_query "127.0.0.1" "asn.$ZONE_NAME" "10.3.3.3" "Other (127.0.0.1) → 10.3.3.3"

echo ""
echo "========================================"
echo "Test Results"
echo "========================================"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "Total:  $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
