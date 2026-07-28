#!/bin/sh
# Sync shared core assets into language adaptors before build.
set -eu

ROOT=$(CDPATH= cd "$(dirname "$0")/.." && pwd)

CORE_MIGRATIONS="$ROOT/core/migrations"
CORE_UI_DIST="$ROOT/core/ui/dist"

GO_MIGRATIONS="$ROOT/adaptor/go/migrations"
GO_UI_DIST="$ROOT/adaptor/go/ui/dist"
PHP_UI="$ROOT/adaptor/php/resources/ui/dist"
PY_UI="$ROOT/adaptor/python/microscope/static/dist"
PY_MIGRATIONS="$ROOT/adaptor/python/microscope/migrations"

if [ -f "$GO_MIGRATIONS" ]; then
  rm -f "$GO_MIGRATIONS"
fi

mkdir -p "$GO_MIGRATIONS" "$GO_UI_DIST" "$PHP_UI" "$PY_UI" "$PY_MIGRATIONS"

cp -R "$CORE_MIGRATIONS/." "$GO_MIGRATIONS/"
cp -R "$CORE_MIGRATIONS/." "$PY_MIGRATIONS/"

if [ -d "$CORE_UI_DIST" ]; then
  cp -R "$CORE_UI_DIST/." "$GO_UI_DIST/"
  cp -R "$CORE_UI_DIST/." "$PHP_UI/"
  cp -R "$CORE_UI_DIST/." "$PY_UI/"
fi

echo "Synced core assets to adaptors."
