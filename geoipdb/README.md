GeoIP Test Databases / Тестовые базы GeoIP

English

- Purpose: Tiny MMDB files for local GeoDNS and ECS testing; used by unit/integration tests. Not real geolocation data.
- Files:
  - `city-localhost.mmdb` (IPv4):
    - `127.0.1.0/24` → country `RU`, continent `EU`
    - `127.0.2.0/24` → country `GB`, continent `EU`
  - `asn-localhost.mmdb` (IPv4):
    - `127.0.1.0/24` → `AS65001`
    - `127.0.2.0/24` → `AS65002`
  - `city-localhost6.mmdb` (IPv6):
    - `2001:db8:1::/64` → country `RU`, continent `EU`
    - `2001:db8:2::/64` → country `GB`, continent `EU`
  - `asn-localhost6.mmdb` (IPv6):
    - `2001:db8:1::/64` → `AS65101`
    - `2001:db8:2::/64` → `AS65102`
- Usage:
  - Point server config `geoip.mmdb_path` to this directory.
  - Quick check with `mmdblookup`:
    - `mmdblookup --file city-localhost.mmdb --ip 127.0.1.10 country iso_code`
    - `mmdblookup --file asn-localhost.mmdb --ip 127.0.2.10 autonomous_system_number`
    - `mmdblookup --file city-localhost6.mmdb --ip 2001:db8:1::1 continent code`
- Notes:
  - These files are minimal and only for tests; replace with real MMDBs (GeoLite2/DBIP) for production.

Русский

- Назначение: Маленькие MMDB-файлы для локальных тестов GeoDNS и ECS; используются в модульных/интеграционных тестах. Не являются реальными геоданными.
- Файлы:
  - `city-localhost.mmdb` (IPv4):
    - `127.0.1.0/24` → страна `RU`, континент `EU`
    - `127.0.2.0/24` → страна `GB`, континент `EU`
  - `asn-localhost.mmdb` (IPv4):
    - `127.0.1.0/24` → `AS65001`
    - `127.0.2.0/24` → `AS65002`
  - `city-localhost6.mmdb` (IPv6):
    - `2001:db8:1::/64` → страна `RU`, континент `EU`
    - `2001:db8:2::/64` → страна `GB`, континент `EU`
  - `asn-localhost6.mmdb` (IPv6):
    - `2001:db8:1::/64` → `AS65101`
    - `2001:db8:2::/64` → `AS65102`
- Использование:
  - Укажите `geoip.mmdb_path` на эту директорию в конфиге сервера.
  - Быстрая проверка с `mmdblookup`:
    - `mmdblookup --file city-localhost.mmdb --ip 127.0.1.10 country iso_code`
    - `mmdblookup --file asn-localhost.mmdb --ip 127.0.2.10 autonomous_system_number`
    - `mmdblookup --file city-localhost6.mmdb --ip 2001:db8:1::1 continent code`
- Примечания:
  - Файлы минимальные и предназначены только для тестов; для продакшена замените на реальные MMDB (GeoLite2/DBIP).

