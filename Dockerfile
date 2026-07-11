# syntax=docker/dockerfile:1
#
# KorisPanel — single multi-stage build.
# Stage 1 compiles the three Vue SPAs and the statically-linked Go panel
# binary (assets are embedded at build time via //go:embed).
# Stage 2 is a minimal Alpine runtime that also ships the built www/ dirs and
# the SQL migrations so they can be overridden at the binary's CWD (/app).
#
# Build args (override per-deployment):
#   BUILD_TAGS        e.g. "lite" for the Lite edition
#   KORIS_ADMIN_BASE  Vite base path — must match PANEL_ADMIN_PATH
#   KORIS_PORTAL_BASE Vite base path — must match PANEL_PORTAL_PATH

# ─────────────────────────────── Stage 1: builder ────────────────────────────
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git nodejs npm && npm install -g pnpm@9
WORKDIR /build

# Cache Go module downloads before copying the full source tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the three SPAs. Vite base paths MUST match the runtime routing
# prefixes or the bundled asset URLs will 404.
ARG KORIS_ADMIN_BASE=/admin/
ARG KORIS_PORTAL_BASE=/account/
ENV KORIS_ADMIN_BASE=${KORIS_ADMIN_BASE}
ENV KORIS_PORTAL_BASE=${KORIS_PORTAL_BASE}
WORKDIR /build/web
RUN pnpm install --frozen-lockfile || pnpm install \
    && pnpm --filter admin build \
    && pnpm --filter portal build \
    && pnpm --filter landing build

# Build the panel binary: static, stripped, no CGO.
WORKDIR /build
ARG BUILD_TAGS=""
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" ${BUILD_TAGS:+-tags "$BUILD_TAGS"} -o /koris ./cmd/panel

# ─────────────────────────────── Stage 2: runtime ────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget libcap postgresql-client

# Non-root user. The binary keeps cap_net_bind_service so it can bind 80/443
# without being root.
RUN addgroup -S koris && adduser -S koris -G koris -h /app -s /sbin/nologin
WORKDIR /app

COPY --from=builder /koris /app/koris

# SPA assets are embedded in the binary at build time (web/*.go //go:embed), so
# no on-disk www/ copy is needed. To override an SPA without rebuilding, mount
# built assets and set PANEL_*_WEB_DIR to their path.

# SQL migrations: db.Migrate resolves this relative to CWD (/app).
COPY --from=builder /build/migrations /app/migrations

# Grant the privilege to bind ports 80/443, then fix ownership.
RUN setcap cap_net_bind_service=+ep /app/koris \
    && mkdir -p /var/lib/koris /etc/koris /opt/koris \
    && chown -R koris:koris /app /var/lib/koris /etc/koris /opt/koris

USER koris
# Single HTTPS port — the panel never binds 80/443.
EXPOSE 2096
HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=15s \
    CMD wget --no-check-certificate -q --spider "https://localhost:2096/api/health" \
        || wget -q --spider "http://127.0.0.1:2096/api/health" \
        || exit 1
ENTRYPOINT ["/app/koris"]
