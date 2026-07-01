#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

install_frontend() {
  local app_dir="$1"

  if [ -f "$app_dir/package-lock.json" ]; then
    npm --prefix "$app_dir" ci
  else
    npm --prefix "$app_dir" install
  fi
}

install_frontend "$ROOT/web/dm"
install_frontend "$ROOT/web/player"

npm --prefix "$ROOT/web/dm" run build
npm --prefix "$ROOT/web/player" run build

mkdir -p "$ROOT/bin"
go build -o "$ROOT/bin/fogcast" "$ROOT/cmd/fogcast"

