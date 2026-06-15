# Koris Next

Clean KorisPanel rewrite split into Panel and Node.

Current version: see [`VERSION`](VERSION). Release notes: [`CHANGELOG.md`](CHANGELOG.md).

Stack:
- Go backend/services
- Vue 3 + TypeScript admin dashboard and customer portal
- MariaDB + FreeRADIUS schema
- HTTP node push; no SSH dependency
- Clean DB names; no brand prefixes in tables

Runtime split:
- `panel/` admin API, customer portal API, DB/RADIUS integration, Vue static UI
- `node/` node agent and VPN protocol control

## Frontend

The admin app is served at `/dashboard/` and builds into `panel/web/admin/www`.
The customer portal is served at `/portal/` and builds into `panel/web/portal/www`.

```bash
cd panel/web/admin && npm install && npm run build
cd ../portal && npm install && npm run build
```

## Install panel

```bash
sudo PANEL_ADDR=127.0.0.1:8088 \
  DB_NAME=radius_next DB_USER=radius_next DB_PASS='RadiusNext2026' \
  SETUP_KEY='change-me' \
  bash scripts/install-panel.sh
```

The installer copies prebuilt `www/` frontends if present. If `www/` is missing and `npm` exists, it builds them during install.

## Install node

```bash
sudo bash scripts/install-node.sh http://PANEL/api TOKEN Node1
```
