#!/usr/bin/env bash
#
# KorisPanel Unified Installer
# Usage: bash <(curl -Ls https://raw.githubusercontent.com/anonysec/panel/main/install.sh)
#
# Installs one of:
#   1) koris     — Full panel (all features)
#   2) korislite — Lite panel (OpenVPN/L2TP, users, nodes, settings only)
#   3) knode     — Node agent only (for VPN servers managed by the panel)
#

set -e

export TERM="${TERM:-xterm}"

red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; blue='\033[0;34m'; cyan='\033[0;36m'; bold='\033[1m'; plain='\033[0m'
info()  { echo -e "${green}[INFO]${plain} $*"; }
warn()  { echo -e "${yellow}[WARN]${plain} $*"; }
error() { echo -e "${red}[ERROR]${plain} $*"; }
fatal() { echo -e "${red}[FATAL]${plain} $*"; exit 1; }
gen_secret() { openssl rand -hex "$1" 2>/dev/null || head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

[[ $EUID -ne 0 ]] && fatal "Run as root: sudo bash install.sh"

# OS detect
[[ -f /etc/os-release ]] && source /etc/os-release || fatal "Cannot detect OS."
OS=$ID

REPO="anonysec/panel"
KNODE_REPO="anonysec/knode"
BRANCH="main"
INSTALL_DIR="/opt/KorisPanel"
CONFIG_DIR="/etc/koris"

# Banner
clear 2>/dev/null || true
echo -e "${bold}${blue}"
cat << 'EOF'
  ██╗  ██╗ ██████╗ ██████╗ ██╗███████╗
  ██║ ██╔╝██╔═══██╗██╔══██╗██║██╔════╝
  █████╔╝ ██║   ██║██████╔╝██║███████╗
  ██╔═██╗ ██║   ██║██╔══██╗██║╚════██║
  ██║  ██╗╚██████╔╝██║  ██║██║███████║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚══════╝
                      Unified Installer
EOF
echo -e "${plain}"
echo -e "  ${cyan}OS:${plain} ${PRETTY_NAME:-$OS} ($(uname -m))"
echo ""

# ─── Edition Selection ───────────────────────────────────────────────
echo -e "${bold}What do you want to install?${plain}"
echo ""
echo -e "  ${cyan}1)${plain} koris      — Full panel (billing, tickets, xray, reseller, all features)"
echo -e "  ${cyan}2)${plain} korislite  — Lite panel (OpenVPN, L2TP, users, nodes, settings)"
echo -e "  ${cyan}3)${plain} knode      — Node agent only (install on VPN servers)"
echo ""
read -rp "$(echo -e "${cyan}Choose [1/2/3]: ${plain}")" EDITION_CHOICE </dev/tty
case "$EDITION_CHOICE" in
    1) EDITION="full"; BINARY_NAME="koris"; SERVICE_NAME="koris"; BUILD_TAGS="" ;;
    2) EDITION="lite"; BINARY_NAME="korislite"; SERVICE_NAME="korislite"; BUILD_TAGS="-tags lite" ;;
    3) EDITION="node"; BINARY_NAME="knode"; SERVICE_NAME="knode" ;;
    *) fatal "Invalid choice. Run the script again." ;;
esac
echo ""
info "Selected: ${bold}${BINARY_NAME}${plain}"
echo ""

