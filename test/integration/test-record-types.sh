#!/bin/bash
# Integration tests for DNS record types
# Tests A, AAAA, CNAME, MX, TXT record types

set -e

API_BASE="http://127.0.0.1:7070"
DNS_SERVER="127.0.0.1"
DNS_PORT="5353"
ZONE_NAME="recordtypes.test"
CONFIG_FILE="test/integration/test-config.yaml"
DB_FILE="test/integration/test.db"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASSED=0
FAILED=0
SERVER_PID=""

echo "========================================"
echo "DNS Record Types Test Suite"
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

# Create A record
echo -n "Creating A record (www.$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"www\",
    \"type\": \"A\",
    \"ttl\": 300,
    \"records\": [
      {\"data\": \"192.0.2.10\"},
      {\"data\": \"192.0.2.11\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create AAAA record
echo -n "Creating AAAA record (ipv6.$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"ipv6\",
    \"type\": \"AAAA\",
    \"ttl\": 300,
    \"records\": [
      {\"data\": \"2001:db8::10\"},
      {\"data\": \"2001:db8::11\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create CNAME record
echo -n "Creating CNAME record (alias.$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"alias\",
    \"type\": \"CNAME\",
    \"ttl\": 300,
    \"records\": [{\"data\": \"www.$ZONE_NAME.\"}]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create MX record
echo -n "Creating MX record (@$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"@\",
    \"type\": \"MX\",
    \"ttl\": 3600,
    \"records\": [
      {\"data\": \"10 mail1.$ZONE_NAME.\"},
      {\"data\": \"20 mail2.$ZONE_NAME.\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create TXT record
echo -n "Creating TXT record (_dmarc.$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"_dmarc\",
    \"type\": \"TXT\",
    \"ttl\": 300,
    \"records\": [
      {\"data\": \"\\\"v=DMARC1; p=none; rua=mailto:dmarc@$ZONE_NAME\\\"\"}
    ]
  }" > /dev/null 2>&1; then
  echo -e "${GREEN}OK${NC}"
else
  echo -e "${RED}FAIL${NC}"
  exit 1
fi

# Create another TXT record for SPF
echo -n "Creating TXT record (spf @$ZONE_NAME)... "
if curl -s -X POST "$API_BASE/zones/$ZONE_ID/rrsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"@\",
    \"type\": \"TXT\",
    \"ttl\": 300,
    \"records\": [
      {\"data\": \"\\\"v=spf1 mx -all\\\"\"}
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
  local query_name=$1
  local query_type=$2
  local expected_value=$3
  local test_description=$4

  echo -n "Testing: $test_description... "

  # Execute dig query
  result=$(dig @$DNS_SERVER -p $DNS_PORT $query_name $query_type +short | head -1)

  if [[ "$result" == *"$expected_value"* ]]; then
    echo -e "${GREEN}PASS${NC} (got: $result)"
    PASSED=$((PASSED + 1))
  else
    echo -e "${RED}FAIL${NC} (expected: *$expected_value*, got: $result)"
    FAILED=$((FAILED + 1))
  fi
}

# Function to test multiple DNS responses
test_query_multiple() {
  local query_name=$1
  local query_type=$2
  local expected_count=$3
  local test_description=$4

  echo -n "Testing: $test_description... "

  # Execute dig query and count results
  result=$(dig @$DNS_SERVER -p $DNS_PORT $query_name $query_type +short)
  count=$(echo "$result" | grep -v '^$' | wc -l)

  if [ "$count" -eq "$expected_count" ]; then
    echo -e "${GREEN}PASS${NC} (got $count records)"
    PASSED=$((PASSED + 1))
  else
    echo -e "${RED}FAIL${NC} (expected: $expected_count records, got: $count)"
    echo "Result: $result"
    FAILED=$((FAILED + 1))
  fi
}

echo "========================================"
echo "A Record Tests"
echo "========================================"
echo ""

test_query_multiple "www.$ZONE_NAME" "A" 2 "A record should return 2 IPv4 addresses"
test_query "www.$ZONE_NAME" "A" "192.0.2.10" "A record should contain 192.0.2.10"

echo ""
echo "========================================"
echo "AAAA Record Tests"
echo "========================================"
echo ""

test_query_multiple "ipv6.$ZONE_NAME" "AAAA" 2 "AAAA record should return 2 IPv6 addresses"
test_query "ipv6.$ZONE_NAME" "AAAA" "2001:db8::10" "AAAA record should contain 2001:db8::10"

echo ""
echo "========================================"
echo "CNAME Record Tests"
echo "========================================"
echo ""

test_query "alias.$ZONE_NAME" "CNAME" "www.$ZONE_NAME" "CNAME should point to www.$ZONE_NAME"

echo ""
echo "========================================"
echo "MX Record Tests"
echo "========================================"
echo ""

test_query_multiple "$ZONE_NAME" "MX" 2 "MX record should return 2 mail servers"
test_query "$ZONE_NAME" "MX" "mail1.$ZONE_NAME" "MX record should contain mail1.$ZONE_NAME"

echo ""
echo "========================================"
echo "TXT Record Tests"
echo "========================================"
echo ""

test_query "_dmarc.$ZONE_NAME" "TXT" "v=DMARC1" "TXT record should contain DMARC policy"
test_query "$ZONE_NAME" "TXT" "v=spf1" "TXT record should contain SPF policy"

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
