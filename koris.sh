#!/usr/bin/env bash
#
# KorisPanel Management CLI
# Usage: koris [command]
#

red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; blue='\033[0;34m'; cyan='\033[0;36m'; bold='\033[1m'; dim='\033[2m'; plain='\033[0m'
info()  { echo -e "${green}[+]${plain} $*"; }
warn()  { echo -e "${yellow}[!]${plain} $*"; }
error() { echo -e "${red}[-]${plain} $*"; }

INSTALL_DIR="/opt/KorisPanel"
PANEL_ENV="/etc/koris/panel.env"
NODE_ENV="/etc/knode/node.env"
COMPOSE_FILE="${INSTALL_DIR}/docker-compose.yml"

is_docker() { [[ -f "$COMPOSE_FILE" ]] && command -v docker &>/dev/null; }
is_panel() { is_docker || [[ -f /usr/local/bin/koris && -f "$PANEL_ENV" ]]; }
is_node()  { [[ -f /usr/local/bin/knode && -f "$NODE_ENV" ]] || docker ps --format '{{.Names}}' 2>/dev/null | grep -qx knode; }
get_version() { cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "?"; }

panel_status() {
    if is_docker; then
        docker inspect -f '{{.State.Status}}' koris 2>/dev/null || echo "not running"
    else
        systemctl is-active koris 2>/dev/null || echo "not installed"
    fi
}
node_status() {
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx knode; then
        echo "running (docker)"
    elif systemctl is-active knode 2>/dev/null; then
        echo "running"
    else
        echo "not running"
    fi
}

cmd_start() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    if is_docker; then
        cd "$INSTALL_DIR" && docker compose up -d && info "Panel started"
    else
        systemctl start koris && info "Panel started"
        systemctl start knode 2>/dev/null && info "Node started"
    fi
}
cmd_stop() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    if is_docker; then
        cd "$INSTALL_DIR" && docker compose down && info "Panel stopped"
    else
        systemctl stop koris && info "Panel stopped"
        systemctl stop knode 2>/dev/null && info "Node stopped"
    fi
}
cmd_restart() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    if is_docker; then
        cd "$INSTALL_DIR" && docker compose restart && info "Panel restarted"
    else
        systemctl restart koris && info "Panel restarted"
        systemctl restart knode 2>/dev/null && info "Node restarted"
    fi
}

cmd_status() {
    echo -e "${bold}${blue}KorisPanel${plain} v$(get_version)"
    echo "───────────────────────────────────"
    printf "  %-14s %s\n" "Panel:" "$(panel_status)"
    printf "  %-14s %s\n" "Node Agent:" "$(node_status)"
    if is_panel; then
        local addr=$(grep 'PANEL_ADDR' "$PANEL_ENV" 2>/dev/null | cut -d= -f2 | tr -d "'\"")
        printf "  %-14s %s\n" "Listen:" "${addr:-?}"
        curl -fsS "http://${addr}/api/health" >/dev/null 2>&1 && printf "  %-14s ${green}%s${plain}\n" "Health:" "OK" || printf "  %-14s ${red}%s${plain}\n" "Health:" "FAIL"
    fi
    echo "───────────────────────────────────"
    printf "  %-14s %s\n" "CPU:" "$(nproc) cores"
    printf "  %-14s %s\n" "RAM:" "$(free -h | awk '/^Mem:/{print $3"/"$2}')"
    printf "  %-14s %s\n" "Disk:" "$(df -h / | awk 'NR==2{print $3"/"$2" ("$5")"}')"
}

cmd_logs() {
    if is_docker; then
        cd "$INSTALL_DIR" && docker compose logs --tail 50
    else
        is_panel && { echo -e "${cyan}=== Panel ===${plain}"; journalctl -u koris -n 50 --no-pager; }
        is_node  && { echo -e "${cyan}=== Node ===${plain}"; journalctl -u knode -n 50 --no-pager; }
    fi
}

cmd_follow() {
    if is_docker; then
        cd "$INSTALL_DIR" && exec docker compose logs -f
    else
        is_panel && exec journalctl -u koris -f
    fi
}

cmd_update() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    [[ ! -d "$INSTALL_DIR/.git" ]] && { error "Not a git install"; exit 1; }
    cd "$INSTALL_DIR"
    local old=$(get_version)
    git fetch origin main --depth=1 >/dev/null 2>&1
    git reset --hard origin/main >/dev/null 2>&1
    local new=$(get_version)
    [[ "$old" == "$new" ]] && { info "Already up to date (v${new})."; return; }
    info "Updating v${old} -> v${new}..."
    bash "$INSTALL_DIR/deploy.sh"
    # Update self
    cp "$INSTALL_DIR/koris.sh" /usr/local/bin/koris 2>/dev/null; chmod +x /usr/local/bin/koris 2>/dev/null
    info "Done: v${new}"
}

