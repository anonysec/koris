# Requirements: Security Hardening

## Requirement 1: Cryptographically Secure Token Generation

### Acceptance Criteria

1.1. The `randomHex()` function in `bot.go` MUST use `crypto/rand.Read` as its entropy source instead of `time.Now().UnixNano()`.

1.2. The function MUST return a hex-encoded string of exactly `2*n` characters for input parameter `n`.

1.3. The function MUST panic if `crypto/rand.Read` returns an error (entropy exhaustion).

1.4. All existing call sites of `randomHex()` (customer `sub_token` creation via Telegram bot) MUST continue working without interface changes.

---

## Requirement 2: Constant-Time Password Comparison for Customer Portal Login

### Acceptance Criteria

2.1. The customer login handler (`/api/auth/customer`) MUST use `crypto/subtle.ConstantTimeCompare` to compare the stored password with the provided password.

2.2. When the user is not found in the database, the handler MUST still perform a dummy constant-time comparison to prevent timing-based user enumeration.

2.3. The handler MUST return HTTP 401 with `{"ok":false,"error":"invalid"}` for both incorrect password and non-existent user (no distinction).

2.4. The stored password format in `radcheck` (Cleartext-Password attribute) MUST remain unchanged to preserve FreeRADIUS compatibility.

---

## Requirement 3: Mandatory Configuration in Production

### Acceptance Criteria

3.1. When `PANEL_DEV_MODE` is not set to `"true"`, the application MUST exit with a fatal error if `PANEL_SESSION_SECRET` is not set or is empty.

3.2. When `PANEL_DEV_MODE` is not set to `"true"`, the application MUST exit with a fatal error if `PANEL_DB_DSN` is not set or is empty.

3.3. The fatal error message MUST clearly indicate which environment variable is missing and suggest setting `PANEL_DEV_MODE=true` for development.

3.4. When `PANEL_DEV_MODE=true`, the existing default values MUST be used (preserving current development workflow) and a security warning MUST be logged.

3.5. `PANEL_SESSION_SECRET` MUST be at least 32 characters long in production mode; shorter values MUST cause a fatal error.

---

## Requirement 4: CSRF Protection for State-Changing Endpoints

### Acceptance Criteria

4.1. All POST, PUT, PATCH, and DELETE requests to cookie-authenticated endpoints MUST require a valid CSRF token in the `X-CSRF-Token` request header.

4.2. The CSRF token MUST be delivered to clients via the `X-CSRF-Token` response header on all responses.

4.3. Requests missing or providing an invalid CSRF token MUST receive HTTP 403 with `{"ok":false,"error":"csrf_invalid"}`.

4.4. The following endpoints MUST be exempt from CSRF validation: all paths under `/api/node/*` and `/api/bot/webhook` (these use non-cookie authentication).

4.5. GET, HEAD, and OPTIONS requests MUST pass through without CSRF validation.

4.6. The CSRF token MUST be cryptographically bound to the user's session (HMAC of session value with a secret key).

---

## Requirement 5: Secure Cookie Flags

### Acceptance Criteria

5.1. Session cookies (both admin and customer) MUST have the `Secure` flag set to `true` in production mode (`PANEL_DEV_MODE` not `"true"`).

5.2. Session cookies MUST always have `HttpOnly: true` (already implemented, must be preserved).

5.3. Session cookies MUST always have `SameSite: Lax` (already implemented, must be preserved).

5.4. In development mode (`PANEL_DEV_MODE=true`), the `Secure` flag MAY be omitted to allow HTTP-only local development.

---

## Requirement 6: WebSocket Origin Validation

### Acceptance Criteria

6.1. The WebSocket upgrader's `CheckOrigin` function MUST validate the `Origin` request header against allowed origins.

6.2. Allowed origins MUST include the configured `PANEL_PUBLIC_BASE` domain and optionally additional origins from `PANEL_ALLOWED_ORIGINS` environment variable.

6.3. Requests with an `Origin` header that does not match any allowed origin MUST be rejected (upgrade denied).

6.4. Requests without an `Origin` header (same-origin requests from some browsers) MUST be allowed.

---

## Requirement 7: Trusted Proxy Configuration for Rate Limiter

### Acceptance Criteria

7.1. The rate limiter MUST only use the `X-Real-IP` header value when `r.RemoteAddr` (with port stripped) matches an entry in the configured trusted proxies list.

7.2. When `r.RemoteAddr` is not a trusted proxy, the rate limiter MUST use `r.RemoteAddr` (port stripped) as the client identifier, ignoring any `X-Real-IP` header.

7.3. Trusted proxies MUST be configurable via `PANEL_TRUSTED_PROXIES` environment variable (comma-separated IPs or CIDR ranges).

7.4. If `PANEL_TRUSTED_PROXIES` is not set, the rate limiter MUST default to using `r.RemoteAddr` only (trust no proxies).

---

## Requirement 8: Non-Destructive Node Authentication

### Acceptance Criteria

8.1. The `authNode()` function MUST read the node token exclusively from the `X-Node-Token` HTTP header.

8.2. The `authNode()` function MUST NOT read, decode, or consume `r.Body` under any circumstances.

8.3. After `authNode()` returns, `r.Body` MUST remain fully readable by downstream handlers.

8.4. Existing node agents that send the token via `X-Node-Token` header MUST continue to authenticate successfully.

---

## Requirement 9: Password Not Visible in Process List During Backup

### Acceptance Criteria

9.1. The backup worker MUST NOT pass the database password as a command-line argument to `mysqldump` (no `-p{password}` flag).

9.2. The database password MUST be passed to `mysqldump` via the `MYSQL_PWD` environment variable on the child process.

9.3. The `MYSQL_PWD` variable MUST only be set on the child process environment, not the parent process.

9.4. The backup functionality (successful database dump) MUST be preserved.

---

## Requirement 10: Remove Hardcoded Default Database Credentials

### Acceptance Criteria

10.1. The hardcoded default DSN `"radius:RadiusDb2026@tcp(127.0.0.1:3306)/radius..."` MUST NOT be used in production mode.

10.2. In production mode (`PANEL_DEV_MODE` not `"true"`), the absence of `PANEL_DB_DSN` MUST cause a fatal startup error (covered by Requirement 3.2).

10.3. In development mode (`PANEL_DEV_MODE=true`), a default DSN MAY be used with a logged security warning.
