SmaillGeoDNS — Lightweight DNS server with REST + GeoDNS

Overview
- UDP/TCP DNS on :53
- Zones and records in DB (GORM: Postgres/MySQL/SQLite)
- REST API for zone management (+ JSON/BIND export, JSON import)
- Geo-aware responses (subnet/country/continent), ECS support
- Optional forwarder for cache-miss
- Simple in-memory TTL cache

Requirements
- Go >= 1.23 (рекомендуется 1.24+)

Quick Start
1) Create `config.yaml` in repo root:

```
listen: ":53"
forwarder: "8.8.8.8"
enable_dnssec: false
api_token: "devtoken"
rest_listen: ":8080"
auto_soa_on_missing: true
default_ttl: 300

db:
  driver: "sqlite"
  dsn: "file:smaillgeodns.db?_foreign_keys=on"

geoip:
  enabled: false
  mmdb_path: "/var/lib/maxmind/GeoLite2-City.mmdb"
  reload_sec: 300
  use_ecs: true
```

2) Build and run:
- `go build ./cmd/smaillgeodns`
- `sudo ./smaillgeodns` (DNS on :53 requires privileges or port redirect)

REST API (Bearer devtoken)
- Create zone: `POST /zones {"name":"example.com"}`
- Add rrset: `POST /zones/{id}/rrsets` with body similar to tz.md
- Export: `GET /zones/{id}/export?format=json|bind`
- Import: `POST /zones/{id}/import?format=json&mode=upsert|replace`

Notes
- DNSSEC dynamic signing is not implemented yet. You can store DNSSEC records (DNSKEY/RRSIG/DS) in DB and serve them as-is when queried.
- Geo selection currently supports subnet/country/continent attributes on records. ASN requires GeoIP DB integration and is a TODO.

Dynamic Updates (RFC 2136)
- Enable via config `update.enabled: true`. Optionally enforce TSIG: `update.require_tsig: true`.
- Configure `update.tsig_secrets` as a map of keyname to base64 HMAC secret. Example in `examples/config.yaml`.
- The DNS server processes basic add/delete operations and bumps SOA serial.

BIND Import
- REST: `POST /zones/{id}/import?format=bind&mode=upsert|replace` with raw zone text in body.
- Export remains available via `GET /zones/{id}/export?format=bind`.

Testing
- Unit tests (modules):
  - BIND import/export: `go test ./internal/server/rest/zoneio -run TestImportBIND_And_ToBind -count=1`
  - RFC2136 dynamic updates: `go test ./internal/server/dns -run TestDynamicUpdate_AddAndDelete -count=1`
  - Default TTL behavior: `go test ./internal/server/dns -run TestDynamicUpdate_DefaultTTLZero_NoOverride -count=1`
- All tests: `go test ./...`
- Tests use in-memory SQLite, сетевые сервисы не поднимаются.

Development
- Sync deps: `go mod tidy`
- Build only main: `go build ./cmd/smaillgeodns`
- Lint/format: follow project defaults (no external config added yet)

Config Reference
- `auto_soa_on_missing`: if true, при отсутствии SOA в зоне автоматически создаётся дефолтная запись SOA:
  - MNAME: `ns1.<zone>.`, RNAME: `hostmaster.<zone>.`
  - SERIAL: текущий Unix timestamp
  - Refresh/Retry/Expire/Minimum: 7200/3600/1209600/300
  - TTL: 3600
- `default_ttl`: TTL по умолчанию для записей/наборов, где TTL не указан (или равен 0). Используется в JSON/BIND импорте и при RFC2136 ADD.