# ═══════════════════════════════════════════════════════════════════════
# NODE AGENT ONLY
# ═══════════════════════════════════════════════════════════════════════
if [[ "$EDITION" == "node" ]]; then
    read -rp "$(echo -e "${cyan}Panel URL (e.g. https://panel.example.com): ${plain}")" PANEL_URL </dev/tty
    [[ -z "$PANEL_URL" ]] && fatal "Panel URL is required."
    read -rp "$(echo -e "${cyan}Node token (from panel admin): ${plain}")" NODE_TOKEN </dev/tty
    [[ -z "$NODE_TOKEN" ]] && fatal "Node token is required."
    read -rp "$(echo -e "${cyan}Node name [$(hostname -s)]: ${plain}")" NODE_NAME </dev/tty
    NODE_NAME="${NODE_NAME:-$(hostname -s)}"

    info "Installing dependencies..."
    export DEBIAN_FRONTEND=noninteractive
    case "$OS" in
        ubuntu|debian)
            apt-get update -qq >/dev/null 2>&1
            apt-get install -y -qq git curl golang-go openvpn wireguard-tools strongswan xl2tpd iproute2 >/dev/null 2>&1
            ;;
        centos|almalinux|rocky|rhel|fedora)
            dnf install -y -q git curl golang openvpn wireguard-tools strongswan xl2tpd iproute >/dev/null 2>&1
            ;;
        *) fatal "Unsupported OS: $OS" ;;
    esac

    info "Downloading knode..."
    KNODE_DIR="/opt/knode"
    if [[ -d "$KNODE_DIR/.git" ]]; then
        cd "$KNODE_DIR" && git fetch origin master --depth=1 >/dev/null 2>&1 && git reset --hard origin/master >/dev/null 2>&1
    else
        rm -rf "$KNODE_DIR"
        git clone --depth=1 "https://github.com/${KNODE_REPO}.git" "$KNODE_DIR" >/dev/null 2>&1
    fi
    cd "$KNODE_DIR"

    info "Building knode..."
    go build -ldflags="-s -w" -o /usr/local/bin/knode ./cmd/node/
    chmod +x /usr/local/bin/knode

    mkdir -p /etc/knode
    cat > /etc/knode/node.env <<NENV
PANEL_URL='${PANEL_URL}'
NODE_TOKEN='${NODE_TOKEN}'
NODE_NAME='${NODE_NAME}'
NENV
    chmod 600 /etc/knode/node.env

    cat > /etc/systemd/system/knode.service <<SVC
[Unit]
Description=Koris Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/knode/node.env
ExecStart=/usr/local/bin/knode
Restart=always
RestartSec=3
User=root
WorkingDirectory=/opt/knode

[Install]
WantedBy=multi-user.target
SVC

    systemctl daemon-reload
    systemctl enable --now knode >/dev/null 2>&1
    sleep 2

    if systemctl is-active knode >/dev/null 2>&1; then
        info "knode is ${green}running${plain}"
    else
        warn "knode may have failed to start. Check: journalctl -u knode -n 20"
    fi

    echo ""
    echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
    echo -e "${bold}${green}     knode Installed Successfully!${plain}"
    echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
    echo -e "  ${cyan}Panel URL:${plain}   ${PANEL_URL}"
    echo -e "  ${cyan}Node Name:${plain}  ${NODE_NAME}"
    echo -e "  ${cyan}Config:${plain}     /etc/knode/node.env"
    echo -e "  ${cyan}Logs:${plain}       journalctl -u knode -f"
    echo -e "  ${cyan}Restart:${plain}    systemctl restart knode"
    echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
    exit 0
fi

# ═══════════════════════════════════════════════════════════════════════
# PANEL INSTALL (koris or korislite)
# ═══════════════════════════════════════════════════════════════════════

# Config prompts
read -rp "$(echo -e "${cyan}Panel port [8080]: ${plain}")" PANEL_PORT </dev/tty; PANEL_PORT="${PANEL_PORT:-8080}"
read -rp "$(echo -e "${cyan}Domain (blank for IP): ${plain}")" DOMAIN </dev/tty; DOMAIN="${DOMAIN:-_}"
read -rp "$(echo -e "${cyan}DB name [radius]: ${plain}")" DB_NAME </dev/tty; DB_NAME="${DB_NAME:-radius}"
read -rp "$(echo -e "${cyan}DB user [radius]: ${plain}")" DB_USER </dev/tty; DB_USER="${DB_USER:-radius}"
DB_PASS_DEFAULT="$(gen_secret 16)"
read -rp "$(echo -e "${cyan}DB pass [auto]: ${plain}")" DB_PASS </dev/tty; DB_PASS="${DB_PASS:-$DB_PASS_DEFAULT}"
SETUP_KEY="$(gen_secret 16)"
SESSION_SECRET="$(gen_secret 32)"
PANEL_SECRET="$(gen_secret 32)"

