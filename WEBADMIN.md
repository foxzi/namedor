# Web Admin Panel

GeoDNS includes a built-in web-based admin panel for managing DNS zones and records with GeoIP support.

## Features

- ✅ **Zone Management**: Create, view, and delete DNS zones
- ✅ **DNS Records**: Full CRUD for A, AAAA, CNAME, MX, TXT, NS records
- ✅ **GeoIP Support**: Configure geo-routing by Country, Continent, ASN, or Subnet
- ✅ **Session-based Auth**: Secure login with bcrypt password hashing
- ✅ **HTMX Interface**: Fast, interactive UI without JavaScript frameworks
- ✅ **Easy Configuration**: Enable/disable via config file

## Quick Start

### 1. Generate Password Hash

```bash
go run cmd/hashpwd/main.go yourPassword
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

**⚠️ Security Note**: The example hash above is for password "admin". **Change this in production!**

### 3. Start Server

```bash
./smaillgeodns
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
