# Zone Backup and Restore

This guide explains how to backup and restore DNS zones using the namedot API.

## Table of Contents

- [Export Formats](#export-formats)
- [Import Modes](#import-modes)
- [API Endpoints](#api-endpoints)
- [Backup Scripts](#backup-scripts)
- [Restore Scripts](#restore-scripts)
- [Upgrade Scenario](#upgrade-scenario)

## Export Formats

namedot supports two export formats:

- **JSON** - Structured format with full support for geo-aware records
- **BIND** - Standard BIND zone file format for compatibility

## Import Modes

When importing zones, you can choose between two modes:

- **upsert** (default) - Add new records and update existing ones
- **replace** - Delete all existing records and import new ones

## API Endpoints

### Export Single Zone

**JSON format:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/zones/{id}/export?format=json"
```

**BIND format:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/zones/{id}/export?format=bind"
```

### Export All Zones (Replication)

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/sync/export"
```

### Import Zone

**JSON format:**
```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @backup.json \
  "http://localhost:7070/zones/{id}/import?format=json&mode=upsert"
```

**BIND format:**
```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: text/plain" \
  --data-binary @backup.zone \
  "http://localhost:7070/zones/{id}/import?format=bind&mode=upsert"
```

## CLI Export/Import (Built-in)

The namedot binary includes built-in commands for direct database backup and restore:

### Export All Zones

```bash
# Export to file
namedot -export backup.json

# With custom config
namedot -c /etc/namedot/config.yaml -export backup.json
```

### Import Zones

```bash
# Import with merge mode (default - keeps existing zones)
namedot -import backup.json

# Import with replace mode (deletes all existing zones first)
namedot -import backup.json -import-mode replace

# With custom config
namedot -c /etc/namedot/config.yaml -import backup.json
```

**Import Modes:**
- `merge` (default): Import zones and records, keeping existing data. If a zone exists, its records are replaced.
- `replace`: Delete all existing zones and import from backup (complete restore).

**Advantages of CLI export/import:**
- Works directly with database (no need for running server)
- No authentication required
- Faster than API-based backup
- Useful for maintenance and migrations

## API Export/Import

For backup while the server is running, use the REST API endpoints:

## Backup Scripts

### Backup All Zones

Create a script `backup-zones.sh`:

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
BACKUP_DIR="./backup-$(date +%Y%m%d-%H%M%S)"

if [ -z "$TOKEN" ]; then
  echo "Error: TOKEN environment variable is required"
  echo "Usage: TOKEN=your-token $0"
  exit 1
fi

mkdir -p "$BACKUP_DIR"

echo "Fetching zone list..."
ZONES=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE/zones" | \
  python3 -c "import sys,json; [print(z['id']) for z in json.load(sys.stdin)]" 2>/dev/null)

if [ -z "$ZONES" ]; then
  echo "Error: Failed to fetch zones. Check your token and API endpoint."
  exit 1
fi

echo "Exporting zones..."
for zone_id in $ZONES; do
  curl -s -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/zones/$zone_id/export?format=json" > "$BACKUP_DIR/zone-$zone_id.json"

  if [ $? -eq 0 ]; then
    zone_name=$(python3 -c "import json; print(json.load(open('$BACKUP_DIR/zone-$zone_id.json'))['name'])" 2>/dev/null)
    echo "  Exported zone $zone_id: $zone_name"
  else
    echo "  Failed to export zone $zone_id"
  fi
done

echo ""
echo "Backup complete: $BACKUP_DIR"
echo "Total zones: $(ls -1 $BACKUP_DIR/zone-*.json 2>/dev/null | wc -l)"
```

Usage:
```bash
chmod +x backup-zones.sh
TOKEN="your-api-token" ./backup-zones.sh
```

### Backup Single Zone

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
ZONE_ID="$1"

if [ -z "$TOKEN" ] || [ -z "$ZONE_ID" ]; then
  echo "Usage: TOKEN=your-token $0 <zone_id>"
  exit 1
fi

curl -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/zones/$ZONE_ID/export?format=json" > "zone-$ZONE_ID-$(date +%Y%m%d-%H%M%S).json"

echo "Zone $ZONE_ID backed up"
```

## Restore Scripts

### Restore All Zones

Create a script `restore-zones.sh`:

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
BACKUP_DIR="$1"

if [ -z "$TOKEN" ] || [ -z "$BACKUP_DIR" ]; then
  echo "Usage: TOKEN=your-token $0 <backup_directory>"
  exit 1
fi

if [ ! -d "$BACKUP_DIR" ]; then
  echo "Error: Backup directory not found: $BACKUP_DIR"
  exit 1
fi

echo "Restoring zones from $BACKUP_DIR..."

for backup_file in "$BACKUP_DIR"/zone-*.json; do
  if [ ! -f "$backup_file" ]; then
    echo "No backup files found"
    exit 1
  fi

  # Extract zone name from backup file
  zone_name=$(python3 -c "import json; print(json.load(open('$backup_file'))['name'])" 2>/dev/null)

  if [ -z "$zone_name" ]; then
    echo "  Skipping invalid backup file: $backup_file"
    continue
  fi

  echo "  Processing zone: $zone_name"

  # Check if zone already exists
  existing_zone=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE/zones" | \
    python3 -c "import sys,json; zones=[z for z in json.load(sys.stdin) if z['name']=='$zone_name']; print(zones[0]['id'] if zones else '')" 2>/dev/null)

  if [ -n "$existing_zone" ]; then
    # Zone exists, import into existing zone
    zone_id="$existing_zone"
    echo "    Zone exists (ID: $zone_id), importing records..."
  else
    # Create new zone
    zone_id=$(curl -s -X POST \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"$zone_name\"}" \
      "$API_BASE/zones" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$zone_id" ]; then
      echo "    Failed to create zone"
      continue
    fi
    echo "    Created zone (ID: $zone_id)"
  fi

  # Import records
  response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$backup_file" \
    "$API_BASE/zones/$zone_id/import?format=json&mode=replace")

  http_code=$(echo "$response" | tail -n1)

  if [ "$http_code" = "204" ]; then
    echo "    Successfully restored"
  else
    echo "    Failed to import records (HTTP $http_code)"
  fi
done

echo ""
echo "Restore complete"
```

Usage:
```bash
chmod +x restore-zones.sh
TOKEN="your-api-token" ./restore-zones.sh ./backup-20241118-230000
```

## Upgrade Scenario

When upgrading namedot to a new version (especially with schema changes):

### Step 1: Backup Current Data

**Option A: Using CLI (Recommended)**
```bash
# Stop the current namedot service
sudo systemctl stop namedot

# Create backup directly from database
namedot -c /etc/namedot/config.yaml -export /tmp/backup-$(date +%Y%m%d).json
```

**Option B: Using API (if server is running)**
```bash
# Create backup via API
TOKEN="your-api-token" ./backup-zones.sh
```

### Step 2: Upgrade namedot

```bash
# Install new version
sudo dpkg -i namedot_0.2.0_amd64.deb
# or
sudo rpm -U namedot-0.2.0.x86_64.rpm
```

### Step 3: Initialize New Database

The new version will automatically create a fresh database with the updated schema on first start.

```bash
# Start namedot
sudo systemctl start namedot

# Check status
sudo systemctl status namedot
```

### Step 4: Restore Data

**Option A: Using CLI (Recommended)**
```bash
# Stop namedot
sudo systemctl stop namedot

# Restore from backup (replace mode - clean restore)
namedot -c /etc/namedot/config.yaml -import /tmp/backup-20241118.json -import-mode replace

# Start namedot
sudo systemctl start namedot
```

**Option B: Using API (if server is running)**
```bash
# Restore all zones from backup via API
TOKEN="your-api-token" ./restore-zones.sh ./backup-20241118-230000
```

### Step 5: Verify

```bash
# Check that all zones are restored
curl -H "Authorization: Bearer $TOKEN" http://localhost:7070/zones

# Test DNS resolution
dig @localhost -p 5353 example.com A
```

## Notes

- Backups include all zone data: DNS records, TTLs, and geo-aware routing rules
- The `sync/export` endpoint exports all zones in a single JSON file (useful for replication)
- Always test restore procedures on a non-production system first
- Keep multiple backup copies before major upgrades
- Backup files are plain JSON and can be edited manually if needed
