#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
remove_if_exists() {
  local p="$1"
  if [[ -e "$p" ]]; then
    rm -rf "$p"
    echo "Removed: $p"
  fi
}

# Root-level artifacts.
remove_if_exists "$ROOT/bin"
remove_if_exists "$ROOT/node_modules"
remove_if_exists "$ROOT/test-results"
remove_if_exists "$ROOT/playwright-report"
remove_if_exists "$ROOT/artifacts"

# Web app artifacts.
for app in dm player; do
  remove_if_exists "$ROOT/web/$app/node_modules"
  remove_if_exists "$ROOT/web/$app/dist"
done

# Built static assets.
for app in dm player; do
  remove_if_exists "$ROOT/static/$app"
done

echo "Clean complete."
