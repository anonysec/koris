#!/usr/bin/env bash
set -euo pipefail
[ "$(id -u)" = 0 ] || { echo "run as root"; exit 1; }
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPENVPN_DIR="${OPENVPN_DIR:-/etc/openvpn/server}"
CONF="${OPENVPN_CONF:-$OPENVPN_DIR/server.conf}"
install -m 0755 "$ROOT/scripts/openvpn/koris-radius-auth.sh" "$OPENVPN_DIR/koris-radius-auth.sh"
install -m 0755 "$ROOT/scripts/openvpn/koris-client-connect.sh" "$OPENVPN_DIR/koris-client-connect.sh"
install -m 0755 "$ROOT/scripts/openvpn/koris-client-disconnect.sh" "$OPENVPN_DIR/koris-client-disconnect.sh"
cp -a "$CONF" "$CONF.bak.$(date +%s)"
# Disable old radiusplugin line; auth/accounting are handled by scripts.
sed -i 's#^plugin /usr/lib/openvpn/radiusplugin.so #;plugin /usr/lib/openvpn/radiusplugin.so #' "$CONF"
# Replace old client connect/disconnect hooks with Koris wrappers.
sed -i '/^auth-user-pass-verify /d;/^client-connect /d;/^client-disconnect /d' "$CONF"
awk '
  { print }
  /^username-as-common-name$/ { print "auth-user-pass-verify /etc/openvpn/server/koris-radius-auth.sh via-file" }
  /^script-security / { seen=1 }
  END { if (!seen) print "script-security 2"; print "client-connect /etc/openvpn/server/koris-client-connect.sh"; print "client-disconnect /etc/openvpn/server/koris-client-disconnect.sh" }
' "$CONF" > "$CONF.tmp"
mv "$CONF.tmp" "$CONF"
systemctl restart openvpn-server@server || systemctl restart openvpn@server
