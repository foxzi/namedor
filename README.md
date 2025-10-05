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

log:
  dns_verbose: true
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

GeoIP
- Enable in config:
  - `geoip.enabled: true`
  - `geoip.mmdb_path: <path to .mmdb file or directory>`
  - `geoip.use_ecs: true` to honor EDNS Client Subnet
- File naming: server scans directory and detects DB type by metadata; you can name files e.g. `GeoLite2-City.mmdb`, `GeoLite2-ASN.mmdb`, or `city-ipv4.mmdb`, `city-ipv6.mmdb`, `asn-ipv4.mmdb`, `asn-ipv6.mmdb`. A single City file is applied to both IPv4/IPv6.
- Logs: on startup, server logs which GeoIP DBs are loaded; if none found or unreadable, it logs an error and disables GeoDNS.

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

Integration Tests
- End-to-end (REST + DNS):
  - `go test ./internal/integration -run TestEndToEnd_DNS_and_REST -count=1`
  - Под капотом: поднимает DNS на 19053 и REST на 18089, создаёт зону и A-запись через REST, затем делает DNS-запрос и проверяет ответ, включая повторный запрос (кэш).
- GeoDNS (requires ./geoipdb with .mmdb files):
  - Subnet/ECS selection: `go test ./internal/integration -run TestGeoDNS_WithECS_USCountry -count=1`
  - Country/Continent/ASN selection (auto-skips if data missing): `go test ./internal/integration -run TestGeoDNS_WithECS_Country_Continent_ASN -count=1`

GeoIP Databases
- Repo ships small MMDBs for local tests in `./geoipdb`:
  - IPv4 localhost ranges: `city-localhost.mmdb`, `asn-localhost.mmdb` (127.0.1.0/24 → RU/EU/AS65001, 127.0.2.0/24 → GB/EU/AS65002).
  - IPv6 documentation ranges: `city-localhost6.mmdb`, `asn-localhost6.mmdb` (2001:db8:1::/64 and 2001:db8:2::/64) for ECS IPv6 tests.
- Verify with `mmdblookup`, e.g.:
  - `mmdblookup --file geoipdb/city-localhost.mmdb --ip 127.0.1.10`
  - `mmdblookup --file geoipdb/asn-localhost6.mmdb --ip 2001:db8:1::1`

Development
- Sync deps: `go mod tidy`
- Build only main: `go build ./cmd/smaillgeodns`
- Lint/format: follow project defaults (no external config added yet)

Makefile
- Build server: `make build`
- Run with config: `make run CFG=config.yaml`
- All tests: `make test-all`
- Unit + integration: `make test`
- GeoDNS tests: `make test-geo`
  
  

Config Reference
- `auto_soa_on_missing`: if true, при отсутствии SOA в зоне автоматически создаётся дефолтная запись SOA:
  - MNAME: `ns1.<zone>.`, RNAME: `hostmaster.<zone>.`
  - SERIAL: текущий Unix timestamp
  - Refresh/Retry/Expire/Minimum: 7200/3600/1209600/300
  - TTL: 3600
- `default_ttl`: TTL по умолчанию для записей/наборов, где TTL не указан (или равен 0). Используется в JSON/BIND импорте и при RFC2136 ADD.
