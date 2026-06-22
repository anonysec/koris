FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /koris-lite ./cmd/panel

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -u 1000 -D panel
WORKDIR /app
COPY --from=builder /koris-lite /app/koris-lite
COPY migrations /app/migrations
USER panel
EXPOSE 9080
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget -q --spider http://localhost:9080/api/health || exit 1
ENV PANEL_ADDR=:9080 PANEL_MIGRATIONS=/app/migrations
ENTRYPOINT ["/app/koris-lite"]
