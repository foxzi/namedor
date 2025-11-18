# Резервное копирование и восстановление зон

Это руководство объясняет, как создавать резервные копии и восстанавливать DNS-зоны с помощью API namedot.

## Содержание

- [Форматы экспорта](#форматы-экспорта)
- [Режимы импорта](#режимы-импорта)
- [API эндпоинты](#api-эндпоинты)
- [Скрипты для бэкапа](#скрипты-для-бэкапа)
- [Скрипты для восстановления](#скрипты-для-восстановления)
- [Сценарий обновления](#сценарий-обновления)

## Форматы экспорта

namedot поддерживает два формата экспорта:

- **JSON** - Структурированный формат с полной поддержкой geo-aware записей
- **BIND** - Стандартный формат BIND zone file для совместимости

## Режимы импорта

При импорте зон можно выбрать один из двух режимов:

- **upsert** (по умолчанию) - Добавить новые записи и обновить существующие
- **replace** - Удалить все существующие записи и импортировать новые

## API эндпоинты

### Экспорт одной зоны

**JSON формат:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/zones/{id}/export?format=json"
```

**BIND формат:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/zones/{id}/export?format=bind"
```

### Экспорт всех зон (для репликации)

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:7070/sync/export"
```

### Импорт зоны

**JSON формат:**
```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @backup.json \
  "http://localhost:7070/zones/{id}/import?format=json&mode=upsert"
```

**BIND формат:**
```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: text/plain" \
  --data-binary @backup.zone \
  "http://localhost:7070/zones/{id}/import?format=bind&mode=upsert"
```

## Скрипты для бэкапа

### Бэкап всех зон

Создайте скрипт `backup-zones.sh`:

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
BACKUP_DIR="./backup-$(date +%Y%m%d-%H%M%S)"

if [ -z "$TOKEN" ]; then
  echo "Ошибка: требуется переменная окружения TOKEN"
  echo "Использование: TOKEN=your-token $0"
  exit 1
fi

mkdir -p "$BACKUP_DIR"

echo "Получение списка зон..."
ZONES=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE/zones" | \
  python3 -c "import sys,json; [print(z['id']) for z in json.load(sys.stdin)]" 2>/dev/null)

if [ -z "$ZONES" ]; then
  echo "Ошибка: не удалось получить список зон. Проверьте токен и API endpoint."
  exit 1
fi

echo "Экспорт зон..."
for zone_id in $ZONES; do
  curl -s -H "Authorization: Bearer $TOKEN" \
    "$API_BASE/zones/$zone_id/export?format=json" > "$BACKUP_DIR/zone-$zone_id.json"

  if [ $? -eq 0 ]; then
    zone_name=$(python3 -c "import json; print(json.load(open('$BACKUP_DIR/zone-$zone_id.json'))['name'])" 2>/dev/null)
    echo "  Экспортирована зона $zone_id: $zone_name"
  else
    echo "  Не удалось экспортировать зону $zone_id"
  fi
done

echo ""
echo "Бэкап завершен: $BACKUP_DIR"
echo "Всего зон: $(ls -1 $BACKUP_DIR/zone-*.json 2>/dev/null | wc -l)"
```

Использование:
```bash
chmod +x backup-zones.sh
TOKEN="your-api-token" ./backup-zones.sh
```

### Бэкап одной зоны

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
ZONE_ID="$1"

if [ -z "$TOKEN" ] || [ -z "$ZONE_ID" ]; then
  echo "Использование: TOKEN=your-token $0 <zone_id>"
  exit 1
fi

curl -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/zones/$ZONE_ID/export?format=json" > "zone-$ZONE_ID-$(date +%Y%m%d-%H%M%S).json"

echo "Зона $ZONE_ID сохранена в бэкап"
```

## Скрипты для восстановления

### Восстановление всех зон

Создайте скрипт `restore-zones.sh`:

```bash
#!/bin/bash

API_BASE="${API_BASE:-http://localhost:7070}"
TOKEN="${TOKEN:-}"
BACKUP_DIR="$1"

if [ -z "$TOKEN" ] || [ -z "$BACKUP_DIR" ]; then
  echo "Использование: TOKEN=your-token $0 <директория_с_бэкапом>"
  exit 1
fi

if [ ! -d "$BACKUP_DIR" ]; then
  echo "Ошибка: директория с бэкапом не найдена: $BACKUP_DIR"
  exit 1
fi

echo "Восстановление зон из $BACKUP_DIR..."

for backup_file in "$BACKUP_DIR"/zone-*.json; do
  if [ ! -f "$backup_file" ]; then
    echo "Файлы бэкапа не найдены"
    exit 1
  fi

  # Извлечь имя зоны из файла бэкапа
  zone_name=$(python3 -c "import json; print(json.load(open('$backup_file'))['name'])" 2>/dev/null)

  if [ -z "$zone_name" ]; then
    echo "  Пропуск некорректного файла бэкапа: $backup_file"
    continue
  fi

  echo "  Обработка зоны: $zone_name"

  # Проверить, существует ли зона
  existing_zone=$(curl -s -H "Authorization: Bearer $TOKEN" "$API_BASE/zones" | \
    python3 -c "import sys,json; zones=[z for z in json.load(sys.stdin) if z['name']=='$zone_name']; print(zones[0]['id'] if zones else '')" 2>/dev/null)

  if [ -n "$existing_zone" ]; then
    # Зона существует, импорт в существующую зону
    zone_id="$existing_zone"
    echo "    Зона существует (ID: $zone_id), импорт записей..."
  else
    # Создать новую зону
    zone_id=$(curl -s -X POST \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"$zone_name\"}" \
      "$API_BASE/zones" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$zone_id" ]; then
      echo "    Не удалось создать зону"
      continue
    fi
    echo "    Создана зона (ID: $zone_id)"
  fi

  # Импорт записей
  response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$backup_file" \
    "$API_BASE/zones/$zone_id/import?format=json&mode=replace")

  http_code=$(echo "$response" | tail -n1)

  if [ "$http_code" = "204" ]; then
    echo "    Успешно восстановлено"
  else
    echo "    Не удалось импортировать записи (HTTP $http_code)"
  fi
done

echo ""
echo "Восстановление завершено"
```

Использование:
```bash
chmod +x restore-zones.sh
TOKEN="your-api-token" ./restore-zones.sh ./backup-20241118-230000
```

## Сценарий обновления

При обновлении namedot до новой версии (особенно с изменениями схемы БД):

### Шаг 1: Бэкап текущих данных

```bash
# Остановить текущий сервис namedot
sudo systemctl stop namedot

# Создать бэкап
TOKEN="your-api-token" ./backup-zones.sh
```

### Шаг 2: Обновление namedot

```bash
# Установить новую версию
sudo dpkg -i namedot_0.2.0_amd64.deb
# или
sudo rpm -U namedot-0.2.0.x86_64.rpm
```

### Шаг 3: Инициализация новой БД

Новая версия автоматически создаст свежую БД с обновленной схемой при первом запуске.

```bash
# Запустить namedot
sudo systemctl start namedot

# Проверить статус
sudo systemctl status namedot
```

### Шаг 4: Восстановление данных

```bash
# Восстановить все зоны из бэкапа
TOKEN="your-api-token" ./restore-zones.sh ./backup-20241118-230000
```

### Шаг 5: Проверка

```bash
# Проверить, что все зоны восстановлены
curl -H "Authorization: Bearer $TOKEN" http://localhost:7070/zones

# Проверить DNS резолвинг
dig @localhost -p 5353 example.com A
```

## Примечания

- Бэкапы включают все данные зон: DNS записи, TTL и правила geo-aware маршрутизации
- Эндпоинт `sync/export` экспортирует все зоны в один JSON файл (полезно для репликации)
- Всегда тестируйте процедуры восстановления на тестовой системе
- Храните несколько копий бэкапов перед важными обновлениями
- Файлы бэкапов в формате JSON можно редактировать вручную при необходимости
