# Docker Deployment Guide

This guide covers deploying KorisPanel using Docker and Docker Compose.

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/your-org/KorisPanel.git
cd KorisPanel/panel

# 2. Copy the environment file
cp docker/panel.env.example docker/panel.env

# 3. Edit configuration (set domain, secrets, etc.)
nano docker/panel.env

# 4. Start all services
docker compose up -d

# 5. Access the panel
# Admin:  http://localhost/dashboard/
# Portal: http://localhost/portal/
```

The panel will run database migrations automatically on first startup.

## Configuration

All configuration is done via environment variables in `docker/panel.env`.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | Database hostname | `db` (use service name) |
| `DB_PORT` | Database port | `3306` |
| `DB_NAME` | Database name | `radius` |
| `DB_USER` | Database user | `radius` |
| `DB_PASS` | Database password | `your-secure-password` |
| `DB_ROOT_PASS` | MariaDB root password | `your-root-password` |
| `PANEL_PORT` | Panel HTTP listen port | `8080` |
| `PANEL_DOMAIN` | Public domain name | `panel.example.com` |
| `PANEL_SESSION_SECRET` | Session signing key (64+ chars) | Random string |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PANEL_WORKERS` | `1` | Number of worker processes |
| `PANEL_GRACEFUL_WAIT` | `30` | Seconds to wait during graceful shutdown |
| `PANEL_DB_MAX_OPEN` | Auto-tuned | Max open DB connections |
| `PANEL_DB_MAX_IDLE` | Auto-tuned | Max idle DB connections |
| `PANEL_DB_MAX_LIFETIME` | `5m` | Max connection lifetime |
| `PANEL_TUI_ENABLED` | `false` | TUI dashboard (disable in Docker) |
| `PANEL_MIGRATIONS` | `/app/migrations` | Path to SQL migration files |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token for notifications |
| `TELEGRAM_CHAT_ID` | — | Telegram chat ID for alerts |

## Architecture

The Docker deployment runs three services:

```
┌─────────────────────────────────────────────────┐
│  nginx:alpine (ports 80, 443)                   │
│  Reverse proxy → routes to panel service        │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  panel (port 8080)                              │
│  Go binary + frontend assets + migrations       │
│  Runs as non-root user (UID 1000)              │
└──────────────────────┬──────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────┐
│  mariadb:10.11 (port 3306)                      │
│  Persistent volume: db-data                     │
└─────────────────────────────────────────────────┘
```

### Volumes

- `db-data` — MariaDB data directory (`/var/lib/mysql`)
- `panel-data` — Panel application data (`/var/lib/panel`)

### Health Checks

- **panel**: `wget -q --spider http://localhost:8080/api/health` every 10s
- **db**: MariaDB `healthcheck.sh --connect --innodb_initialized` every 5s
- The panel service waits for the DB health check to pass before starting.

## Development

Use the development override file for local development with hot-reload:

```bash
docker compose -f docker-compose.yml -f docker/docker-compose.dev.yml up
```

This provides:

- **Go hot-reload**: Source is bind-mounted; panel runs via `go run`
- **Admin dev server**: Vite on `http://localhost:5173` with HMR
- **Portal dev server**: Vite on `http://localhost:5174` with HMR
- **Exposed DB port**: MariaDB on `localhost:3306` for local DB tools

### Dev Services

| Service | Port | Purpose |
|---------|------|---------|
| `panel` | 8080 | Go backend (live reload) |
| `admin-dev` | 5173 | Admin dashboard (Vite HMR) |
| `portal-dev` | 5174 | Customer portal (Vite HMR) |
| `db` | 3306 | MariaDB (accessible from host) |

## Scaling

### Worker Processes

Scale within a single container by setting `PANEL_WORKERS`:

```env
PANEL_WORKERS=4
```

Each worker shares the same port via `SO_REUSEPORT`. Only one worker holds the background task leader lock.

### Multiple Containers

For horizontal scaling, run multiple panel containers behind the nginx proxy:

```yaml
# docker-compose.override.yml
services:
  panel:
    deploy:
      replicas: 3
```

