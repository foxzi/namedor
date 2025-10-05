# Web Admin Panel

GeoDNS includes a built-in web-based admin panel for managing DNS zones and records with GeoIP support.

## Features

- **Zone Management**: Create, view, and delete DNS zones
- **DNS Records**: Full CRUD for A, AAAA, CNAME, MX, TXT, NS records
- **GeoIP Support**: Configure geo-routing by Country, Continent, ASN, or Subnet
- **Session-based Auth**: Secure login with bcrypt password hashing
- **HTMX Interface**: Fast, interactive UI without JavaScript frameworks
- **Easy Configuration**: Enable/disable via config file

## Quick Start

### 1. Generate Password Hash

```bash
go run cmd/hashpwd/main.go yourPassword
# or using the main binary
./namedot --password yourPassword
```

Example output:
```
Bcrypt hash for 'yourPassword':
$2a$10$abc123...xyz789

Add this to your config.yaml:
admin:
  enabled: true
  username: admin
  password_hash: "$2a$10$abc123...xyz789"
```

### 2. Update Configuration

Add to your `config.yaml`:

```yaml
admin:
  enabled: true
  username: admin
  password_hash: "$2a$10$0WB2kBhwpbU9.nmxVD2qs.4.1cz9vxrI8Vd58X7arA1rzp57B5zfW"
```

**Security Note**: The example hash above is for password "admin". **Change this in production!**

### 3. Start Server

```bash
./namedot
```

### 4. Access Admin Panel

Open browser: `http://localhost:18080/admin`

Default credentials:
- Username: `admin`
- Password: `admin` (if using example hash)

## Using the Admin Panel

### Managing Zones

1. **Create Zone**: Click "+ New Zone" button
2. **Enter zone name**: e.g., `example.com`
3. **View Records**: Click "View Records" for any zone
4. **Delete Zone**: Click "Delete" (confirms before deleting)

### Managing DNS Records

1. **Navigate to zone**: Click "View Records" on a zone
2. **Add Record**: Click "+ Add Record"
3. **Fill in details**:
   - **Name**: Record name (e.g., `www`, `mail`)
   - **Type**: A, AAAA, CNAME, MX, TXT, or NS
   - **TTL**: Time to live in seconds (default: 300)
   - **Data**: IP address or record value

### GeoIP Targeting

When adding a record, optionally specify geo-targeting:

**Country-based routing:**
```
Country Code: RU
Data: 192.0.2.10
```

**Continent-based routing:**
```
Continent Code: EU
Data: 203.0.113.10
```

**ASN-based routing:**
```
ASN: 65001
Data: 198.51.100.10
```

**Subnet-based routing:**
```
Subnet: 10.0.0.0/8
Data: 192.0.2.20
```

**Priority**: Country > Continent > ASN > Subnet > Default

## Configuration Options

```yaml
admin:
  enabled: true                    # Enable/disable admin panel
  username: admin                  # Admin username
  password_hash: "$2a$10$..."     # Bcrypt hash of password
```

### Disable Admin Panel

Set `admin.enabled: false` in config to completely disable the web UI.

**Note**: Admin panel is **automatically disabled** when server runs in slave replication mode (`replication.mode: "slave"`). This prevents direct modifications on slave servers and ensures read-only operation. See [REPLICATION.md](REPLICATION.md) for details.

## Security Best Practices

1. **Strong Password**: Use a strong, unique password
   ```bash
   go run cmd/hashpwd/main.go "MyStr0ng!P@ssw0rd"
   ```

2. **HTTPS**: Use a reverse proxy (nginx, Caddy) with TLS:
   ```nginx
   server {
       listen 443 ssl;
       server_name geodns.example.com;

       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;

       location / {
           proxy_pass http://localhost:18080;
           proxy_set_header Host $host;
       }
   }
   ```

3. **Firewall**: Restrict access to admin panel:
   ```bash
   # Allow only from specific IP
   iptables -A INPUT -p tcp --dport 18080 -s 192.168.1.0/24 -j ACCEPT
   iptables -A INPUT -p tcp --dport 18080 -j DROP
   ```

4. **VPN/Bastion**: Access admin panel only via VPN or bastion host

## Session Management

