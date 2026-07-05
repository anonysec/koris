# KorisPanel Multi-Stage Dockerfile
# Builds the Go binary + frontend assets from source.
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git nodejs npm && npm install -g pnpm@9
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /build/panel/web
RUN pnpm install --frozen-lockfile && \
    pnpm --filter admin build && \
    pnpm --filter portal build && \
    pnpm --filter landing build
WORKDIR /build
ARG BUILD_TAGS=""
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" ${BUILD_TAGS:+-tags $BUILD_TAGS} -o /koris ./panel/cmd/panel

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /koris /app/koris
# Admin UI static assets (served by the panel)
COPY --from=builder /build/panel/web/admin/www /opt/KorisPanel/panel/web/admin/www
# SQL migrations: the binary resolves this relative to its CWD (/app),
# so it must live at /app/panel/migrations.
COPY --from=builder /build/panel/migrations /app/panel/migrations
RUN mkdir -p /var/lib/koris
# NOTE: the process binds 443/80 and writes to /etc/koris and /var/lib/koris,
# so it runs as root in the default deployment. For non-root: setcap
# cap_net_bind_service=+ep /app/koris, chown the bind mounts, then USER <uid>.
EXPOSE 2026 443
HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=15s \
    CMD wget --no-check-certificate -q --spider https://localhost:2026/api/health || \
        wget --no-check-certificate -q --spider https://localhost:443/api/health || exit 1
ENTRYPOINT ["/app/koris"]
