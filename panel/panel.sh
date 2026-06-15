#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
case "${1:-help}" in
  run) cd "$ROOT" && PANEL_MIGRATIONS="$ROOT/panel/migrations" go run ./panel/cmd/panel ;;
  build) cd "$ROOT" && mkdir -p dist && go build -o dist/panel ./panel/cmd/panel ;;
  schema) cat "$ROOT/panel/migrations/001_init.sql" ;;
  *) echo "usage: panel.sh run|build|schema" ;;
esac