cmd_uninstall() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    echo -e "${red}This will remove KorisPanel completely.${plain}"
    read -rp "Type 'yes' to confirm: " c; [[ "$c" != "yes" ]] && exit 0
    systemctl stop koris knode 2>/dev/null; systemctl disable koris knode 2>/dev/null
    rm -f /etc/systemd/system/koris.service /etc/systemd/system/knode.service
    systemctl daemon-reload
    rm -f /usr/local/bin/koris /usr/local/bin/knode
    rm -rf /etc/koris /etc/knode "$INSTALL_DIR"
    rm -f /etc/nginx/sites-enabled/koris-panel.conf /etc/nginx/sites-available/koris-panel.conf
    systemctl reload nginx 2>/dev/null || true
    info "Uninstalled. Database not removed (manual cleanup needed)."
}

cmd_config() {
    is_panel && { echo -e "${cyan}Panel Config:${plain}"; grep -v 'SECRET\|PASSWORD\|TOKEN' "$PANEL_ENV" 2>/dev/null | sed 's/^/  /'; echo "  (secrets hidden)"; }
    is_node  && { echo -e "${cyan}Node Config:${plain}"; grep -v 'TOKEN' "$NODE_ENV" 2>/dev/null | sed 's/^/  /'; echo "  (token hidden)"; }
}

