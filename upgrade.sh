#!/bin/bash
# KorisPanel Lite → Full upgrade script
# This migrates your Lite installation to the full KorisPanel
#
# Prerequisites:
#   - Lite panel is running and has data
#   - Full panel binary is downloaded
#
# What this does:
#   1. Stops Lite panel
#   2. Backs up the database
#   3. Switches to the full panel binary (same database)
#   4. Full panel's migrations handle schema differences
#   5. Starts full panel on its port (8080)
#
# Usage: bash upgrade.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}╔══════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  KorisPanel Lite → Full Upgrade      ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════╝${NC}"
echo ""

# Configuration
LITE_SERVICE="koris-lite"
FULL_SERVICE="koris-panel"
FULL_BINARY_URL="${FULL_BINARY_URL:-}"
DB_NAME="${DB_NAME:-radius_lite}"
BACKUP_DIR="/opt/koris-lite/backups"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Step 1: Create backup
echo -e "${YELLOW}[1/5] Creating database backup...${NC}"
mkdir -p "$BACKUP_DIR"
BACKUP_FILE="$BACKUP_DIR/pre-upgrade-$(date +%Y%m%d_%H%M%S).sql"
mysqldump "$DB_NAME" > "$BACKUP_FILE"
echo -e "  Backup saved: $BACKUP_FILE"

# Step 2: Stop Lite panel
echo -e "${YELLOW}[2/5] Stopping Lite panel...${NC}"
if systemctl is-active --quiet "$LITE_SERVICE" 2>/dev/null; then
    systemctl stop "$LITE_SERVICE"
    echo -e "  $LITE_SERVICE stopped"
else
    echo -e "  $LITE_SERVICE not running (OK)"
fi

# Step 3: Download full panel binary (if URL provided)
echo -e "${YELLOW}[3/5] Setting up full panel...${NC}"
if [ -n "$FULL_BINARY_URL" ]; then
    echo -e "  Downloading full panel binary..."
    wget -q -O /usr/local/bin/panel "$FULL_BINARY_URL"
    chmod +x /usr/local/bin/panel
    echo -e "  Downloaded to /usr/local/bin/panel"
elif [ -f "/usr/local/bin/panel" ]; then
    echo -e "  Full panel binary already at /usr/local/bin/panel"
else
    echo -e "${RED}  No full panel binary found and FULL_BINARY_URL not set.${NC}"
    echo -e "${RED}  Download the full panel binary to /usr/local/bin/panel first.${NC}"
    echo -e "${RED}  Or set: export FULL_BINARY_URL=https://...${NC}"
    # Restore lite service
    systemctl start "$LITE_SERVICE" 2>/dev/null || true
    exit 1
fi

# Step 4: Configure full panel to use same database
echo -e "${YELLOW}[4/5] Configuring full panel...${NC}"
mkdir -p /etc/panel
if [ -f "/etc/koris-lite/.env" ]; then
    # Copy lite env and adjust
    cp /etc/koris-lite/.env /etc/panel/panel.env
    # Change port from 9080 to 8080
    sed -i 's/PANEL_ADDR=:9080/PANEL_ADDR=:8080/' /etc/panel/panel.env
    # Update migrations path
    sed -i 's|PANEL_MIGRATIONS=.*|PANEL_MIGRATIONS=/opt/KorisPanel/panel/migrations|' /etc/panel/panel.env
    echo -e "  Config copied to /etc/panel/panel.env (port changed to 8080)"
else
    echo -e "  Creating new config from Lite defaults..."
    cat > /etc/panel/panel.env << 'ENVEOF'
PANEL_ADDR=:8080
PANEL_DB_DSN=radius:radius@tcp(127.0.0.1:3306)/radius_lite?parseTime=true&multiStatements=true
PANEL_SESSION_SECRET=change-me-in-production-now!!!
ENVEOF
fi

# Step 5: Start full panel
echo -e "${YELLOW}[5/5] Starting full panel...${NC}"
if [ -f "/etc/systemd/system/$FULL_SERVICE.service" ]; then
    systemctl daemon-reload
    systemctl start "$FULL_SERVICE"
    echo -e "  $FULL_SERVICE started"
else
    echo -e "  No systemd service for full panel. Start manually:"
    echo -e "  /usr/local/bin/panel"
fi

# Disable lite service so it doesn't auto-start
systemctl disable "$LITE_SERVICE" 2>/dev/null || true

echo ""
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo -e "${GREEN}  Upgrade complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo ""
echo -e "  Full panel: http://$(hostname -I | awk '{print $1}'):8080"
echo -e "  Database:   $DB_NAME (same data, migrations auto-applied)"
echo -e "  Backup:     $BACKUP_FILE"
echo ""
echo -e "  ${YELLOW}Note: The full panel will run additional migrations on first${NC}"
echo -e "  ${YELLOW}start to add tables for billing, themes, statistics, etc.${NC}"
echo -e "  ${YELLOW}Your existing users and nodes are preserved.${NC}"
echo ""
