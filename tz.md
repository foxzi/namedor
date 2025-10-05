# DNS Server — Technical Specification (ТЗ)

## Overview
Лёгкий **DNS-сервер (UDP:53, TCP:53)** с хранением зон в БД, REST-API для управления DNS-записями, поддержкой **DNSSEC** и **GeoIP-ответов (MaxMind)**.  
Проект рассчитан на использование в корпоративной или DevOps-инфраструктуре, с возможностью развёртывания одной командой.

## 1. Цель
Реализовать минималистичный, производительный и расширяемый DNS-сервер:
- хранение зон в базе данных через ORM;
- управление записями через REST API;
- поддержка GeoDNS (MaxMind);
- DNSSEC (вкл./выкл. по флагу);
- кэширование, логирование, простая конфигурация.
- поддержка динамического обновления зон (RFC 2136).

## 2. Основные функции
- Обработка DNS-запросов по **UDP:53**, **TCP:53**.
- Ответы из БД; при отсутствии — форвардинг на внешний DNS.
- Поддерживаемые типы записей:
  **A, CNAME, MX, TXT, SRV, NS, SOA, SPF, DNSKEY, RRSIG, DS, CAA.**
- DNSSEC (включается флагом).
- GeoDNS: выбор ответа по IP клиента (MaxMind GeoLite2).
- Кэширование по TTL.
- Логирование запросов и ответов.
- Файл конфигурации (YAML).
- **Master-Slave репликация**: автоматическая синхронизация данных между серверами через REST API.

## 3. Хранение данных

**ORM** — для независимости от конкретной СУБД.  
Поддержка: **PostgreSQL / MySQL / SQLite**.

**Минимальная схема:**
```
zones(id, name UNIQUE)
rrsets(id, zone_id→zones, name, type, ttl)
rdata(
  id, rrset_id→rrsets, data TEXT/JSON,
  country CHAR(2)?,
  continent CHAR(2)?,
  asn INT?,
  subnet CIDR?
)
```
- `SOA.serial` — автоинкремент при изменениях зоны.  
- Поля `country`, `continent`, `asn`, `subnet` используются для GeoIP-ответов.  
- `rdata.data` — значение DNS-записи (RDATA).  

## 4. REST API

**Авторизация:** `Authorization: Bearer <token>`

###ones
| Метод | Путь | Описание |
|--------|------|----------|
| `POST /zones` | Создать зону |
| `GET /zones` / `/zones/{id}` | Список / получение зоны |
| `DELETE /zones/{id}` | Удалить зону и связанные RRsets |

###Rsets
| Метод | Путь | Описание |
|--------|------|----------|
| `POST /zones/{id}/rrsets` | Создать RRset |
| `PUT /zones/{id}/rrsets/{rid}` | Обновить RRset |
| `PATCH /zones/{id}/rrsets/{rid}` | Частично изменить |
| `DELETE /zones/{id}/rrsets/{rid}` | Удалить |
| `GET /zones/{id}/rrsets` | Список RRsets в зоне |

####ример
```json
{
  "name": "www",
  "type": "A",
  "ttl": 300,
  "records": [
    {"data": "192.0.2.10"},
    {"data": "198.51.100.10", "country": "DE"},
    {"data": "203.0.113.10", "continent": "EU"}
  ]
}
```

### Import / Export
| Метод | Путь | Описание |
|--------|------|----------|
| `GET /zones/{id}/export?format=bind\|json` | Экспорт зоны |
| `POST /zones/{id}/import?format=bind\|json&mode=upsert\|replace&serial=auto\|preserve` | Импорт зоны |

- Формат `bind` — текст зонфайла.
- Формат `json` — сериализованный список RRsets.
- При изменении зоны — автоматическое обновление `SOA.serial`.

### Replication (Master-Slave)
| Метод | Путь | Описание |
|--------|------|----------|
| `GET /sync/export` | Экспорт всех зон и шаблонов для репликации |
| `POST /sync/import` | Импорт данных на слейв-сервер |

- **Master сервер**: предоставляет данные через `/sync/export`
- **Slave сервер**: автоматически запрашивает данные с настраиваемым интервалом
- Синхронизируются: зоны, RRsets, записи, шаблоны
- Авторизация через Bearer token (тот же что и для API)  

## 5. Конфигурация
```yaml
listen: "0.0.0.0:53"
forwarder: "8.8.8.8"
enable_dnssec: true
api_token: "your-secure-token"
rest_listen: "0.0.0.0:8080"

db:
  driver: "postgres"
  dsn: "postgresql://user:pass@localhost/dns"

geoip:
  enabled: true
  mmdb_path: "/var/lib/maxmind/GeoLite2-City.mmdb"
  reload_sec: 300
  use_ecs: true

# Опционально: Master-Slave репликация
replication:
  mode: "master"  # или "slave", или не указывать
  # Для slave режима:
  master_url: "http://master-server:8080"
  sync_interval_sec: 60
  api_token: "your-secure-token"
```

**Replication режимы:**
- `mode: "master"` - сервер предоставляет данные для репликации
- `mode: "slave"` - сервер автоматически синхронизируется с мастером
- В slave режиме автоматически отключаются: веб-админка и DNS-обновления (read-only)
- Подробнее: [REPLICATION.md](REPLICATION.md)

## 6. Требования
- Язык реализации: **Go**.  
- Производительность: ≥ **1000 QPS** на локальных данных.  
- Простая установка (1 бинарь + конфиг).  
- Поддержка миграций БД.  
- Логи и базовые тесты.

## 7. Тестирование
- Юнит-тесты для всех компонентов.
- Интеграционные тесты для проверки взаимодействия между компонентами.
- Нагрузочные тесты для оценки производительности под высокой нагрузкой.
- Тестирование на соответствие спецификациям (например, RFC для DNS).

## 8. Утилиты
- Утилита для миграции данных между СУБД.
- Утилита для генерации тестовых данных.
- Утилита для мониторинга и отладки DNS-запросов.
- Утилита для работы с DNS-записями (например, bulk update).

## 9. Документация
- Документация по API (Swagger/OpenAPI).
- Документация по конфигурации (YAML).
- Документация по миграции данных.
- Документация по утилитам.
- Документация по тестированию.