# Input validation
[[ ! "$DB_NAME" =~ ^[a-zA-Z0-9_]+$ ]] && fatal "Invalid DB name (alphanumeric and underscore only)"
[[ ! "$DB_USER" =~ ^[a-zA-Z0-9_]+$ ]] && fatal "Invalid DB user (alphanumeric and underscore only)"
[[ ! "$PANEL_PORT" =~ ^[0-9]+$ ]] && fatal "Port must be numeric"

echo ""
info "Installing dependencies..."
export DEBIAN_FRONTEND=noninteractive
case "$OS" in
    ubuntu|debian)
        apt-get update -qq >/dev/null 2>&1
        apt-get install -y -qq git curl openssl ca-certificates mariadb-server \
            freeradius freeradius-mysql freeradius-utils nginx golang-go iproute2 \
            wireguard-tools openvpn easy-rsa strongswan xl2tpd certbot python3-certbot-nginx >/dev/null 2>&1
        ;;
    centos|almalinux|rocky|rhel|fedora)
        dnf install -y -q git curl openssl ca-certificates mariadb-server \
            freeradius freeradius-mysql freeradius-utils nginx golang iproute \
            wireguard-tools openvpn strongswan xl2tpd certbot python3-certbot-nginx >/dev/null 2>&1
        ;;
    *) fatal "Unsupported OS: $OS" ;;
esac
info "Dependencies installed."