cmd_ssl() {
    [[ $EUID -ne 0 ]] && { error "Need root"; exit 1; }
    echo -e "${bold}${cyan}SSL Certificate Manager${plain}"
    echo ""

    # Show current SSL status
    if [[ -f /etc/letsencrypt/live/*/fullchain.pem ]] 2>/dev/null; then
        CERT_DOMAIN=$(ls /etc/letsencrypt/live/ 2>/dev/null | grep -v README | head -1)
        if [[ -n "$CERT_DOMAIN" ]]; then
            EXPIRY=$(openssl x509 -enddate -noout -in "/etc/letsencrypt/live/${CERT_DOMAIN}/fullchain.pem" 2>/dev/null | cut -d= -f2)
            echo -e "  ${green}●${plain} SSL active for: ${cyan}${CERT_DOMAIN}${plain}"
            echo -e "  ${dim}Expires: ${EXPIRY}${plain}"
            echo ""
        fi
    else
        echo -e "  ${yellow}●${plain} No SSL certificate found"
        echo ""
    fi

    echo -e "  ${green}1.${plain} Install/renew SSL certificate"
    echo -e "  ${green}2.${plain} Force renewal"
    echo -e "  ${green}3.${plain} Remove SSL (revert to HTTP)"
    echo -e "  ${green}0.${plain} Back"
    echo ""
    read -rp "$(echo -e "${cyan}Choose: ${plain}")" ssl_ch

    case "$ssl_ch" in
        1)
            read -rp "$(echo -e "${cyan}Domain: ${plain}")" SSL_DOMAIN
            [[ -z "$SSL_DOMAIN" ]] && { error "Domain required."; return; }
            read -rp "$(echo -e "${cyan}Email (for renewal notices, blank to skip): ${plain}")" SSL_EMAIL

            # Install certbot if missing
            if ! command -v certbot >/dev/null 2>&1; then
                info "Installing Certbot..."
                if [[ -f /etc/debian_version ]]; then
                    apt-get update -qq >/dev/null 2>&1
                    apt-get install -y -qq certbot python3-certbot-nginx >/dev/null 2>&1
                else
                    dnf install -y -q certbot python3-certbot-nginx >/dev/null 2>&1
                fi
            fi

            if ! command -v certbot >/dev/null 2>&1; then
                error "Failed to install certbot."; return
            fi

            # Update nginx server_name
            if [[ -f /etc/nginx/sites-available/koris-panel.conf ]]; then
                sed -i "s/server_name .*/server_name ${SSL_DOMAIN};/" /etc/nginx/sites-available/koris-panel.conf
                nginx -t >/dev/null 2>&1 && systemctl reload nginx
            fi

            # Obtain certificate
            EMAIL_ARG="--register-unsafely-without-email"
            [[ -n "$SSL_EMAIL" ]] && EMAIL_ARG="--email $SSL_EMAIL"

            info "Requesting certificate for ${SSL_DOMAIN}..."
            if certbot --nginx -d "$SSL_DOMAIN" --non-interactive --agree-tos $EMAIL_ARG --redirect; then
                info "SSL certificate installed successfully!"
                # Enable secure cookies
                if grep -q '^PANEL_SECURE_COOKIES=' "$PANEL_ENV" 2>/dev/null; then
                    sed -i "s|^PANEL_SECURE_COOKIES=.*|PANEL_SECURE_COOKIES='true'|" "$PANEL_ENV"
                else
                    echo "PANEL_SECURE_COOKIES='true'" >> "$PANEL_ENV"
                fi
                systemctl restart koris
                echo ""
                echo -e "  ${green}✓${plain} https://${SSL_DOMAIN}/dashboard/"
                echo -e "  ${dim}Auto-renewal is enabled via certbot timer.${plain}"
            else
                error "Certbot failed. Check that:"
                echo "  - DNS A record for ${SSL_DOMAIN} points to this server"
                echo "  - Port 80 is open (for HTTP-01 challenge)"
                echo "  - Nginx is running"
            fi
            ;;
        2)
            info "Forcing certificate renewal..."
            certbot renew --force-renewal
            nginx -t >/dev/null 2>&1 && systemctl reload nginx
            info "Done."
            ;;
        3)
            read -rp "$(echo -e "${yellow}Remove SSL and revert to HTTP? [y/N]: ${plain}")" confirm
            [[ "$confirm" != "y" && "$confirm" != "Y" ]] && return
            # Rewrite nginx config without SSL
            PANEL_ADDR="$(grep -E '^PANEL_ADDR=' "$PANEL_ENV" 2>/dev/null | cut -d= -f2 | tr -d "'\"")"
            PANEL_ADDR="${PANEL_ADDR:-127.0.0.1:8080}"
            cat > /etc/nginx/sites-available/koris-panel.conf <<NGINX
server {
    listen 80 default_server;
    server_name _;
    client_max_body_size 20m;
    location = / { return 302 /dashboard/; }
    location = /dashboard { return 302 /dashboard/; }
    location /dashboard/ { proxy_pass http://${PANEL_ADDR}; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto \$scheme; }
    location /api/ { proxy_pass http://${PANEL_ADDR}; proxy_http_version 1.1; proxy_set_header Upgrade \$http_upgrade; proxy_set_header Connection "upgrade"; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto \$scheme; }
    location = /portal { return 302 /portal/; }
    location /portal/ { proxy_pass http://${PANEL_ADDR}; proxy_set_header Host \$host; proxy_set_header X-Real-IP \$remote_addr; proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto \$scheme; }
}
NGINX
            nginx -t >/dev/null 2>&1 && systemctl reload nginx
            # Disable secure cookies
            sed -i "s|^PANEL_SECURE_COOKIES=.*|PANEL_SECURE_COOKIES='false'|" "$PANEL_ENV" 2>/dev/null
            systemctl restart koris
            info "SSL removed. Panel is now HTTP-only."
            ;;
        0) return ;;
        *) warn "Invalid." ;;
    esac
}

show_menu() {
    clear
    echo -e "${bold}${blue}KorisPanel${plain} v$(get_version)    Panel: $(panel_status)  Node: $(node_status)"
    echo ""
    echo -e "  ${green}1.${plain} Start       ${green}5.${plain} Logs          ${green}9.${plain}  Enable autostart"
    echo -e "  ${green}2.${plain} Stop        ${green}6.${plain} Live logs     ${green}10.${plain} Disable autostart"
    echo -e "  ${green}3.${plain} Restart     ${green}7.${plain} Update        ${green}11.${plain} Uninstall"
    echo -e "  ${green}4.${plain} Status      ${green}8.${plain} Config        ${green}12.${plain} SSL Certificate"
    echo -e "                                    ${green}0.${plain}  Exit"
    echo ""
    read -rp "$(echo -e "${cyan}Choose: ${plain}")" ch
    case "$ch" in
        1) cmd_start;; 2) cmd_stop;; 3) cmd_restart;; 4) cmd_status;;
        5) cmd_logs;; 6) cmd_follow;; 7) cmd_update;; 8) cmd_config;;
        9) systemctl enable koris knode 2>/dev/null; info "Enabled.";;
        10) systemctl disable koris knode 2>/dev/null; info "Disabled.";;
        11) cmd_uninstall;; 12) cmd_ssl;; 0) exit 0;; *) warn "Invalid.";;
    esac
    echo ""; read -rp "Enter to continue..." _; show_menu
}

case "${1:-}" in
    start)     cmd_start;; stop) cmd_stop;; restart) cmd_restart;;
    status)    cmd_status;; logs) cmd_logs;; follow|logs-live) cmd_follow;;
    update)    cmd_update;; config) cmd_config;; uninstall) cmd_uninstall;;
    ssl)       cmd_ssl;;
    enable)    systemctl enable koris knode 2>/dev/null; info "Enabled.";;
    disable)   systemctl disable koris knode 2>/dev/null; info "Disabled.";;
    node-status)  echo "Node Agent: $(node_status)";;
    node-restart) systemctl restart knode 2>/dev/null; info "Node restarted.";;
    node-logs)    journalctl -u knode -n 50 --no-pager;;
    help|-h|--help) echo "Usage: koris [start|stop|restart|status|logs|follow|update|config|ssl|uninstall|enable|disable|node-status|node-restart|node-logs]"; echo "Run without args for interactive menu.";;
    "") show_menu;;
    *) error "Unknown: $1. Run 'koris help'."; exit 1;;
esac