- **Session Duration**: 24 hours
- **Cookie Name**: `session`
- **Cookie Attributes**: HttpOnly (prevents XSS)
- **Auto Logout**: Sessions expire after 24h

To logout manually: Click "Logout" in navigation bar

## Troubleshooting

### Cannot login

1. Check password hash is correct:
   ```bash
   go run cmd/hashpwd/main.go yourPassword
   ```

2. Verify `admin.enabled: true` in config

3. Check server logs for errors

### Admin panel not loading

1. Verify templates exist:
   ```bash
   ls internal/web/templates/
   # Should show: dashboard.html, login.html
   ```

2. Check REST API is running:
   ```bash
   curl http://localhost:18080/health
   ```

3. Review server logs on startup:
   ```
   Web admin panel enabled at /admin
   ```

### Session expired immediately

- Check system time is correct (session uses server time)
- Ensure cookies are enabled in browser
- Try clearing browser cookies

## API Integration

The admin panel uses the existing REST API. You can also manage zones via API:

```bash
# List zones
curl -H "Authorization: Bearer your-api-token" \
  http://localhost:18080/api/zones

# Create zone
curl -X POST -H "Authorization: Bearer your-api-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}' \
  http://localhost:18080/api/zones
```

See main README for full API documentation.

## Development

Built with:
- **Backend**: Gin (Go web framework)
- **Frontend**: HTMX (dynamic HTML interactions)
- **Auth**: bcrypt (password hashing)
- **Sessions**: In-memory (cookie-based)

To add custom features, modify files in `internal/web/`:
- `admin.go` - Core admin logic, authentication
- `zones.go` - Zone management handlers
- `records.go` - DNS record handlers
- `templates/*.html` - UI templates

---

# Русская версия / Russian Version

# Веб-панель администратора

GeoDNS включает встроенную веб-панель администрирования для управления DNS-зонами и записями с поддержкой GeoIP.

## Возможности

- **Управление зонами**: Создание, просмотр и удаление DNS-зон
- **DNS записи**: Полный CRUD для записей A, AAAA, CNAME, MX, TXT, NS
- **Поддержка GeoIP**: Настройка гео-маршрутизации по стране, континенту, ASN или подсети
- **Аутентификация на основе сессий**: Безопасный вход с хешированием паролей bcrypt
- **HTMX интерфейс**: Быстрый, интерактивный UI без JavaScript-фреймворков
- **Простая настройка**: Включение/отключение через конфигурационный файл

## Быстрый старт

### 1. Генерация хеша пароля

```bash
go run cmd/hashpwd/main.go yourPassword
# или через основной бинарник
./namedot --password yourPassword
```

Пример вывода:
```
Bcrypt hash for 'yourPassword':
$2a$10$abc123...xyz789

Add this to your config.yaml:
admin:
  enabled: true
  username: admin
  password_hash: "$2a$10$abc123...xyz789"
```

### 2. Обновление конфигурации

Добавьте в ваш `config.yaml`:

```yaml
admin:
  enabled: true
  username: admin
  password_hash: "$2a$10$0WB2kBhwpbU9.nmxVD2qs.4.1cz9vxrI8Vd58X7arA1rzp57B5zfW"
```

**Примечание по безопасности**: Приведенный выше пример хеша для пароля "admin". **Измените это в продакшене!**

### 3. Запуск сервера

```bash
./namedot
```

### 4. Доступ к панели администратора

Откройте в браузере: `http://localhost:18080/admin`

Учетные данные по умолчанию:
- Имя пользователя: `admin`
- Пароль: `admin` (если используется пример хеша)

## Использование панели администратора

### Управление зонами

1. **Создать зону**: Нажмите кнопку "+ New Zone"
2. **Введите имя зоны**: например, `example.com`
3. **Просмотр записей**: Нажмите "View Records" для любой зоны
4. **Удалить зону**: Нажмите "Delete" (запрашивает подтверждение перед удалением)

### Управление DNS-записями

1. **Перейти к зоне**: Нажмите "View Records" на зоне
2. **Добавить запись**: Нажмите "+ Add Record"
3. **Заполните детали**:
   - **Name**: Имя записи (например, `www`, `mail`)
   - **Type**: A, AAAA, CNAME, MX, TXT или NS
   - **TTL**: Время жизни в секундах (по умолчанию: 300)
   - **Data**: IP-адрес или значение записи