# Database
info "Setting up MariaDB..."
systemctl enable --now mariadb >/dev/null 2>&1
mysql -u root <<SQL
CREATE DATABASE IF NOT EXISTS ${DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
ALTER USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
CREATE USER IF NOT EXISTS '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${DB_PASS}';
ALTER USER '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'127.0.0.1';
FLUSH PRIVILEGES;
SQL
SCHEMA="/etc/freeradius/3.0/mods-config/sql/main/mysql/schema.sql"
if [[ -f "$SCHEMA" ]]; then
    mysql -u root "$DB_NAME" -N -B -e "SHOW TABLES LIKE 'radcheck';" 2>/dev/null | grep -q '^radcheck$' || mysql -u root "$DB_NAME" < "$SCHEMA"
fi

# MariaDB performance tuning
TOTAL_RAM_MB=$(free -m | awk '/Mem:/{print $2}')
INNODB_POOL="256M"
[[ $TOTAL_RAM_MB -ge 2000 ]] && INNODB_POOL="512M"
[[ $TOTAL_RAM_MB -ge 4000 ]] && INNODB_POOL="1G"
cat > /etc/mysql/mariadb.conf.d/99-koris-performance.cnf <<MYCNF
[mysqld]
innodb_buffer_pool_size = ${INNODB_POOL}
innodb_log_file_size = 128M
innodb_flush_log_at_trx_commit = 2
innodb_flush_method = O_DIRECT
max_connections = 200
thread_cache_size = 16
skip-name-resolve
MYCNF
sed -i 's/^bind-address\s*=.*/bind-address = 127.0.0.1/' /etc/mysql/mariadb.conf.d/50-server.cnf 2>/dev/null || true
systemctl restart mariadb >/dev/null 2>&1
info "Database ready."

# FreeRADIUS
info "Configuring FreeRADIUS..."
SQL_MOD="/etc/freeradius/3.0/mods-available/sql"
if [[ -f "$SQL_MOD" ]]; then
    sed -i -e 's/^\s*dialect = .*/\tdialect = "mysql"/' \
           -e "s/^\s*login = .*/\tlogin = \"${DB_USER}\"/" \
           -e "s/^\s*password = .*/\tpassword = \"${DB_PASS}\"/" \
           -e "s/^\s*radius_db = .*/\tradius_db = \"${DB_NAME}\"/" "$SQL_MOD"
    ln -sf ../mods-available/sql /etc/freeradius/3.0/mods-enabled/sql 2>/dev/null || true
    systemctl restart freeradius >/dev/null 2>&1 || true
fi

# Clone/Update panel repo
info "Downloading KorisPanel..."
if [[ -d "$INSTALL_DIR/.git" ]]; then
    cd "$INSTALL_DIR" && git fetch origin "$BRANCH" --depth=1 >/dev/null 2>&1 && git reset --hard "origin/$BRANCH" >/dev/null 2>&1
else
    rm -rf "$INSTALL_DIR"
    git clone --depth=1 -b "$BRANCH" "https://github.com/${REPO}.git" "$INSTALL_DIR" >/dev/null 2>&1
fi
cd "$INSTALL_DIR"
VERSION="$(cat VERSION 2>/dev/null || echo dev)"
info "Source ready (v${VERSION})."

# Build panel binary
info "Building ${BINARY_NAME}..."
go mod tidy >/dev/null 2>&1
go build -ldflags="-s -w" ${BUILD_TAGS} -o "/usr/local/bin/${BINARY_NAME}" ./panel/cmd/panel/
chmod +x "/usr/local/bin/${BINARY_NAME}"
info "${BINARY_NAME} built."

# Build knode (panel server also acts as a node)
info "Building knode..."
KNODE_DIR="/opt/knode"
if [[ -d "$KNODE_DIR/.git" ]]; then
    cd "$KNODE_DIR" && git fetch origin master --depth=1 >/dev/null 2>&1 && git reset --hard origin/master >/dev/null 2>&1
else
    rm -rf "$KNODE_DIR"
    git clone --depth=1 "https://github.com/${KNODE_REPO}.git" "$KNODE_DIR" >/dev/null 2>&1
fi
cd "$KNODE_DIR"
go build -ldflags="-s -w" -o /usr/local/bin/knode ./cmd/node/
chmod +x /usr/local/bin/knode
cd "$INSTALL_DIR"
info "knode built."

# VPN hook scripts
if [[ -d "$INSTALL_DIR/scripts/openvpn" ]]; then
    mkdir -p /etc/openvpn/server
    cp -f "$INSTALL_DIR/scripts/openvpn/"*.sh /etc/openvpn/server/ 2>/dev/null || true
    chmod +x /etc/openvpn/server/*.sh 2>/dev/null || true
fi

# Config
info "Writing configuration..."
PANEL_ADDR="127.0.0.1:${PANEL_PORT}"
mkdir -p "$CONFIG_DIR"
cat > "${CONFIG_DIR}/panel.env" <<ENV
PANEL_ADDR='${PANEL_ADDR}'
PANEL_DB_DSN='${DB_USER}:${DB_PASS}@tcp(127.0.0.1:3306)/${DB_NAME}?parseTime=true&multiStatements=true&charset=utf8mb4,utf8'
PANEL_MIGRATIONS='/opt/KorisPanel/panel/migrations'
PANEL_SETUP_KEY='${SETUP_KEY}'
PANEL_SESSION_SECRET='${SESSION_SECRET}'
PANEL_SECRET='${PANEL_SECRET}'
PANEL_PUBLIC_BASE='/dashboard'
PANEL_ADMIN_WEB_DIR='/opt/KorisPanel/panel/web/admin/www'
PANEL_PORTAL_WEB_DIR='/opt/KorisPanel/panel/web/portal/www'
PANEL_VERSION='${VERSION}'
ENV
chmod 600 "${CONFIG_DIR}/panel.env"

# Systemd — Panel
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<SVC
[Unit]
Description=Koris Panel (${EDITION})
After=network-online.target mariadb.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${CONFIG_DIR}/panel.env
ExecStart=/usr/local/bin/${BINARY_NAME}
Restart=always
RestartSec=3
User=root
WorkingDirectory=/opt/KorisPanel

[Install]
WantedBy=multi-user.target
SVC

# Systemd — Node Agent (local)
NODE_TOKEN="kn_$(gen_secret 24)"
mkdir -p /etc/knode
cat > /etc/knode/node.env <<NENV
PANEL_URL='http://${PANEL_ADDR}'
NODE_TOKEN='${NODE_TOKEN}'
NODE_NAME='$(hostname -s)'
NENV
chmod 600 /etc/knode/node.env

cat > /etc/systemd/system/knode.service <<SVC
[Unit]
Description=Koris Node Agent
After=network-online.target ${SERVICE_NAME}.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/knode/node.env
ExecStart=/usr/local/bin/knode
Restart=always
RestartSec=3
User=root
WorkingDirectory=/opt/knode

[Install]
WantedBy=multi-user.target
SVC

systemctl daemon-reload
systemctl enable --now "${SERVICE_NAME}" >/dev/null 2>&1
systemctl enable --now knode >/dev/null 2>&1
sleep 2

# Health check
if curl -fsS "http://${PANEL_ADDR}/api/health" >/dev/null 2>&1; then
    info "Health check ${green}PASSED${plain}"
else
    warn "Health check failed — checking logs:"
    journalctl -u "${SERVICE_NAME}" -n 20 --no-pager
fi

# Nginx
info "Configuring Nginx..."
if [[ ! -f "${CONFIG_DIR}/cert.pem" ]]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout "${CONFIG_DIR}/key.pem" -out "${CONFIG_DIR}/cert.pem" \
        -subj "/CN=${DOMAIN}" >/dev/null 2>&1
fi
cat > /etc/nginx/sites-available/koris.conf <<NGINX
server {
    listen 80 default_server;
    server_name ${DOMAIN};
    return 301 https://\$host\$request_uri;
}
server {
    listen 443 ssl default_server;
    server_name ${DOMAIN};
    client_max_body_size 20m;
    ssl_certificate ${CONFIG_DIR}/cert.pem;
    ssl_certificate_key ${CONFIG_DIR}/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    location = / { return 302 /dashboard/; }
    location = /dashboard { return 302 /dashboard/; }
    location /dashboard/ { proxy_pass http://${PANEL_ADDR}; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto https; }
    location /api/ { proxy_pass http://${PANEL_ADDR}; proxy_http_version 1.1; proxy_set_header Upgrade \$http_upgrade; proxy_set_header Connection "upgrade"; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto https; }
    location = /portal { return 302 /portal/; }
    location /portal/ { proxy_pass http://${PANEL_ADDR}; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto https; }
}
NGINX
rm -f /etc/nginx/sites-enabled/default 2>/dev/null
ln -sf /etc/nginx/sites-available/koris.conf /etc/nginx/sites-enabled/koris.conf
nginx -t >/dev/null 2>&1 && systemctl reload nginx

# Swap
if ! swapon --show | grep -q '/'; then
    SWAP_SIZE="2G"
    [[ $TOTAL_RAM_MB -ge 4000 ]] && SWAP_SIZE="4G"
    fallocate -l "$SWAP_SIZE" /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048 status=none
    chmod 600 /swapfile && mkswap /swapfile >/dev/null 2>&1 && swapon /swapfile
    grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
    sysctl -w vm.swappiness=10 >/dev/null 2>&1
    info "Swap configured (${SWAP_SIZE})."
fi

# Management CLI
cp "$INSTALL_DIR/koris.sh" /usr/local/bin/koris 2>/dev/null || true
chmod +x /usr/local/bin/koris 2>/dev/null || true

# Result
SERVER_IP=$(curl -fsS4 --max-time 3 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')
echo ""
echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
echo -e "${bold}${green}     ${BINARY_NAME} Installed Successfully!${plain}"
echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
echo -e "  ${cyan}Edition:${plain}    ${EDITION}"
echo -e "  ${cyan}Dashboard:${plain}  https://${SERVER_IP}/dashboard/"
echo -e "  ${cyan}Portal:${plain}     https://${SERVER_IP}/portal/"
echo -e "  ${cyan}Setup Key:${plain}  ${yellow}${SETUP_KEY}${plain}"
echo -e "  ${cyan}DB Pass:${plain}    ${DB_PASS}"
echo -e "  ${cyan}Node Token:${plain} ${NODE_TOKEN}"
echo -e "  ${cyan}Version:${plain}    ${VERSION}"
echo -e "${bold}${green}───────────────────────────────────────────────${plain}"
echo -e "  ${cyan}Manage:${plain}     koris"
echo -e "  ${cyan}SSL:${plain}        certbot --nginx -d ${DOMAIN}"
echo -e "  ${cyan}Logs:${plain}       journalctl -u ${SERVICE_NAME} -f"
echo -e "  ${cyan}Restart:${plain}    systemctl restart ${SERVICE_NAME}"
echo -e "${bold}${green}═══════════════════════════════════════════════${plain}"
echo ""
echo -e "${yellow}Open the Dashboard and use the Setup Key above to create your admin account.${plain}"
