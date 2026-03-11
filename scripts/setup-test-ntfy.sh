#!/usr/bin/env bash
set -euo pipefail

NTFY_URL="${NTFY_URL:-http://localhost:8080}"
MAX_WAIT=60
INTERVAL=2

echo "==> Waiting for ntfy to be healthy at ${NTFY_URL}/v1/health ..."

elapsed=0
while true; do
    if curl -sf "${NTFY_URL}/v1/health" > /dev/null 2>&1; then
        echo "==> ntfy is healthy!"
        break
    fi
    if [ "$elapsed" -ge "$MAX_WAIT" ]; then
        echo "ERROR: ntfy did not become healthy within ${MAX_WAIT}s"
        exit 1
    fi
    sleep "$INTERVAL"
    elapsed=$((elapsed + INTERVAL))
done

echo "==> Creating admin user ..."
docker compose exec -T \
    -e NTFY_PASSWORD=admin \
    -e NTFY_AUTH_FILE=/var/lib/ntfy/auth.db \
    ntfy ntfy user add --role=admin admin 2>/dev/null || {
    echo "==> Admin user may already exist, continuing ..."
}

echo "==> Verifying admin credentials ..."
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -u admin:admin "${NTFY_URL}/v1/health")
if [ "$HTTP_CODE" = "200" ]; then
    echo "==> Admin credentials verified (HTTP ${HTTP_CODE})"
else
    echo "ERROR: Admin credentials check failed (HTTP ${HTTP_CODE})"
    exit 1
fi

echo "==> ntfy test infrastructure ready!"
echo "    URL:      ${NTFY_URL}"
echo "    Username: admin"
echo "    Password: admin"
