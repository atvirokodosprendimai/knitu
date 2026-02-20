# Knit

[![Go Report Card](https://goreportcard.com/badge/github.com/atvirokodosprendimai/knitu)](https://goreportcard.com/report/github.com/atvirokodosprendimai/knitu)

Knit is a lightweight, distributed container orchestration system designed for simplicity and ease of use. It leverages the Docker API to manage container lifecycles, NATS for asynchronous communication, and a WireGuard mesh for secure networking.

The goal of Knit is to provide a simple, secure, and robust platform for running containerized applications, drawing inspiration from the simplicity of projects like HashiCorp's Nomad while running directly on the Docker engine.

## Core Features

- **Simple Server-Agent Architecture:** A central server for orchestration and lightweight agents on each node.
- **Secure by Default:** All control plane communication is secured and encrypted using a WireGuard mesh provided by `wg-mesh`.
- **Docker-Native:** Uses the Docker API directly, meaning anything that can run in Docker can be orchestrated by Knit.
- **Asynchronous Tasking:** Leverages NATS.io for a resilient and scalable messaging backbone.
- **Configuration as Code:** Deployments are defined in simple, declarative specifications.
- **Nomad-style Templating:** Dynamically generate configuration files and mount them into containers at runtime.
- **CGO-Free:** Built with a CGO-free stack (`modernc/sqlite`) for easy cross-compilation.

## Architecture

Knit follows a server-agent model where all components communicate over a secure WireGuard network established by `wg-mesh`.

```
+-----------------------------------------------------------------------------+
|                            WireGuard Mesh (wg-mesh)                         |
|                                                                             |
|  +---------------------+      +-----------------------+      +-------------+  |
|  |    Knit Server      |<---->|  Embedded NATS Server |<---->| Knit Agent  |  |
|  | (discovers via RPC) |      | (binds to wg-ip)      |      | (connects)  |  |
|  +---------------------+      +-----------------------+      +-------------+  |
|         ^                                                          |        |
|         | API Requests                                             v        |
|         +------------------------[ User ]                     +-----------+ |
|                                                              |  Docker   | |
|                                                              +-----------+ |
+-----------------------------------------------------------------------------+
```

1.  **Knit Server:** The central control plane, run with the `knit-server start` command. It embeds its own NATS server and discovers other nodes by querying the `wg-mesh` daemon via its RPC socket.
2.  **Knit Agent:** A lightweight agent, run with `knit-agent start`. It connects to the server's NATS instance (via its WireGuard IP) to receive tasks. It then interacts with the local Docker daemon to manage containers.
3.  **wg-mesh (Prerequisite):** Establishes a secure, peer-to-peer VPN, giving each node a stable IP. Knit uses this mesh for all control-plane communication.

## Getting Started

### Prerequisites

- Go (1.18+)
- Docker
- **`wg-mesh`:** Must be installed and running on all nodes, with all nodes joined to the same mesh.

### Building from Source

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/atvirokodosprendimai/knitu.git
    cd knitu
    ```

2.  **Build the binaries:**
    ```sh
    go build -o knit-server ./cmd/knit-server
    go build -o knit-agent ./cmd/knit-agent
    ```

### Configuration & Running

Knit is configured via CLI flags.

#### 1. Run the Server

The server will automatically discover other mesh nodes. You should bind the NATS and HTTP services to the server's WireGuard IP address.

Find the server's WireGuard IP using `wg-mesh status` or a similar command. Let's assume it is `10.54.0.1`.

```sh
# Run the server, binding services to its WireGuard IP
./knit-server start \
  --nats-addr="10.54.0.1:4222" \
  --http-addr="10.54.0.1:8080" \
  --wg-mesh-socket="/var/run/wgmesh.sock"
```

#### 2. Run the Agent(s)

On each agent node, you must point the agent to the NATS server running on the server node's WireGuard IP.

```sh
# Run the agent, pointing it to the server's NATS address
./knit-agent start --nats-url="nats://10.54.0.1:4222"
```

The agent will now connect to the server, send heartbeats, and be ready to receive deployment tasks.

## Roadmap

The detailed development plan is available in the [ROADMAP.md](ROADMAP.md) file. The high-level phases include:
1.  Core Infrastructure
2.  Container Deployment
3.  Networking with WireGuard Mesh
4.  Configuration Templating
5.  Dashboard & Usability

## Contributing

Contributions are welcome! Please feel free to submit a pull request. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate and follow the existing code style.

## License

This project is licensed under the MIT License.
