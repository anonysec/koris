# KorisPanel Lite

Lightweight VPN management panel. Single binary, minimal setup.

**Features:**
- Admin dashboard (settings, users, nodes)
- Customer management (create, edit, delete, data limits)
- Node management (add nodes, toggle protocols, monitor status)
- OpenVPN + L2TP protocol support
- FreeRADIUS integration (authentication + accounting)
- Node agent API (push metrics, poll tasks)

**Not included** (see full version):
- Reseller system
- Ticket support
- Advanced billing (wallets, invoices, gateways)
- WireGuard, IKEv2, SSH, Xray protocols
- Theme customization
- Statistics dashboard

## Quick Start

```bash
# 1. Build
go build -o koris-lite ./cmd/panel

# 2. Configure
export PANEL_DB_DSN="radius:password@tcp(127.0.0.1:3306)/radius?parseTime=true&multiStatements=true"
export PANEL_SESSION_SECRET="your-32-char-secret-here-change!"
export PANEL_ADDR=":8080"

# 3. Run (migrations run automatically)
./koris-lite

# 4. Create admin account
curl -X POST http://localhost:8080/api/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "yourpassword"}'

# 5. Access dashboard
open http://localhost:8080/dashboard/
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/login | Admin login |
| POST | /api/auth/setup | Initial admin creation (first run only) |
| GET | /api/health | Health check |
| GET/POST | /api/admin/settings | Panel settings |
| GET/POST | /api/admin/customers | List/create customers |
| GET/PUT/DELETE | /api/admin/customers/:id | Customer CRUD |
| GET/POST | /api/admin/nodes | List/create nodes |
| GET/PUT/DELETE | /api/admin/nodes/:id | Node CRUD |
| GET/POST | /api/admin/protocols | Protocol management |
| POST | /api/node/push | Node agent metrics push |
| GET | /api/node/tasks/poll | Node agent task polling |

## Requirements

- Go 1.22+
- MariaDB 10.11+
- FreeRADIUS (for VPN authentication)

## Deployment

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o koris-lite ./cmd/panel

# Copy to server
scp koris-lite root@server:/usr/local/bin/
scp -r migrations root@server:/opt/koris-lite/
scp systemd/koris-lite.service root@server:/etc/systemd/system/

# On server
mkdir -p /etc/koris-lite
cat > /etc/koris-lite/.env << 'EOF'
PANEL_DB_DSN=radius:password@tcp(127.0.0.1:3306)/radius?parseTime=true&multiStatements=true
PANEL_SESSION_SECRET=change-me-to-random-32-chars!!!
PANEL_ADDR=:8080
PANEL_MIGRATIONS=/opt/koris-lite/migrations
EOF

systemctl daemon-reload
systemctl enable --now koris-lite
```

## Branch Strategy

This is the `lite` branch of the KorisPanel repository. The `main` branch contains the full-featured version with all protocols, billing, reseller system, etc.
