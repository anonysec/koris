# KorisPanel Multi-Stage Dockerfile
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git nodejs npm && npm install -g pnpm@9
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build all frontends
WORKDIR /build/panel/web
RUN pnpm install --no-frozen-lockfile && pnpm --filter admin build && pnpm --filter portal build && pnpm --filter landing build

# Build Go binary
WORKDIR /build
ARG BUILD_TAGS=""
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" ${BUILD_TAGS:+-tags $BUILD_TAGS} -o /koris ./panel/cmd/panel

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /koris /app/koris
COPY --from=builder /build/panel/web/admin/www /opt/KorisPanel/panel/web/admin/www
COPY --from=builder /build/panel/migrations /app/panel/migrations
HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=15s     CMD curl -skf https://localhost:443/api/health || curl -sf http://localhost:8080/api/health || exit 1
ENTRYPOINT ["/app/koris"]
