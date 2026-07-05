# Contributing to KorisPanel

Thanks for contributing! This document keeps the codebase consistent and CI green.

## Architecture

- **Backend:** Go (`KorisPanel/panel`) under `panel/internal/...`, entrypoint
  `panel/cmd/panel`. Exposes an HTTP/gRPC API consumed by the web UIs.
- **Frontend:** pnpm workspace under `panel/web` with apps `admin`, `portal`,
  `landing` (Vue 3 + Vite).
- **Database:** TimescaleDB (PostgreSQL) with migrations in `panel/migrations`.
- **Node agent:** the separate [`knode`](https://github.com/anonysec/knode)
  repository connects to this panel.

## Prerequisites

- Go 1.25+
- Node.js 20+ and pnpm 9 (`corepack enable`)
- Docker (for container builds / full stack)
- `golangci-lint` (optional locally; enforced in CI)

## Local Workflow

```bash
# Backend
go build ./...
go vet ./...
go test ./... -count=1

# Frontend
cd panel/web
pnpm install --frozen-lockfile
pnpm --filter admin build
pnpm --filter portal build
pnpm --filter landing build

# Full stack
docker compose -f docker/docker-compose.dev.yml up --build
```

## Commit & PR Guidelines

- Branch from `main`: `feat/...`, `fix/...`, `chore/...`, `docs/...`.
- Keep commits focused; write imperative messages.
- Add/update tests for behavior changes.
- Run `go vet` + `go test` and the frontend build before pushing.
- CI must pass: backend build/vet/test, frontend install/build, `golangci-lint`.
- Security-sensitive changes require a note in the PR description.

## Code Style

- `gofmt` + `goimports` (enforced by CI).
- Wrap errors with context: `fmt.Errorf("do thing: %w", err)`.
- Prefer structured responses via the API helpers; no raw `fmt.Fprint` to clients.
