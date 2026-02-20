# Knit API Documentation

Base URL: `http://<knit-server-ip>:8080`

Authentication is not implemented yet.

## Endpoints

### `GET /dashboard`
Basic GUI dashboard (deploy, undeploy, node/deployment/instance views).

### `GET /ping`
Health check.

Response:
```json
{
  "message": "pong"
}
```

### `POST /deployments`
Create a deployment and enqueue a task for an agent.

Request body fields:

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | deployment name |
| `image` | string | yes | container image |
| `registry` | object | no | private registry auth |
| `templates` | array | no | files rendered with Go `text/template` |
| `ports` | array | no | host/container port mappings |
| `node_selector` | object | no | schedule on node matching labels |
| `env` | object | no | reserved, not fully wired yet |
| `network` | string | no | reserved, not fully wired yet |

Registry object:

| Field | Type | Required |
|---|---|---|
| `username` | string | yes |
| `password` | string | yes |

Template object:

| Field | Type | Required | Notes |
|---|---|---|---|
| `destination` | string | yes | absolute path inside container |
| `content` | string | yes | Go template text |

Port object:

| Field | Type | Required | Notes |
|---|---|---|---|
| `host_ip` | string | no | bind address on host, example `10.54.0.15` |
| `host_port` | integer | yes | host port |
| `container_port` | integer | yes | container port |

Node selector object:

- map of `key: value`
- deployment is sent to one healthy node whose labels match all pairs
- if not provided, deployment is broadcast to all agents (first one to process wins)

Example request:
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

Responses:

- `202 Accepted`: deployment stored and task published.
- `400 Bad Request`: invalid JSON or no matching node for selector.
- `500 Internal Server Error`: persistence or publish failure.

### `DELETE /deployments/{name}`
Queue undeploy by deployment name (broadcast to agents).

Responses:

- `202 Accepted`: undeploy queued.
- `400 Bad Request`: invalid/missing name.

Example `202` response:
```json
{
  "ID": 1,
  "CreatedAt": "2026-02-20T12:00:00Z",
  "UpdatedAt": "2026-02-20T12:00:00Z",
  "DeletedAt": null,
  "Name": "nginx-eu-api",
  "Image": "nginx:latest",
  "RegistryCredentialsID": 0,
  "NetworkAttachments": "",
  "Templates": "[{\"destination\":\"/usr/share/nginx/html/index.html\",\"content\":\"<h1>Hello from Knit</h1><p>Rendered by agent.</p>\"}]"
}
```
