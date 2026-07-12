# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Requirements: Go 1.25+, Node 20+, pnpm 9+. The Makefile is the entry point (`make help`).

```bash
make backend          # Go panel binary only (full): CGO_ENABLED=0 go build ./cmd/panel
make backend-lite     # Lite edition: same with -tags lite
make frontend         # Build all three SPAs (pnpm, frozen lockfile) → web/{admin,portal,landing}/www
make build            # frontend + backend (full)
make vet              # go vet ./...
make test             # go test ./...
make check            # vet + test (CI gate)
make clean            # remove artifacts
```

- **Run a single test:** `go test ./internal/api/ -run TestFailover -v`. Frontend tests: `cd web && pnpm test`.
- **Lint:** `golangci-lint run` (config in `.golangci.yml`: errcheck, gosec, staticcheck, revive, govet, misspell, bodyclose, unparam). gosec/errcheck relaxed inside `_test.go`.
- **Proto regen:** `buf.gen.yaml` generates `internal/knodepb` from the **external** `../knode/proto` tree (needs `protoc-gen-go` + `protoc-gen-go-grpc`). See the file header for the exact `buf generate ../knode/proto` / `protoc` invocations.

**Embed gotcha:** the frontends are embedded via `web/embed.go` (`//go:embed all:admin/www`, etc.). `make backend` / `go build ./cmd/panel` only produce a usable binary after `make frontend` has populated `web/{admin,portal,landing}/www`; those dirs are empty on a fresh checkout. Any test that imports the panel package also triggers the embed, so run `make frontend` first when building/testing locally without the CI artifacts.

**Frontend lockfile gotcha:** if `make frontend` aborts with `ERR_PNPM_OUTDATED_LOCKFILE` (stale `web/pnpm-lock.yaml` vs `web/core/package.json`), run `make frontend-dev` instead (non-frozen `pnpm install`).

If `go`/`pnpm` are not on PATH, source the toolchain env that sets `GOROOT`/`GOPATH`/`GOFLAGS` (and `sudo npm i -g pnpm@9` if pnpm is absent). `make backend` writes the binary to `./koris` in the repo root.

## Architecture

A **single Go binary** (`cmd/panel`) is the entire product: a Go HTTP API (`internal/api`) plus three Vue 3 SPAs (admin, portal, landing) embedded via `web/embed.go`, with SQL migrations embedded too. There is no separate worker/gateway binary in this repo — job processing runs **in-process** via `internal/worker` (billing, invoice, email, report jobs); the Lite build replaces it with no-op stubs (`worker_stubs_lite.go`).

Communication & external systems:

