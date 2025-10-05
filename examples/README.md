# Configuration Examples

This directory contains example configuration files for different deployment scenarios.

## Files

### config.yaml
Basic configuration example for a standalone DNS server.

### config.master.yaml
Configuration for a **master server** in a replication setup.

Key features:
- `replication.mode: "master"` - enables master mode
- Admin panel and DNS updates are enabled
- Serves replication data via `/sync/export` endpoint

### config.slave.yaml
Configuration for a **slave server** in a replication setup.

Key features:
- `replication.mode: "slave"` - enables slave mode
- `replication.master_url` - URL of the master server
- `replication.sync_interval_sec` - synchronization interval
- **Auto-disabled** features in slave mode:
  - Admin panel (`admin.enabled` → `false`)
  - DNS updates (`update.enabled` → `false`)

## Usage

### Deploy Master Server

```bash
cp examples/config.master.yaml config.yaml
# Edit api_token with a secure token
./namedot
```

### Deploy Slave Server

```bash
cp examples/config.slave.yaml config.yaml
# Edit:
#   - replication.master_url (e.g., "http://192.168.1.100:8080")
#   - replication.api_token (same as master)
#   - replication.sync_interval_sec (optional, default: 60)
./namedot
```

## Notes

- Slave servers automatically synchronize data from the master
- Admin panel and updates are automatically disabled on slave servers for security
- Both master and slave use the same binary
- See [REPLICATION.md](../REPLICATION.md) for detailed documentation
