#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="$ROOT/bin/fogcast"
URL="http://localhost:8080"

if [[ ! -x "$BINARY" ]]; then
  echo "Binary not found or not executable: $BINARY"
  echo "Run ./scripts/build.sh first."
  exit 1
fi

open_browser() {
  if command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$URL" >/dev/null 2>&1 || true
  elif command -v open >/dev/null 2>&1; then
    open "$URL" >/dev/null 2>&1 || true
  elif command -v start >/dev/null 2>&1; then
    start "$URL" >/dev/null 2>&1 || true
  else
    echo "Could not detect a browser opener command."
    echo "Open this URL manually: $URL"
  fi
}

"$BINARY" &
SERVER_PID=$!

trap 'kill "$SERVER_PID" 2>/dev/null || true' EXIT

sleep 0.5
open_browser

wait "$SERVER_PID"
