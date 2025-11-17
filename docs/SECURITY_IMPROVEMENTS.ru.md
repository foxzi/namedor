# Улучшения безопасности веб-панели администратора

## Обзор
Этот документ описывает улучшения безопасности, реализованные для устранения CSRF уязвимостей и небезопасной обработки cookie в веб-панели администратора.

## Внесенные изменения

### 1. Защита от CSRF
- **Генерация CSRF токенов**: Каждая сессия генерирует уникальный CSRF токен
- **Валидация токенов**: Все изменяющие запросы (POST/PUT/DELETE) проверяют CSRF токены
- **Передача токенов**: CSRF токены отправляются через:
  - HTTP заголовок: `X-CSRF-Token` (для HTMX запросов)
  - Поле формы: `csrf_token` (для обычных форм)

### 2. Безопасная конфигурация Cookie
Все cookie теперь используют защитные флаги:
- **Secure**: `true` когда включен TLS (cookie отправляются только через HTTPS)
- **SameSite**: `Strict` (максимальная защита от CSRF)
- **HttpOnly**: `true` (защита от XSS)

### 3. Валидация Origin/Referer
Дополнительная защита по принципу эшелонированной обороны:
- Проверяет, что заголовок `Origin` совпадает с хостом сервера
- Проверяет, что заголовок `Referer` начинается с хоста сервера
- Отклоняет запросы без обоих заголовков

## Технические детали

### Измененные файлы
- `internal/web/admin.go`:
  - Добавлено поле `CSRFToken` в структуру `Session`
  - Создан `csrfMiddleware()` для валидации токенов
  - Создан `validateOrigin()` для проверки Origin/Referer
  - Создан `setSecureCookie()` для безопасной установки cookie
  - Обновлены все вызовы установки cookie на безопасную конфигурацию

- `internal/web/templates/dashboard.html`:
  - Добавлен тег `<meta name="csrf-token">`
  - Настроен HTMX для автоматической отправки CSRF токена со всеми запросами

### Защиты безопасности

#### До
- ❌ Нет защиты от CSRF
- ❌ Cookie без флага Secure (передаются по HTTP)
- ❌ Cookie без SameSite (уязвимы к CSRF)
- ❌ Нет валидации Origin/Referer

#### После
- ✅ CSRF токены для всех изменяющих операций
- ✅ Безопасные cookie (только HTTPS когда включен TLS)
- ✅ SameSite=Strict (нет кросс-сайтовой передачи cookie)
- ✅ Валидация Origin/Referer (эшелонированная защита)

## Предотвращенные сценарии атак

### 1. CSRF атака
**До**: Атакующий мог создать вредоносный сайт с кодом:
```html
<form action="https://your-dns-server.com/admin/zones/delete/1" method="POST">
  <input type="submit" value="Нажмите для получения приза!">
</form>
```
Если администратор залогинен и нажмет, зона будет удалена.

**После**: Запрос блокируется из-за:
- Отсутствующего/невалидного CSRF токена → 403 Forbidden
- Невалидного Origin/Referer → 403 Forbidden

### 2. Кража cookie через прослушивание сети
**До**: Сессионные cookie, передаваемые по HTTP, могли быть перехвачены

**После**: Cookie передаются только по HTTPS (когда включен TLS)

### 3. Кросс-сайтовая передача cookie
**До**: Cookie отправлялись со всеми запросами, даже с вредоносных сайтов

**После**: SameSite=Strict предотвращает передачу cookie на ваш домен с внешних сайтов

## Требования к конфигурации

### Настоятельно рекомендуется HTTPS
Для максимальной безопасности настройте TLS в `config.yaml`:
```yaml
tls_cert_file: /path/to/cert.pem
tls_key_file: /path/to/key.pem
```

Когда TLS включен, флаг Secure автоматически устанавливается для всех cookie.

### Режим HTTP (только для разработки)
При работе без TLS:
- Флаг Secure будет `false` (cookie работают через HTTP)
- Защита от CSRF и валидация Origin остаются активными
- ⚠️ **Не рекомендуется для production**

## Тестирование

Для проверки работы CSRF защиты:

1. **Валидный запрос** (должен пройти):
```bash
# Логин и получение сессионной cookie
curl -c cookies.txt -X POST http://localhost:8080/admin/login \
  -d "username=admin&password=yourpass"

# Запрос с CSRF токеном
curl -b cookies.txt -X POST http://localhost:8080/admin/zones \
  -H "X-CSRF-Token: <токен-из-сессии>" \
  -d "name=example.com"
```

2. **Невалидный запрос** (должен вернуть 403):
```bash
# Запрос без CSRF токена
curl -b cookies.txt -X POST http://localhost:8080/admin/zones \
  -d "name=example.com"
```

## Обратная совместимость

Эти изменения **НЕ обратно совместимы** с:
- Пользовательскими скриптами/инструментами, выполняющими POST/PUT/DELETE запросы к admin эндпоинтам
- Автоматизированными взаимодействиями с панелью администратора

Такие интеграции должны быть обновлены:
1. Выполнить логин для получения сессии
2. Извлечь CSRF токен из сессии или HTML dashboard
3. Включить CSRF токен в изменяющие запросы

## Ссылки

- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [MDN: SameSite cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite)
- [MDN: Secure cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#security)
