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

Knit follows a server-agent model where all components communicate over a secure WireGuard network.

```
+-------------------------------------------------------------------------+
|                              WireGuard Mesh                             |
|                                (wg-mesh)                                |
|                                                                         |
|  +----------------+      +------------------+      +------------------+  |
|  |  Knit Server   |<---->|   NATS Server    |<---->|   Knit Agent 1   |  |
|  | (Orchestrator) |      +------------------+      |  (Node 1)        |  |
|  +----------------+                                +------------------+  |
|         ^                                                  |            |
|         |                                                  v            |
|         | API Requests                               +-----------+      |
|         +------------------[ User ]                   |  Docker   |      |
|                                                      +-----------+      |
+-------------------------------------------------------------------------+

```

1.  **Knit Server:** The central control plane. It exposes a REST API (built with Chi), persists state in a SQLite database via GORM, and dispatches tasks to agents via NATS.
2.  **Knit Agent:** A lightweight agent running on each worker node. It listens for tasks, interacts with the local Docker daemon, and reports status back to the server.
3.  **NATS:** Acts as the message bus for all asynchronous communication.
4.  **wg-mesh:** Establishes a secure, peer-to-peer VPN, ensuring all traffic between nodes is encrypted and authenticated.

## Getting Started

### Prerequisites

- Go (1.18+)
- Docker
- A running NATS server accessible from the server and agents.
- WireGuard tools installed on all nodes.

### Building from Source

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/atvirokodosprendimai/knitu.git
    cd knitu
    ```

2.  **Build the server and agent binaries:**
    ```sh
    # Build the server
    go build -o knit-server ./cmd/knit-server

    # Build the agent
    go build -o knit-agent ./cmd/knit-agent
    ```

### Configuration

The server and agent can be configured via environment variables or configuration files. Example configuration files will be available in the `/configs` directory.

- **Knit Server:** Needs to be configured with the NATS server address and database path.
- **Knit Agent:** Needs the NATS server address and configuration for joining the `wg-mesh`.

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
