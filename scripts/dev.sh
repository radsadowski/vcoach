#!/usr/bin/env bash
# Dev hot loop: postgres via compose, Go API (air hot reload), Vite dev server.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
export DATABASE_URL="${DATABASE_URL:-postgres://vcoach:vcoach@localhost:5432/vcoach?sslmode=disable}"
export PORT="${PORT:-8080}"

docker compose up -d db

cleanup() {
  # shellcheck disable=SC2046
  kill $(jobs -p) 2>/dev/null || true
}
trap cleanup EXIT INT TERM

(cd server && exec air) &
(cd web && exec npx vite) &

echo
echo "  api : http://localhost:${PORT} (hot reload via air)"
echo "  web : http://localhost:5173"
echo

wait
