# KorisPanel Multi-Stage Dockerfile
# Builds the Go binary + frontend assets from source.
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git nodejs npm && npm install -g pnpm@9

WORKDIR /build

# Go module dependencies (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Source
COPY . .

# Build frontend via pnpm workspace (frozen lockfile for reproducible builds)
WORKDIR /build/panel/web
RUN pnpm install --frozen-lockfile && \
    pnpm --filter admin build && \
    pnpm --filter portal build && \
    pnpm --filter landing build

# Build Go binary (static, no CGO)
WORKDIR /build
ARG BUILD_TAGS=""
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" ${BUILD_TAGS:+-tags $BUILD_TAGS} -o /koris ./panel/cmd/panel

# ---- Runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

# Binary + migrations
COPY --from=builder /koris /app/koris
COPY --from=builder /build/panel/migrations /app/migrations

# Runtime data directory (writable; mounted/overlay at deploy time).
RUN mkdir -p /var/lib/koris

# NOTE: the process binds 443/80 and writes to /etc/koris and /var/lib/koris,
# so it runs as root in the default deployment. For a locked-down non-root
# deployment: `setcap cap_net_bind_service=+ep /app/koris`, `chown` the bind
# mounts to a dedicated uid, then add `USER <uid>` here.

EXPOSE 2026 443

HEALTHCHECK --interval=15s --timeout=5s --retries=5 --start-period=15s \
    CMD wget --no-check-certificate -q --spider https://localhost:2026/api/health || \
        wget --no-check-certificate -q --spider https://localhost:443/api/health || exit 1

ENTRYPOINT ["/app/koris"]
