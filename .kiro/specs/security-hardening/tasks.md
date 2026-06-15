# Tasks: Security Hardening

## Task 1: Fix `randomHex()` to use crypto/rand

> Requirement: 1

- [x] 1.1 Replace the `randomHex()` function body in `panel/internal/bot/bot.go` to use `crypto/rand.Read` + `hex.EncodeToString`
- [x] 1.2 Add `crypto/rand` and `encoding/hex` imports to `bot.go` (remove `time` import if no longer needed elsewhere)
- [x] 1.3 Remove the `time.Sleep(time.Nanosecond)` loop pattern
- [x] 1.4 Add unit test in `panel/internal/bot/bot_test.go`: verify output length is `2*n`, all hex chars, and 1000 calls produce unique values

## Task 2: Add constant-time comparison to customer login

> Requirement: 2

- [x] 2.1 Import `crypto/subtle` in `panel/internal/api/api.go`
- [x] 2.2 Replace `pw != in.Password` comparison in `customerLogin()` with `subtle.ConstantTimeCompare([]byte(pw), []byte(in.Password)) != 1`
- [x] 2.3 Add dummy comparison on the `err != nil` path (user not found) to prevent timing-based enumeration
- [x] 2.4 Apply same constant-time pattern to `portalPassword()` handler where current password is verified

## Task 3: Enforce mandatory config in production

> Requirements: 3, 10

- [x] 3.1 Add `PANEL_DEV_MODE` environment variable check in `config.Load()`
- [x] 3.2 When `PANEL_DEV_MODE != "true"`: call `log.Fatalf` if `PANEL_SESSION_SECRET` is empty
- [x] 3.3 When `PANEL_DEV_MODE != "true"`: call `log.Fatalf` if `PANEL_DB_DSN` is empty
- [x] 3.4 When `PANEL_DEV_MODE != "true"`: validate `PANEL_SESSION_SECRET` is at least 32 characters, `log.Fatalf` otherwise
- [x] 3.5 When `PANEL_DEV_MODE == "true"`: retain existing defaults with security warning logs (current behavior)
- [x] 3.6 Add `SecureCookies bool` field to `Config` struct (true when `PANEL_DEV_MODE != "true"`)
- [x] 3.7 Add `TrustedProxies []string` field to `Config` struct, parsed from `PANEL_TRUSTED_PROXIES` (comma-separated)
- [x] 3.8 Add `AllowedOrigins []string` field to `Config` struct, parsed from `PANEL_ALLOWED_ORIGINS` (comma-separated)

## Task 4: Implement CSRF middleware

> Requirement: 4

- [x] 4.1 Create `panel/internal/csrf/csrf.go` with `Middleware(secret string, next http.Handler) http.Handler` function
- [x] 4.2 Implement token generation: HMAC-SHA256 of session cookie value with the secret, base64url-encoded
- [x] 4.3 On safe methods (GET, HEAD, OPTIONS): set `X-CSRF-Token` response header and pass through
- [x] 4.4 On state-changing methods (POST, PUT, PATCH, DELETE): validate `X-CSRF-Token` request header
- [x] 4.5 Implement path exemption logic: skip validation for paths starting with `/api/node/` and exact path `/api/bot/webhook`
- [x] 4.6 Return HTTP 403 `{"ok":false,"error":"csrf_invalid"}` when token is missing or invalid
- [x] 4.7 Wire CSRF middleware into `panel/cmd/panel/main.go` between rate limiter and route handler
- [x] 4.8 Add unit tests in `panel/internal/csrf/csrf_test.go`: test generation, validation, exemption, rejection

## Task 5: Add Secure flag to session cookies

> Requirement: 5

- [x] 5.1 Add `secure bool` parameter to `auth.SetSession()` function signature
- [x] 5.2 Add `secure bool` parameter to `auth.ClearSession()` function signature
- [x] 5.3 Set `Secure: secure` on the cookie in `SetSession()`
- [x] 5.4 Set `Secure: secure` on the clear cookie in `ClearSession()`
- [x] 5.5 Update all call sites of `SetSession` in `api.go` to pass `s.Config.SecureCookies`
- [x] 5.6 Update all call sites of `ClearSession` in `api.go` to pass `s.Config.SecureCookies`

## Task 6: Validate WebSocket origin

> Requirement: 6

- [x] 6.1 Replace the inline `websocket.Upgrader` in `realtimeWS()` with a `CheckOrigin` function that validates the Origin header
- [x] 6.2 Implement origin matching: parse Origin header, compare host against `PANEL_PUBLIC_BASE` host and `AllowedOrigins` config list
- [x] 6.3 Allow requests with empty Origin header (same-origin browser behavior)
- [x] 6.4 Reject requests with non-matching Origin by returning `false` from `CheckOrigin`
- [x] 6.5 Add unit test verifying allowed and rejected origins

## Task 7: Implement trusted proxy for rate limiter

> Requirement: 7

- [x] 7.1 Add `trustedProxies map[string]bool` field to `Limiter` struct in `panel/internal/ratelimit/ratelimit.go`
- [x] 7.2 Update `New()` constructor to accept `trustedProxies []string` and build the lookup map (support both IPs and CIDRs)
- [x] 7.3 Add `clientIP(r *http.Request) string` method: strip port from RemoteAddr, check trust, conditionally use X-Real-IP
- [x] 7.4 Update `Middleware()` to call `clientIP(r)` instead of directly reading `r.Header.Get("X-Real-IP")`
- [x] 7.5 Update `main.go` to pass `cfg.TrustedProxies` to `ratelimit.New()`
- [x] 7.6 Add unit tests: untrusted proxy ignores X-Real-IP, trusted proxy uses X-Real-IP, CIDR matching works

## Task 8: Fix authNode to not consume request body

> Requirement: 8

- [x] 8.1 Remove the `json.NewDecoder(r.Body).Decode(&in)` fallback in `authNode()` function in `api.go`
- [x] 8.2 Ensure `authNode()` returns `(0, false)` when `X-Node-Token` header is empty (no body reading)
- [x] 8.3 Verify existing node agent code (`node/cmd/node/main.go`) always sends `X-Node-Token` header (confirm no breaking change)
- [x] 8.4 Add comment documenting that token must be in header only

## Task 9: Fix backup password exposure

> Requirement: 9

- [x] 9.1 In `panel/cmd/panel/main.go` backup section, remove `-p`+pass from `exec.Command` arguments
- [x] 9.2 Set `cmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)` on the mysqldump command
- [x] 9.3 Keep `-u` user flag in command arguments
- [x] 9.4 Add comment explaining why `MYSQL_PWD` env var is used instead of `-p` flag

## Task 10: Remove hardcoded fallback sign secret in auth package

> Requirement: 3

- [x] 10.1 Remove the fallback `"koris-next-dev-session-secret"` from the `sign()` function in `panel/internal/auth/auth.go`
- [x] 10.2 Ensure `sign()` uses only the passed `secret` parameter (which is already validated by config loader)
- [x] 10.3 If `secret` is empty in `sign()`, panic with a clear message (defense in depth)
