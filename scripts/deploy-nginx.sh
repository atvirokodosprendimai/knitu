#!/usr/bin/env bash
set -euo pipefail

# Deploy nginx to a label-selected node with host port mapping.
# Required agent label example: region=eu,role=api

KNIT_SERVER_URL="${KNIT_SERVER_URL:-http://127.0.0.1:8080}"
HOST_IP="${HOST_IP:-10.54.0.15}"
HOST_PORT="${HOST_PORT:-8080}"
REGION="${REGION:-eu}"
ROLE="${ROLE:-api}"

read -r -d '' PAYLOAD <<JSON || true
{
  "name": "nginx-labeled-${HOST_PORT}",
  "image": "nginx:latest",
  "node_selector": {
    "region": "${REGION}",
    "role": "${ROLE}"
  },
  "ports": [
    {
      "host_ip": "${HOST_IP}",
      "host_port": ${HOST_PORT},
      "container_port": 80
    }
  ],
  "templates": [
    {
      "destination": "/usr/share/nginx/html/index.html",
      "content": "<h1>Knit Nginx</h1><p>region=${REGION} role=${ROLE}</p>"
    }
  ]
}
JSON

echo "POST ${KNIT_SERVER_URL}/deployments"
echo "$PAYLOAD"

curl -sS -X POST "${KNIT_SERVER_URL}/deployments" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD"
echo
