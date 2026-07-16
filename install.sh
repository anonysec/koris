#!/usr/bin/env bash
#
# KorisPanel installer — curl-pipeable one-liner.
#
#   curl -sSL https://raw.githubusercontent.com/anonysec/koris/main/install.sh | bash
#
# What it does:
#   1. Checks root + Docker.
#   2. Clones the koris + knode source repos (shallow, HTTPS).
#   3. Creates KORIS_HOME (/opt/koris) for data/certs and the TLS env file.
#   4. Generates .env files (preserved on reinstall — secrets/DB password kept).
#   5. Deploys the Docker stack (docker compose up -d --build).
#   6. Installs a host `koris` wrapper so all management is done via the binary:
#        koris status | nodes | users | admin | cert | logs | update ...
#        koris start | stop | restart           (stack lifecycle)
#
# Default install serves plaintext HTTP on 127.0.0.1 only (loopback, not public).
# Expose HTTPS later with:  koris cert selfsign|letsencrypt|path
#
set -euo pipefail

KORIS_SRC="${KORIS_SRC:-/opt/koris-src}"
KNODE_SRC="$(dirname "$KORIS_SRC")/knode"
KORIS_HOME="${KORIS_HOME:-/opt/koris}"
KORIS_REPO="${KORIS_REPO:-https://github.com/anonysec/koris.git}"
KNODE_REPO="${KNODE_REPO:-https://github.com/anonysec/knode.git}"
DRYRUN=0

case "${1:-}" in
  --dry-run) DRYRUN=1 ;;
  --help|-h)
    sed -n '2,22p' "$0"
    exit 0 ;;
esac

log(){ echo -e "\033[0;32m[+]\033[0m $*"; }
warn(){ echo -e "\033[1;33m[!]\033[0m $*"; }
err(){ echo -e "\033[0;31m[✗]\033[0m $*"; exit 1; }

# ─── Preconditions ───────────────────────────────────────────────────────────
[ "$(id -u)" -eq 0 ] || err "Must run as root"
command -v docker >/dev/null 2>&1 || err "Docker is not installed"
docker info >/dev/null 2>&1 || err "Docker daemon not available"
command -v git >/dev/null 2>&1 || err "git is not installed"
command -v openssl >/dev/null 2>&1 || err "openssl is not installed"

mkdir -p "$KORIS_HOME"/{certs,data,acme}
chmod 777 "$KORIS_HOME" "$KORIS_HOME/certs" 2>/dev/null || true

# ─── Clone sources (shallow) if absent ───────────────────────────────────────
clone() {
  local repo="$1" dir="$2"
  if [ -d "$dir/.git" ]; then
    log "Reusing $dir"
  else
    log "Cloning $repo -> $dir"
    git clone --depth 1 "$repo" "$dir"
  fi
}
clone "$KORIS_REPO" "$KORIS_SRC"
clone "$KNODE_REPO" "$KNODE_SRC"

COMPOSE="$KORIS_SRC/docker-compose.yml"
[ -f "$COMPOSE" ] || err "docker-compose.yml not found in $KORIS_SRC"

# ─── compose .env (variable interpolation) — preserve if it exists ───────────
ENV_COMPOSE="$KORIS_SRC/.env"
if [ ! -f "$ENV_COMPOSE" ]; then
  log "Generating $ENV_COMPOSE"
  PG=$(openssl rand -base64 18 | tr -d '/+=' | head -c 24)
  SESS=$(openssl rand -base64 32 | tr -d '/+=' | head -c 44)
  SETUP=$(openssl rand -hex 8)
  cat > "$ENV_COMPOSE" <<EOF
POSTGRES_PASSWORD=$PG
POSTGRES_USER=koris
PANEL_SESSION_SECRET=$SESS
PANEL_SETUP_KEY=$SETUP
PANEL_DOMAIN=localhost
PANEL_PORT=2096
KNODE_PORT=2087
PANEL_DEV_MODE=false
EOF
else
  log "Preserving existing $ENV_COMPOSE"
fi

# ─── KORIS_HOME/.env (TLS loader, read by panel at startup) — preserve ───────
ENV_TLS="$KORIS_HOME/.env"
if [ ! -f "$ENV_TLS" ]; then
  log "Generating $ENV_TLS (default: HTTP on 127.0.0.1, loopback only)"
  cat > "$ENV_TLS" <<'EOF'
PANEL_TLS_ENABLED=false
PANEL_TLS_MODE=disabled
PANEL_TLS_CERT=/etc/koris/certs/cert.pem
PANEL_TLS_KEY=/etc/koris/certs/key.pem
PANEL_TLS_CERT_DIR=/etc/koris/certs
PANEL_TLS_DOMAIN=
PANEL_TLS_EMAIL=
EOF
  chmod 666 "$ENV_TLS"
else
  log "Preserving existing $ENV_TLS"
fi

# ─── host wrapper: `koris` delegates to the container binary / docker ────────
log "Installing host 'koris' wrapper -> /usr/local/bin/koris"
cat > /usr/local/bin/koris <<WRAP
#!/usr/bin/env bash
# KorisPanel host wrapper — built by install.sh
SRC="$KORIS_SRC"
case "\${1:-}" in
  start|stop|restart|up|down)
    cd "\$SRC" && docker compose "\$@" ;;
  logs|follow)
    shift; cd "\$SRC" && exec docker compose logs -f "\$@" ;;
  status|stack)
    docker compose -f "\$SRC/docker-compose.yml" ps ;;
  *)
    docker exec koris /app/koris "\$@" ;;
esac
WRAP
chmod +x /usr/local/bin/koris

# ─── deploy ──────────────────────────────────────────────────────────────────
if [ "$DRYRUN" -eq 1 ]; then
  log "DRYRUN: skipping 'docker compose up'"
else
  log "Starting KorisPanel: docker compose up -d --build"
  ( cd "$KORIS_SRC" && docker compose up -d --build )
fi

echo
log "Done."
log "Panel (default install): HTTP on http://127.0.0.1:2096  (loopback only, not public)"
log "Install a cert to expose HTTPS:  koris cert selfsign|letsencrypt|path"
log "For a public URL, also publish the port on 0.0.0.0 in the compose and restart."
