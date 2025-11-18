# Integration Tests / Интеграционные тесты

## English

This directory contains integration tests for namedot DNS server.

### DNS Record Types Integration Tests

**File:** `test-record-types.sh`

Full integration test for DNS record types that:
- Automatically starts the namedot server
- Creates test zones and records via REST API
- Tests A records (IPv4 addresses)
- Tests AAAA records (IPv6 addresses)
- Tests CNAME records (canonical names)
- Tests MX records (mail exchange)
- Tests TXT records (text data, SPF, DMARC)
- Validates DNS responses using `dig`
- Automatically stops the server and cleans up

**Requirements:**
- `dig` command (install: `sudo apt-get install dnsutils`)
- `namedot` binary (build: `make build`)

**Usage:**
```bash
# Run from project root
./test/integration/test-record-types.sh

# Or via Makefile
make test-integration-records
```

**Test Coverage:**
- A records (multiple IPv4 addresses)
- AAAA records (multiple IPv6 addresses)
- CNAME records (aliases)
- MX records (multiple mail servers with priorities)
- TXT records (DMARC, SPF policies)
- 9 test cases total

### GeoDNS Integration Tests

**File:** `test-geodns.sh`

Full integration test for GeoDNS functionality that:
- Automatically starts the namedot server
- Creates test zones and records via REST API
- Tests country-based routing (using GeoIP)
- Tests ASN-based routing
- Validates DNS responses using `dig` with `-b` flag (source IP simulation)
- Automatically stops the server and cleans up

**Requirements:**
- `dig` command (install: `sudo apt-get install dnsutils`)
- `namedot` binary (build: `make build`)
- Test MMDB databases in `geoipdb/` directory

**Usage:**
```bash
# Run from project root
./test/integration/test-geodns.sh

# Or via Makefile
make test-integration-geodns

# Run all integration tests
make test-integration
```

**Test Coverage:**
- Country-based GeoDNS routing (RU, GB, default)
- ASN-based routing (AS65001, AS65002, default)
- 12 test cases total

**Configuration:**
- Server config: `test-config.yaml`
- Test database: `test.db` (auto-cleaned after tests)
- GeoIP databases: `../../geoipdb/*.mmdb`

---

## Русский

Эта директория содержит интеграционные тесты для DNS сервера namedot.

### Интеграционные тесты типов DNS записей

**Файл:** `test-record-types.sh`

Полный интеграционный тест типов DNS записей, который:
- Автоматически запускает сервер namedot
- Создает тестовые зоны и записи через REST API
- Тестирует A записи (IPv4 адреса)
- Тестирует AAAA записи (IPv6 адреса)
- Тестирует CNAME записи (канонические имена)
- Тестирует MX записи (почтовые серверы)
- Тестирует TXT записи (текстовые данные, SPF, DMARC)
- Проверяет DNS ответы используя `dig`
- Автоматически останавливает сервер и очищает данные

**Требования:**
- Утилита `dig` (установка: `sudo apt-get install dnsutils`)
- Бинарник `namedot` (сборка: `make build`)

**Использование:**
```bash
# Запуск из корня проекта
./test/integration/test-record-types.sh

# Или через Makefile
make test-integration-records
```

**Покрытие тестами:**
- A записи (несколько IPv4 адресов)
- AAAA записи (несколько IPv6 адресов)
- CNAME записи (алиасы)
- MX записи (несколько почтовых серверов с приоритетами)
- TXT записи (DMARC, SPF политики)
- Всего 9 тестовых сценариев

### Интеграционные тесты GeoDNS

**Файл:** `test-geodns.sh`

Полный интеграционный тест функциональности GeoDNS, который:
- Автоматически запускает сервер namedot
- Создает тестовые зоны и записи через REST API
- Тестирует маршрутизацию по странам (через GeoIP)
- Тестирует маршрутизацию по ASN
- Проверяет DNS ответы используя `dig` с флагом `-b` (симуляция source IP)
- Автоматически останавливает сервер и очищает данные

**Требования:**
- Утилита `dig` (установка: `sudo apt-get install dnsutils`)
- Бинарник `namedot` (сборка: `make build`)
- Тестовые MMDB базы в директории `geoipdb/`

**Использование:**
```bash
# Запуск из корня проекта
./test/integration/test-geodns.sh

# Или через Makefile
make test-integration-geodns

# Запуск всех интеграционных тестов
make test-integration
```

**Покрытие тестами:**
- GeoDNS маршрутизация по странам (RU, GB, default)
- Маршрутизация по ASN (AS65001, AS65002, default)
- Всего 12 тестовых сценариев

**Конфигурация:**
- Конфиг сервера: `test-config.yaml`
- Тестовая БД: `test.db` (автоматически очищается после тестов)
- GeoIP базы: `../../geoipdb/*.mmdb`
