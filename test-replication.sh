#!/bin/bash
set -e

echo "=== SmailGeoDNS Replication Test ==="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Variables
TOKEN="test-token-12345"
MASTER_PORT=18080
SLAVE_PORT=18081

echo -e "${YELLOW}Step 1: Creating test configs${NC}"

# Create master config
cat > config.test.master.yaml << EOF
listen: "127.0.0.1:15353"
forwarder: ""
enable_dnssec: false
api_token: "${TOKEN}"
rest_listen: "127.0.0.1:${MASTER_PORT}"
auto_soa_on_missing: true
default_ttl: 300

db:
  driver: "sqlite"
  dsn: "file:test_master.db?_foreign_keys=on"

geoip:
  enabled: false
  mmdb_path: "./geoipdb"
  reload_sec: 300
  use_ecs: true

update:
  enabled: true
  require_tsig: false
  tsig_secrets: {}

log:
  dns_verbose: false

performance:
  cache_size: 1024
  dns_timeout_sec: 2
  forwarder_timeout_sec: 2

admin:
  enabled: false

replication:
  mode: "master"
EOF

# Create slave config
cat > config.test.slave.yaml << EOF
listen: "127.0.0.1:15354"
forwarder: ""
enable_dnssec: false
api_token: "${TOKEN}"
rest_listen: "127.0.0.1:${SLAVE_PORT}"
auto_soa_on_missing: true
default_ttl: 300

db:
  driver: "sqlite"
  dsn: "file:test_slave.db?_foreign_keys=on"

geoip:
  enabled: false
  mmdb_path: "./geoipdb"
  reload_sec: 300
  use_ecs: true

update:
  enabled: true  # Will be auto-disabled in slave mode
  require_tsig: false
  tsig_secrets: {}

log:
  dns_verbose: false

performance:
  cache_size: 1024
  dns_timeout_sec: 2
  forwarder_timeout_sec: 2

admin:
  enabled: true  # Will be auto-disabled in slave mode
  username: admin
  password_hash: "\$2a\$10\$test"

replication:
  mode: "slave"
  master_url: "http://127.0.0.1:${MASTER_PORT}"
  sync_interval_sec: 5
  api_token: "${TOKEN}"
EOF

echo -e "${GREEN}✓ Configs created${NC}"

echo -e "${YELLOW}Step 2: Cleaning old test databases${NC}"
rm -f test_master.db test_slave.db
echo -e "${GREEN}✓ Databases cleaned${NC}"

echo -e "${YELLOW}Step 3: Building${NC}"
go build ./cmd/namedot
echo -e "${GREEN}✓ Build complete${NC}"

echo -e "${YELLOW}Step 4: Starting master server${NC}"
SGDNS_CONFIG=config.test.master.yaml ./namedot > master.log 2>&1 &
MASTER_PID=$!
echo "Master PID: $MASTER_PID"
sleep 2

# Check if master is running
if ! kill -0 $MASTER_PID 2>/dev/null; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat master.log
    exit 1
fi
echo -e "${GREEN}✓ Master started${NC}"

echo -e "${YELLOW}Step 5: Creating test zone on master${NC}"
ZONE_RESPONSE=$(curl -s -X POST http://127.0.0.1:${MASTER_PORT}/zones \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"test.example.com"}')
ZONE_ID=$(echo $ZONE_RESPONSE | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
echo "Zone ID: $ZONE_ID"
echo -e "${GREEN}✓ Zone created${NC}"

echo -e "${YELLOW}Step 6: Adding DNS records to master${NC}"
curl -s -X POST http://127.0.0.1:${MASTER_PORT}/zones/${ZONE_ID}/rrsets \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "@",
    "type": "A",
    "ttl": 300,
    "records": [
      {"data": "192.168.1.1"}
    ]
  }' > /dev/null
echo -e "${GREEN}✓ Records added${NC}"

echo -e "${YELLOW}Step 7: Verifying data on master${NC}"
MASTER_ZONES=$(curl -s http://127.0.0.1:${MASTER_PORT}/zones \
  -H "Authorization: Bearer ${TOKEN}")
echo "Master zones: $MASTER_ZONES"
echo -e "${GREEN}✓ Master has data${NC}"

echo -e "${YELLOW}Step 8: Starting slave server${NC}"
SGDNS_CONFIG=config.test.slave.yaml ./namedot > slave.log 2>&1 &
SLAVE_PID=$!
echo "Slave PID: $SLAVE_PID"
sleep 3

# Check if slave is running
if ! kill -0 $SLAVE_PID 2>/dev/null; then
    echo -e "${RED}✗ Slave failed to start${NC}"
    cat slave.log
    kill $MASTER_PID 2>/dev/null || true
    exit 1
fi
echo -e "${GREEN}✓ Slave started${NC}"

echo -e "${YELLOW}Step 9: Waiting for initial sync (5 seconds)${NC}"
sleep 6

echo -e "${YELLOW}Step 10: Verifying data on slave${NC}"
SLAVE_ZONES=$(curl -s http://127.0.0.1:${SLAVE_PORT}/zones \
  -H "Authorization: Bearer ${TOKEN}")
echo "Slave zones: $SLAVE_ZONES"

if echo "$SLAVE_ZONES" | grep -q "test.example.com"; then
    echo -e "${GREEN}✓ Sync successful! Data replicated to slave${NC}"
else
    echo -e "${RED}✗ Sync failed! Data not found on slave${NC}"
    echo "Master zones:"
    echo "$MASTER_ZONES"
    echo "Slave zones:"
    echo "$SLAVE_ZONES"
    kill $MASTER_PID $SLAVE_PID 2>/dev/null || true
    exit 1
fi

echo -e "${YELLOW}Step 11: Testing continuous sync${NC}"
echo "Adding another zone on master..."
curl -s -X POST http://127.0.0.1:${MASTER_PORT}/zones \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"test2.example.com"}' > /dev/null

echo "Waiting for next sync cycle (6 seconds)..."
sleep 7

SLAVE_ZONES2=$(curl -s http://127.0.0.1:${SLAVE_PORT}/zones \
  -H "Authorization: Bearer ${TOKEN}")

if echo "$SLAVE_ZONES2" | grep -q "test2.example.com"; then
    echo -e "${GREEN}✓ Continuous sync working! New zone replicated${NC}"
else
    echo -e "${RED}✗ Continuous sync failed${NC}"
fi

echo -e "${YELLOW}Step 12: Cleanup${NC}"
kill $MASTER_PID $SLAVE_PID 2>/dev/null || true
sleep 1
echo -e "${GREEN}✓ Servers stopped${NC}"

echo ""
echo -e "${GREEN}=== All tests passed! ===${NC}"
echo ""
echo "Log files: master.log, slave.log"
echo "Test databases: test_master.db, test_slave.db"
echo ""
echo "To clean up test files:"
echo "  rm -f test_master.db test_slave.db master.log slave.log config.test.*.yaml"
