#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
case "${1:-help}" in
  run) shift || true; cd "$ROOT" && go run ./node/cmd/node ;;
  build) cd "$ROOT" && mkdir -p dist && go build -o dist/node ./node/cmd/node ;;
  *) echo "usage: node.sh run|build" ;;
esac
