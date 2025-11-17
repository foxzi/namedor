# Security Improvements for Web Admin Panel

## Overview
This document describes the security improvements implemented to address CSRF vulnerabilities and insecure cookie handling in the web admin panel.

## Changes Implemented

### 1. CSRF Protection
- **CSRF Token Generation**: Each session now generates a unique CSRF token
- **Token Validation**: All state-changing requests (POST/PUT/DELETE) validate CSRF tokens
- **Token Transmission**: CSRF tokens are sent via:
  - HTTP header: `X-CSRF-Token` (for HTMX requests)
  - Form field: `csrf_token` (for regular form submissions)

### 2. Secure Cookie Configuration
All cookies now use secure flags:
- **Secure**: `true` when TLS is enabled (cookies only sent over HTTPS)
- **SameSite**: `Strict` (maximum protection against CSRF)
- **HttpOnly**: `true` (protection against XSS)

### 3. Origin/Referer Validation
Additional defense-in-depth protection:
- Validates `Origin` header matches server host
- Validates `Referer` header starts with server host
- Rejects requests without either header

## Technical Details

### Modified Files
- `internal/web/admin.go`:
  - Added `CSRFToken` field to `Session` struct
  - Created `csrfMiddleware()` for token validation
  - Created `validateOrigin()` for Origin/Referer checking
  - Created `setSecureCookie()` for secure cookie handling
  - Updated all cookie-setting calls to use secure configuration

- `internal/web/templates/dashboard.html`:
  - Added `<meta name="csrf-token">` tag
  - Configured HTMX to automatically send CSRF token with all requests

### Security Protections

#### Before
- ❌ No CSRF protection
- ❌ Cookies without Secure flag (transmitted over HTTP)
- ❌ Cookies without SameSite (vulnerable to CSRF)
- ❌ No Origin/Referer validation

#### After
- ✅ CSRF tokens for all state-changing operations
- ✅ Secure cookies (HTTPS-only when TLS enabled)
- ✅ SameSite=Strict (no cross-site cookie transmission)
- ✅ Origin/Referer validation (defense in depth)

## Attack Scenarios Mitigated

### 1. CSRF Attack
**Before**: Attacker could create malicious website with code like:
```html
<form action="https://your-dns-server.com/admin/zones/delete/1" method="POST">
  <input type="submit" value="Click here for free prize!">
</form>
```
If admin is logged in and clicks, zone gets deleted.

**After**: Request blocked due to:
- Missing/invalid CSRF token → 403 Forbidden
- Invalid Origin/Referer → 403 Forbidden

### 2. Cookie Theft via Network Sniffing
**Before**: Session cookies transmitted over HTTP could be intercepted

**After**: Cookies only transmitted over HTTPS (when TLS enabled)

### 3. Cross-Site Cookie Transmission
**Before**: Cookies sent with all requests, even from malicious sites

**After**: SameSite=Strict prevents cookie transmission to your domain from external sites

## Configuration Requirements

### HTTPS Strongly Recommended
For maximum security, configure TLS in `config.yaml`:
```yaml
tls_cert_file: /path/to/cert.pem
tls_key_file: /path/to/key.pem
```

When TLS is enabled, the Secure flag is automatically set on all cookies.

### HTTP Mode (Development Only)
If running without TLS:
- Secure flag will be `false` (cookies work over HTTP)
- CSRF protection and Origin validation still active
- ⚠️ **Not recommended for production**

## Testing

To verify CSRF protection is working:

1. **Valid Request** (should succeed):
```bash
# Login and get session cookie
curl -c cookies.txt -X POST http://localhost:8080/admin/login \
  -d "username=admin&password=yourpass"

# Make request with CSRF token
curl -b cookies.txt -X POST http://localhost:8080/admin/zones \
  -H "X-CSRF-Token: <token-from-session>" \
  -d "name=example.com"
```

2. **Invalid Request** (should fail with 403):
```bash
# Request without CSRF token
curl -b cookies.txt -X POST http://localhost:8080/admin/zones \
  -d "name=example.com"
```

## Backward Compatibility

These changes are **NOT backward compatible** with:
- Custom scripts/tools that make POST/PUT/DELETE requests to admin endpoints
- Automated admin panel interactions

Such integrations must be updated to:
1. Login to obtain session
2. Extract CSRF token from session or dashboard HTML
3. Include CSRF token in state-changing requests

## References

- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [MDN: SameSite cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite)
- [MDN: Secure cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#security)
