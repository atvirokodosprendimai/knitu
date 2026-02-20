# Knit

Knit is a lightweight container orchestrator built on Docker, NATS, and `wg-mesh`.

## Features

- Server/agent architecture.
- Embedded NATS in the server.
- Agent heartbeats and task status reporting.
- Deployment API (`POST /deployments`).
- Undeploy API (`DELETE /deployments/{name}`).
- Node-aware scheduling with `node_selector` and agent `labels`.
- Nomad-like template rendering to real files + bind mounts.
- Host port exposure (`host_ip:host_port -> container_port`).
- `wg-mesh` peer discovery via JSON-RPC socket.

## Architecture

- `knit-server`
  - HTTP API (chi)
  - embedded NATS
  - SQLite via GORM
  - `wg-mesh` peer discovery (`/var/run/wgmesh.sock` by default)
- `knit-agent`
  - connects to NATS
  - executes deployment tasks via Docker API
  - sends heartbeat with node labels

## Build

```sh
go build -o knit-server ./cmd/knit-server
go build -o knit-agent ./cmd/knit-agent
```

## Run

### Server

Bind HTTP and NATS to the server WireGuard IP.

```sh
./knit-server start \
  --http-addr "10.54.0.1:8080" \
  --nats-addr "10.54.0.1:4222" \
  --wg-mesh-socket "/var/run/wgmesh.sock"
```

### Agent

Point to server NATS URL and add labels:

```sh
./knit-agent start \
  --nats-url "nats://10.54.0.1:4222" \
  --labels "region=eu,role=api,ssd=true"
```

## Deployment Example

```json
{
  "name": "nginx-eu-api",
  "image": "nginx:latest",
  "node_selector": {
    "region": "eu",
    "role": "api"
  },
  "ports": [
    {
      "host_ip": "10.54.0.15",
      "host_port": 8080,
      "container_port": 80
    }
  ],
  "templates": [
    {
      "destination": "/usr/share/nginx/html/index.html",
      "content": "<h1>Hello from Knit</h1><p>Rendered by agent.</p>"
    }
  ]
}
```

Send it:

```sh
curl -X POST http://10.54.0.1:8080/deployments \
  -H "Content-Type: application/json" \
  -d @deployment.json
```

## Scripts

See `scripts/` for runnable examples.

## Dashboard

- Open `http://<server-ip>:8080/dashboard`
- Supports deploy + undeploy forms and live status views (auto-refresh).

## Notes

- If `node_selector` is omitted, task is published to broadcast subject.
- Templates are rendered using Go `text/template`.
- Temporary template files are currently not auto-cleaned after deployment.

## Docs

- API: `API.md`
- Spec: `SPEC.md`
- Roadmap: `ROADMAP.md`