### GeoIP таргетинг

При добавлении записи можно опционально указать гео-таргетинг:

**Маршрутизация по стране:**
```
Country Code: RU
Data: 192.0.2.10
```

**Маршрутизация по континенту:**
```
Continent Code: EU
Data: 203.0.113.10
```

**Маршрутизация по ASN:**
```
ASN: 65001
Data: 198.51.100.10
```

**Маршрутизация по подсети:**
```
Subnet: 10.0.0.0/8
Data: 192.0.2.20
```

**Приоритет**: Страна > Континент > ASN > Подсеть > По умолчанию

## Параметры конфигурации

```yaml
admin:
  enabled: true                    # Включить/отключить панель администратора
  username: admin                  # Имя пользователя администратора
  password_hash: "$2a$10$..."     # Bcrypt хеш пароля
```

### Отключение панели администратора

Установите `admin.enabled: false` в конфиге для полного отключения веб-интерфейса.

**Примечание**: Панель администратора **автоматически отключается**, когда сервер работает в режиме слейв-репликации (`replication.mode: "slave"`). Это предотвращает прямые изменения на слейв-серверах и обеспечивает работу только для чтения. См. [REPLICATION.md](REPLICATION.md) для подробностей.

## Рекомендации по безопасности

1. **Надежный пароль**: Используйте надежный, уникальный пароль
   ```bash
   go run cmd/hashpwd/main.go "MyStr0ng!P@ssw0rd"
   ```

2. **HTTPS**: Используйте reverse proxy (nginx, Caddy) с TLS:
   ```nginx
   server {
       listen 443 ssl;
       server_name geodns.example.com;

       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;

       location / {
           proxy_pass http://localhost:18080;
           proxy_set_header Host $host;
       }
   }
   ```

3. **Firewall**: Ограничьте доступ к панели администратора:
   ```bash
   # Разрешить только с конкретного IP
   iptables -A INPUT -p tcp --dport 18080 -s 192.168.1.0/24 -j ACCEPT
   iptables -A INPUT -p tcp --dport 18080 -j DROP
   ```

4. **VPN/Bastion**: Доступ к панели администратора только через VPN или bastion-хост

## Управление сессиями

- **Длительность сессии**: 24 часа
- **Имя cookie**: `session`
- **Атрибуты cookie**: HttpOnly (предотвращает XSS)
- **Автоматический выход**: Сессии истекают через 24 часа

Для ручного выхода: Нажмите "Logout" в навигационной панели

## Устранение неполадок

### Не удается войти

1. Проверьте правильность хеша пароля:
   ```bash
   go run cmd/hashpwd/main.go yourPassword
   ```

2. Убедитесь что `admin.enabled: true` в конфиге

3. Проверьте логи сервера на наличие ошибок

### Панель администратора не загружается

1. Убедитесь что шаблоны существуют:
   ```bash
   ls internal/web/templates/
   # Должно показать: dashboard.html, login.html
   ```

2. Проверьте что REST API работает:
   ```bash
   curl http://localhost:18080/health
   ```

3. Проверьте логи сервера при запуске:
   ```
   Web admin panel enabled at /admin
   ```

### Сессия немедленно истекает

- Проверьте что системное время правильное (сессия использует время сервера)
- Убедитесь что cookies включены в браузере
- Попробуйте очистить cookies браузера

## Интеграция с API

Панель администратора использует существующий REST API. Вы также можете управлять зонами через API:

```bash
# Список зон
curl -H "Authorization: Bearer your-api-token" \
  http://localhost:18080/api/zones

# Создать зону
curl -X POST -H "Authorization: Bearer your-api-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}' \
  http://localhost:18080/api/zones
```

См. основной README для полной документации API.

## Разработка

Построено с помощью:
- **Backend**: Gin (Go веб-фреймворк)
- **Frontend**: HTMX (динамические HTML-взаимодействия)
- **Auth**: bcrypt (хеширование паролей)
- **Sessions**: In-memory (на основе cookies)

Для добавления пользовательских функций измените файлы в `internal/web/`:
- `admin.go` - Основная логика администрирования, аутентификация
- `zones.go` - Обработчики управления зонами
- `records.go` - Обработчики DNS-записей
- `templates/*.html` - UI шаблоны