- **knode** (external node agent, `github.com/anonysec/knode`) is the VPN data plane. The panel talks to it over gRPC + mTLS via `internal/grpcclient`, using the generated `internal/knodepb` types (generated from knode's proto, not checked in here).
- **Data layer:** `internal/db` (`db.Open`, `db.Migrate`) + `internal/dbstore` over Postgres/TimescaleDB; SQL migrations live in `migrations/`.
- **Config:** `internal/config.Load()` reads the panel's `PANEL_*` env vars (full env list in `config.go`).
- **Notifications:** `internal/notify` is a unified `Notifier` fanning out through `EmailSender` and `TelegramSender`.
- **Security primitives:** `internal/safepath` (path-traversal-safe file ops), `internal/safeexec` (allowlisted command execution), `internal/safehttp` (safe HTTP helpers), `internal/ratelimit` (in-process rate limiting, used by `cmd/panel/main.go`).

## Edition split (Full vs Lite)

Done entirely with Go build tags via paired files. The Lite build strips billing/payment/support (and related API routes/handlers) plus the in-process worker. Paired files:

```
//go:build !lite   … services_full.go,  worker_full.go          (cmd/panel)
                  … api_settings_full.go, routes_full.go, edition_full.go (internal/api)
                  … internal/{billing,payment,support,antidpi,teleproxy}/*_full.go
//go:build lite    … services_lite.go,  worker_lite.go, worker_stubs_lite.go (cmd/panel)
                  … api_settings_lite.go, routes_lite.go, edition_lite.go (internal/api)
                  … internal/{billing,payment,support,antidpi,teleproxy}/*_lite.go
```

When modifying shared features, keep both the `_full.go` and `_lite.go` counterparts in sync.

## Port & TLS model

- **Single port** `:2096` serves **HTTPS** (`PANEL_TLS_ADDR`, default `:2096`). This is the only externally published port; HTTPS is mandatory for external traffic.
- If the cert can't be loaded, the panel falls back to **loopback-only HTTP on the same port** (`127.0.0.1:2096`) so an admin can fix the cert locally. Plaintext is never exposed off-host.
- `PANEL_ADDR` (default `:8080`) is a *separate* loopback HTTP listener, started only when it differs from `PANEL_TLS_ADDR`. In the Docker deployment `PANEL_ADDR=0.0.0.0:2096`, so it is not started — you get one port. It exists for local non-Docker dev.
- **TLS modes** (`PANEL_TLS_MODE`, default `selfsigned`): `manual` (custom cert at `/etc/koris/certs`), `selfsigned` (dev auto-generated), `acme` (Let's Encrypt/ZeroSSL via acme.sh, driven by `PANEL_DOMAIN`), `disabled`.

## KORIS_HOME & data layout

`KORIS_HOME` (default `/opt/koris`) is the unified host data dir, mounted into the container at `/etc/koris`. Layout:

```
/opt/koris/
  panel.env         # all panel config (compose injects the same vars)
  certs/            # cert.pem, key.pem  → /etc/koris/certs
  data/             # PostgreSQL/TimescaleDB data (bind-mounted to the db service)
  acme/             # acme.sh home (when using auto SSL)
  pgadmin/          # optional pgAdmin data
```

`.env.example` is fully commented (`#KEY=default` per line); copy to `.env` and uncomment to override. The live `.env` keeps most lines commented — only compose-required vars, the persistent Postgres credentials, and the knode pairing key stay active (changing those breaks the running DB/node).

## Docker & koris.sh

Docker is the **primary, supported deployment**. `docker-compose.yml` brings up the panel + TimescaleDB + pgAdmin (optional `redis` and `pgadmin` profiles).

`koris.sh` is the **single operations script** (the old `install.sh`/`helpers.sh`/`koris.sh` were merged into it). Subcommands: `install`, `start`, `stop`, `restart`, `status`, `logs`, `follow`, `update`, `config`, `uninstall`, `clean`, `admin` (`list`/`passwd`/`create` — runs the panel binary against the DB directly, so it works even when the panel is stopped or you are locked out), `db` (`backup`/`restore`/`migrate`/`reset`/`shell`/`status`), `pgadmin` (`status`/`enable`/`disable`/`url`/`reset-password`/`port`), `reinstall`, `downgrade`, `enable`, `disable`, `node-status`, `node-restart`, `node-logs`, `help`.

Install flags of note: `--port=`, `--home=DIR` (KORIS_HOME), `--domain=`, `--ssl=domain|ip|custom|selfsigned`, `--ssl-target=`, `--cert-path=`, `--key-path=`, `--lite`/`--full`, `--no-knode`, `--from-source`/`--from-release`, `--reinstall`. **Native (non-Docker) install was removed** — Docker only.

## Running locally (no Docker)

The panel is a single binary needing a reachable Postgres/TimescaleDB and (on a fresh DB) the SQL in `migrations/`. Full env list is in `internal/config/config.go`:

- **DB:** `PANEL_DB_DSN` (or `PANEL_PG_DSN`), e.g. `postgres://koris@127.0.0.1:5432/koris?sslmode=disable`. `PANEL_MIGRATIONS` points at the `migrations/` dir (default relative to cwd).
- **Secret/dev:** `PANEL_SESSION_SECRET` must be ≥32 chars in production, or set `PANEL_DEV_MODE=true` to relax it (and the DB-DSN requirement for local dev).
- **TLS:** for local dev use `PANEL_TLS_MODE=selfsigned` + `PANEL_TLS_ENABLED=true` (auto dev cert) or `PANEL_TLS_ENABLED=false` to use the `:8080` loopback HTTP. With a real domain, `PANEL_TLS_MODE=acme` + `PANEL_DOMAIN` triggers built-in ACME.
- **CLI socket:** `PANEL_SOCKET_PATH` defaults to `/var/run/panel.sock`, which a non-root user can't write (harmless warning; CLI falls back to HTTP). Point it at a writable path, e.g. `PANEL_SOCKET_PATH=/run/koris/panel.sock` after `sudo mkdir -p /run/koris && sudo chown $USER /run/koris`.

## CI note

In `.github/workflows/ci.yml` the **Backend job `needs: [frontend]`** and downloads the built `web/*/www` dirs as an artifact before `go build`/`go vet`/`go test`. This is required because of the embed: on a fresh checkout those dirs are empty, so building the backend in parallel with (or before) the frontend fails with `pattern all:admin/www: no matching files found`. If you change CI, keep the SPA build ahead of any Go compile that pulls in the `web` package.
