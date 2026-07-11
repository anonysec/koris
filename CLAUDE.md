# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Requirements: Go 1.25+, Node 20+, pnpm 9+. The Makefile is the entry point (`make help`).

```bash
make backend          # Go panel binary only (full edition): CGO_ENABLED=0 go build ./cmd/panel
make backend-lite     # Lite edition: same with -tags lite
make frontend         # Build all three SPAs (pnpm, frozen lockfile) → web/{admin,portal,landing}/www
make build            # frontend + backend (full)
make vet              # go vet ./...
make test             # go test ./...
make check            # vet + test (CI gate)
make clean            # remove artifacts
```

- **Run a single test:** `go test ./internal/api/ -run TestFailover -v` (or `-run TestFailoverHand -v`). Frontend tests: `cd web && pnpm test`.
- **Lint:** `golangci-lint run` (config in `.golangci.yml` — enables errcheck, gosec, staticcheck, revive, govet, misspell, bodyclose, unparam, etc.). gosec/errcheck are relaxed only inside `_test.go`.
- **Proto regen:** `buf.gen.yaml` drives generation of `internal/knodepb` from an external `../knode/proto` tree (needs `protoc-gen-go` + `protoc-gen-go-grpc`). See the file header for the exact `buf generate` / `protoc` invocations.

Note: frontends are embedded via `web/embed.go` (`go:embed all:admin/www`, etc.). `make build`/`make backend` only produce a usable binary after `make frontend` has populated those `www` dirs; the dirs exist but are empty in a fresh checkout, so a backend-only build embeds no assets.

If `go`/`pnpm` are not on PATH, source the toolchain env that sets `GOROOT`/`GOPATH`/`GOFLAGS` before any Go command (and `sudo npm i -g pnpm@9` if pnpm is absent). `make backend` writes the binary to `./koris` in the repo root.

**Frontend lockfile gotcha:** if `make frontend` aborts with `ERR_PNPM_OUTDATED_LOCKFILE` (the checked-in `web/pnpm-lock.yaml` is stale vs `web/core/package.json`), run `make frontend-dev` instead — a non-frozen `pnpm install` that updates the lockfile.

## Architecture

Three separate binaries, one Go module (`github.com/anonysec/koris`), ~43 internal packages:

- **`cmd/panel`** — the single self-contained service: a Go HTTP API (`internal/api`) plus three Vue 3 SPAs (admin, portal, landing) embedded via `web/embed.go`. This is the main product binary.
- **`cmd/worker`** — a gRPC server implementing `korispb.WorkerService`. It owns a job queue (`internal/jobs`: `Queue` + `Processor`) that runs billing, invoice, email, and report jobs; the panel enqueues work here over gRPC and polls `GetJobStatus`.
- **`cmd/gateway`** — a TLS-terminating reverse proxy that sits in front of panel/worker, doing rate limiting (`internal/ratelimit`), API-key auth (`internal/gateway`), and TLS handling.

Communication & external systems:

- **knode** (external node agent at `github.com/anonysec/knode`) talks to the panel over gRPC + mTLS via `internal/grpcclient`, using the generated `internal/knodepb` types. `proto/korispb` is the *separate, hand-maintained* proto for the panel↔worker service (not generated from a checked-in `.proto`).
- **Data layer:** `internal/db` (`db.Open`, `db.Migrate`) + `internal/dbstore` over Postgres/TimescaleDB; SQL migrations live in `migrations/`.
- **Config:** `internal/config` exposes `Load()` (panel), `LoadWorker()`, `LoadGateway()` — each reads its own env vars.
- **Notifications:** `internal/notify` exposes a unified `Notifier` that fans out through an `EmailSender` and a `TelegramSender`.

Edition split (Full vs Lite) is done entirely with Go build tags, via paired files:

```
//go:build !lite   … services_full.go,   worker_full.go
//go:build lite    … services_lite.go,   worker_lite.go, api_settings_lite.go, routes_lite.go, edition_lite.go, internal/{billing,payment,support,antidpi}/*_lite.go
```

The Lite build strips payment/billing/support (and related API routes/handlers). When modifying shared features, keep both the `_full.go` and `_lite.go` counterparts in sync.

