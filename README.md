SmaillGeoDNS — Lightweight DNS server with REST + GeoDNS

Overview
- UDP/TCP DNS on :53
- Zones and records in DB (GORM: Postgres/MySQL/SQLite)
- REST API for zone management (+ JSON/BIND export, JSON import)
- Geo-aware responses (subnet/country/continent), ECS support
- Optional forwarder for cache-miss
- Simple in-memory TTL cache
- Master-Slave replication via REST API

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

Replication
- Master-Slave replication via REST API with automatic sync
- See [REPLICATION.md](REPLICATION.md) for setup and configuration
- Example configs: [examples/config.master.yaml](examples/config.master.yaml) and [examples/config.slave.yaml](examples/config.slave.yaml)

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

---

# Русская версия / Russian Version

SmaillGeoDNS — Легковесный DNS-сервер с REST API + GeoDNS

## Обзор
- UDP/TCP DNS на порту :53
- Зоны и записи в БД (GORM: Postgres/MySQL/SQLite)
- REST API для управления зонами (+ JSON/BIND экспорт, JSON импорт)
- Geo-aware ответы (подсеть/страна/континент), поддержка ECS
- Опциональный форвардер при отсутствии записи в кеше
- Простой in-memory TTL кеш
- Master-Slave репликация через REST API

## Требования
- Go >= 1.23 (рекомендуется 1.24+)

## Быстрый старт

1) Создайте `config.yaml` в корне репозитория:

```yaml
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

2) Сборка и запуск:
- `go build ./cmd/smaillgeodns`
- `sudo ./smaillgeodns` (DNS на :53 требует привилегий или проброса порта)

## REST API (Bearer devtoken)
- Создать зону: `POST /zones {"name":"example.com"}`
- Добавить rrset: `POST /zones/{id}/rrsets` с телом аналогичным tz.md
- Экспорт: `GET /zones/{id}/export?format=json|bind`
- Импорт: `POST /zones/{id}/import?format=json&mode=upsert|replace`

## Репликация
- Master-Slave репликация через REST API с автоматической синхронизацией
- См. [REPLICATION.md](REPLICATION.md) для настройки и конфигурации
- Примеры конфигов: [examples/config.master.yaml](examples/config.master.yaml) и [examples/config.slave.yaml](examples/config.slave.yaml)

## Примечания
- Динамическая подпись DNSSEC пока не реализована. Вы можете хранить DNSSEC-записи (DNSKEY/RRSIG/DS) в БД и отдавать их как есть при запросе.
- Geo-выбор в настоящее время поддерживает атрибуты subnet/country/continent на записях. ASN требует интеграции GeoIP DB и находится в TODO.

## GeoIP
- Включить в конфиге:
  - `geoip.enabled: true`
  - `geoip.mmdb_path: <путь к .mmdb файлу или директории>`
  - `geoip.use_ecs: true` для учета EDNS Client Subnet
- Именование файлов: сервер сканирует директорию и определяет тип БД по метаданным; вы можете именовать файлы, например, `GeoLite2-City.mmdb`, `GeoLite2-ASN.mmdb`, или `city-ipv4.mmdb`, `city-ipv6.mmdb`, `asn-ipv4.mmdb`, `asn-ipv6.mmdb`. Единый City файл применяется и к IPv4, и к IPv6.
- Логи: при запуске сервер логирует, какие GeoIP БД загружены; если ни одна не найдена или нечитаема, он логирует ошибку и отключает GeoDNS.

## Динамические обновления (RFC 2136)
- Включить через конфиг `update.enabled: true`. Опционально принудительно использовать TSIG: `update.require_tsig: true`.
- Настройте `update.tsig_secrets` как map имени ключа к base64 HMAC секрету. Пример в `examples/config.yaml`.
- DNS-сервер обрабатывает базовые операции add/delete и увеличивает SOA serial.

## BIND импорт
- REST: `POST /zones/{id}/import?format=bind&mode=upsert|replace` с сырым текстом зоны в теле.
- Экспорт остаётся доступен через `GET /zones/{id}/export?format=bind`.

## Тестирование
- Модульные тесты (модули):
  - BIND импорт/экспорт: `go test ./internal/server/rest/zoneio -run TestImportBIND_And_ToBind -count=1`
  - RFC2136 динамические обновления: `go test ./internal/server/dns -run TestDynamicUpdate_AddAndDelete -count=1`
  - Поведение TTL по умолчанию: `go test ./internal/server/dns -run TestDynamicUpdate_DefaultTTLZero_NoOverride -count=1`
- Все тесты: `go test ./...`
- Тесты используют in-memory SQLite, сетевые сервисы не поднимаются.

## Интеграционные тесты
- End-to-end (REST + DNS):
  - `go test ./internal/integration -run TestEndToEnd_DNS_and_REST -count=1`
  - Под капотом: поднимает DNS на 19053 и REST на 18089, создаёт зону и A-запись через REST, затем делает DNS-запрос и проверяет ответ, включая повторный запрос (кэш).
- GeoDNS (требует ./geoipdb с .mmdb файлами):
  - Subnet/ECS выбор: `go test ./internal/integration -run TestGeoDNS_WithECS_USCountry -count=1`
  - Country/Continent/ASN выбор (авто-пропуск при отсутствии данных): `go test ./internal/integration -run TestGeoDNS_WithECS_Country_Continent_ASN -count=1`

## GeoIP базы данных
- Репозиторий содержит небольшие MMDB для локальных тестов в `./geoipdb`:
  - IPv4 localhost диапазоны: `city-localhost.mmdb`, `asn-localhost.mmdb` (127.0.1.0/24 → RU/EU/AS65001, 127.0.2.0/24 → GB/EU/AS65002).
  - IPv6 документационные диапазоны: `city-localhost6.mmdb`, `asn-localhost6.mmdb` (2001:db8:1::/64 и 2001:db8:2::/64) для ECS IPv6 тестов.
- Проверка с помощью `mmdblookup`, например:
  - `mmdblookup --file geoipdb/city-localhost.mmdb --ip 127.0.1.10`
  - `mmdblookup --file geoipdb/asn-localhost6.mmdb --ip 2001:db8:1::1`

## Разработка
- Синхронизация зависимостей: `go mod tidy`
- Сборка только main: `go build ./cmd/smaillgeodns`
- Линтинг/форматирование: следуйте дефолтным настройкам проекта (внешний конфиг пока не добавлен)

## Makefile
- Сборка сервера: `make build`
- Запуск с конфигом: `make run CFG=config.yaml`
- Все тесты: `make test-all`
- Модульные + интеграционные: `make test`
- GeoDNS тесты: `make test-geo`

## Справка по конфигурации
- `auto_soa_on_missing`: если true, при отсутствии SOA в зоне автоматически создаётся дефолтная запись SOA:
  - MNAME: `ns1.<zone>.`, RNAME: `hostmaster.<zone>.`
  - SERIAL: текущий Unix timestamp
  - Refresh/Retry/Expire/Minimum: 7200/3600/1209600/300
  - TTL: 3600
- `default_ttl`: TTL по умолчанию для записей/наборов, где TTL не указан (или равен 0). Используется в JSON/BIND импорте и при RFC2136 ADD.
