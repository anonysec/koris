#!/bin/bash
# deploy-report.sh — Posts deploy diagnostics to a GitHub Issue for remote debugging.
#
# Requires:
#   - GITHUB_TOKEN env var with `repo` scope (create at https://github.com/settings/tokens)
#   - GITHUB_REPO set to "owner/repo" (defaults to "anonysec/panel")
#   - A "deploy-log" label created on the target GitHub repository
#
# Each deploy creates a new GitHub Issue containing:
#   - Service status (active/inactive/failed)
#   - Panel version
#   - Health check response
#   - Last 50 lines of panel service logs
#
# Usage:
#   Set GITHUB_TOKEN and GITHUB_REPO in /etc/panel/panel.env or export them before running.
#   This script is called automatically at the end of deploy.sh.

GITHUB_REPO="${GITHUB_REPO:-anonysec/panel}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
ISSUE_TITLE="[Auto] Deploy Report - $(date '+%Y-%m-%d %H:%M:%S')"

if [ -z "$GITHUB_TOKEN" ]; then
    echo "[deploy-report] GITHUB_TOKEN not set, skipping log upload"
    exit 0
fi

# Collect diagnostics
PANEL_LOGS=$(journalctl -u panel -n 50 --no-pager -o short-iso 2>/dev/null || echo "journalctl not available")
SERVICE_STATUS=$(systemctl is-active panel 2>/dev/null || echo "unknown")
HEALTH_CHECK=$(curl -s --max-time 5 http://127.0.0.1:${PANEL_PORT:-8088}/api/health 2>/dev/null || echo "health check failed")
PANEL_VERSION=$(cat /opt/koris-next/VERSION 2>/dev/null || echo "unknown")

# Build issue body as markdown
BODY=$(cat <<EOF
## Deploy Report — $(date '+%Y-%m-%d %H:%M:%S UTC')

### Service Status: \`$SERVICE_STATUS\`
### Version: \`$PANEL_VERSION\`

### Health Check
\`\`\`json
$HEALTH_CHECK
\`\`\`

### Last 50 Panel Logs
\`\`\`
$PANEL_LOGS
\`\`\`
EOF
)

# Escape for JSON
BODY_JSON=$(echo "$BODY" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo "$BODY" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')

# Create GitHub issue
curl -s -X POST \
    -H "Authorization: token $GITHUB_TOKEN" \
    -H "Accept: application/vnd.github.v3+json" \
    "https://api.github.com/repos/$GITHUB_REPO/issues" \
    -d "{\"title\": \"$ISSUE_TITLE\", \"body\": $BODY_JSON, \"labels\": [\"deploy-log\"]}" \
    > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "[deploy-report] Diagnostics posted to GitHub issue"
else
    echo "[deploy-report] Failed to post to GitHub"
fi