## Redis (optional, scales horizontally)

`REDIS_ADDR` (plus optional `REDIS_PASSWORD`, `REDIS_DB`) enables Redis. When unset, all
backends fall back to their in-memory implementations — Redis is strictly opt-in.

- **Shared job queue:** `internal/jobs` has `Queue` (in-memory) and `redisQueue` (Redis LPUSH/BRPOP + a hash per job, payloads serialized with protojson). `cmd/worker` picks `redisQueue` when Redis is reachable, else `Queue`. This lets multiple `cmd/worker` instances share work.
- **API cache:** `internal/api` uses a `cache.Cache` interface, `QueryCache` (in-memory LRU) or `RedisCache`. Selected in `api.New` via `newCache()`.
- **Gateway rate limiter:** `internal/gateway/ratelimit.go` uses an in-memory token bucket by default, or a shared Redis token bucket (atomic Lua script) when Redis is configured. Fails open on Redis errors.

Enable in compose with `docker compose --profile redis up -d` and `REDIS_ADDR=redis:6379`.

## Running the panel locally (no Docker)

The panel is a single binary that needs a reachable Postgres/TimescaleDB and (on a fresh DB) the SQL in `migrations/`. Full env list is in `internal/config/config.go`; the ones that bite most often:

- **DB:** `PANEL_DB_DSN` (or `PANEL_PG_DSN`), e.g. `postgres://postgres@127.0.0.1:5432/koris?sslmode=disable`. `PANEL_MIGRATIONS` points at the `migrations/` dir (default `migrations` relative to cwd).
- **Secret/dev:** `PANEL_SESSION_SECRET` must be ≥32 chars in production, or set `PANEL_DEV_MODE=true` to relax it (and the DB-DSN requirement).
- **TLS / Let's Encrypt autocert:** `PANEL_TLS_ENABLED=true` + `PANEL_DOMAIN=example.com` turns on the **built-in autocert** (ACME HTTP-01). The panel binds `:80` (challenge handler + 302 redirect) and `:443`, and provisions a trusted cert into `PANEL_TLS_CERT_DIR` (default `/etc/koris/certs`, auto-created). This path triggers **only when the default custom-cert files `/etc/koris/cert.pem` and `/etc/koris/key.pem` do NOT exist** — if either exists, it serves those and skips ACME. `PANEL_DOMAIN` drives the whole decision (read from env in `cmd/panel/main.go`, not from `config.go`).
- **Non-root port binding:** binding `:80`/`:443` as a non-root user requires `sudo setcap cap_net_bind_service=ep <binary>`; otherwise run as root. `:8080` (loopback only) always serves plain HTTP for local tooling and is redirected to HTTPS for external traffic.
- **CLI socket:** `PANEL_SOCKET_PATH` defaults to `/var/run/panel.sock`, which a non-root user can't write — you'll get a harmless "permission denied" warning and the CLI falls back to HTTP. Point it at a writable path, e.g. `PANEL_SOCKET_PATH=/run/koris/panel.sock` (after `sudo mkdir -p /run/koris && sudo chown $USER /run/koris`).

## Known rough edges (as of this writing)

These currently **compile** but are not runtime-complete — flag before relying on them:

- (Resolved) `proto/korispb` was regenerated from `proto/korispb/worker.proto` (real `proto.Message` types), so gRPC marshaling now works. The `.proto` is checked in under `proto/korispb/`.
- (Resolved) The gateway `KorisProxy` now stores `AuthMiddleware` and injects the API-key header per request instead of the prior `auth.InjectAPIKeyHeader(nil)` nil-panic. The gateway still has no `WorkerService` client fan-out path.
- (Resolved) Fresh-install migration bug — `migrations/001_init.sql` created `api_keys` with an old schema, and the original `005_api_keys.sql` re-declared the table and added an index on a non-existent `key_prefix` column, which **failed on a clean database** and aborted `db.Migrate`. `005_api_keys.sql` was reconciled to use additive, idempotent DDL (`ADD COLUMN IF NOT EXISTS key_prefix/created_by`, drop legacy columns, `CREATE INDEX IF NOT EXISTS idx_api_keys_prefix`), so a fresh install now succeeds and matches the schema the app expects.

