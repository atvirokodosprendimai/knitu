#!/usr/bin/env bash
set -euo pipefail

# Deploy nginx without node selector (broadcast scheduling).

KNIT_SERVER_URL="${KNIT_SERVER_URL:-http://127.0.0.1:8080}"
HOST_PORT="${HOST_PORT:-8081}"

read -r -d '' PAYLOAD <<JSON || true
{
  "name": "nginx-broadcast-${HOST_PORT}",
  "image": "nginx:latest",
  "ports": [
    {
      "host_port": ${HOST_PORT},
      "container_port": 80
    }
  ]
}
JSON

curl -sS -X POST "${KNIT_SERVER_URL}/deployments" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD"
echo