Or scale manually:

```bash
docker compose up -d --scale panel=3
```

Update `docker/nginx.conf` to load-balance across replicas:

```nginx
upstream panel_backend {
    server panel:8080;
    # Docker DNS resolves to all container IPs
}
```

### Scaling Guidelines

| RAM | `PANEL_WORKERS` | `PANEL_DB_MAX_OPEN` | Replicas |
|-----|----------------|---------------------|----------|
| 1 GB | 1 | 10 | 1 |
| 2 GB | 2 | 25 | 1 |
| 4 GB | 4 | 50 | 1–2 |
| 8 GB+ | 4 | 50 | 2–4 |

## Backup & Restore

### Database Backup

```bash
# Backup
docker compose exec db mariadb-dump -u root -p"$DB_ROOT_PASS" radius > backup_$(date +%Y%m%d).sql

# Restore
docker compose exec -T db mariadb -u root -p"$DB_ROOT_PASS" radius < backup_20240101.sql
```

### Volume Backup

```bash
# Stop services first for consistency
docker compose stop

# Backup DB volume
docker run --rm -v panel_db-data:/data -v $(pwd):/backup alpine \
    tar czf /backup/db-data-$(date +%Y%m%d).tar.gz -C /data .

# Backup panel data volume
docker run --rm -v panel_panel-data:/data -v $(pwd):/backup alpine \
    tar czf /backup/panel-data-$(date +%Y%m%d).tar.gz -C /data .

# Restart services
docker compose start
```

### Volume Restore

```bash
docker compose down

# Restore DB volume
docker run --rm -v panel_db-data:/data -v $(pwd):/backup alpine \
    sh -c "rm -rf /data/* && tar xzf /backup/db-data-20240101.tar.gz -C /data"

docker compose up -d
```

## Troubleshooting

### Port Conflicts

**Symptom**: `bind: address already in use` on port 80 or 443.

```bash
# Find what's using the port
sudo lsof -i :80
# or
sudo ss -tlnp | grep :80

# Stop conflicting service or change nginx port in docker-compose.yml:
# ports:
#   - "8443:443"
#   - "8080:80"
```

### Database Connection Failures

**Symptom**: Panel logs `dial tcp db:3306: connect: connection refused`.

```bash
# Check if DB is healthy
docker compose ps db
docker compose logs db --tail=20

# Wait for DB to be ready, then restart panel
docker compose restart panel

# Verify credentials match between panel.env and compose file
grep DB_PASS docker/panel.env
```

### Permission Errors

**Symptom**: `permission denied` when writing to volumes.

```bash
# Fix volume ownership (panel runs as UID 1000)
docker compose exec --user root panel chown -R 1000:1000 /var/lib/panel
docker compose exec --user root panel chown -R 1000:1000 /var/log/panel
```

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f panel
docker compose logs -f db
docker compose logs -f nginx

# Last 50 lines
docker compose logs --tail=50 panel
```

### Container Won't Start

```bash
# Check exit code and logs
docker compose ps -a
docker compose logs panel

# Rebuild from scratch
docker compose down
docker compose build --no-cache panel
docker compose up -d
```

### Migrations Failing

**Symptom**: Panel exits with migration errors on startup.

```bash
# Check migration logs
docker compose logs panel | grep -i migrat

# Connect to DB and check state
docker compose exec db mariadb -u radius -p radius -e "SHOW TABLES;"

# If stuck, reset migrations (WARNING: data loss)
docker compose exec db mariadb -u root -p"$DB_ROOT_PASS" -e "DROP DATABASE radius; CREATE DATABASE radius;"
docker compose restart panel
```

## Upgrading

```bash
# 1. Pull latest changes
git pull origin main

# 2. Rebuild the panel image
docker compose build panel

# 3. Restart with new image (migrations run automatically)
docker compose up -d

# 4. Verify health
docker compose ps
docker compose logs --tail=20 panel
```

For zero-downtime upgrades with multiple replicas:

```bash
# Rolling restart
docker compose up -d --no-deps --build panel
```
