# Knit Specification

## 1. Introduction

Knit is a lightweight, distributed container orchestration system designed for simplicity and ease of use. It leverages the Docker API to manage container lifecycles, NATS for communication between components, and GORM for state persistence. The goal is to provide a subset of features found in systems like Kubernetes or Nomad, with a focus on running containerized applications on a flat network.

## 2. System Architecture

Knit follows a server-agent model.

*   **Knit Server:** The central control plane. It's responsible for:
    *   Providing a RESTful API (built with Chi) for users and the web dashboard.
    *   Maintaining the desired state of the cluster (deployments, networks, etc.) in a database (SQLite via GORM).
    *   Publishing tasks and commands to agents via NATS.
    *   Aggregating status and heartbeats from agents.

*   **Knit Agent:** A lightweight agent running on every host in the cluster. It's responsible for:
    *   Registering itself with the server on startup.
    *   Sending periodic heartbeats to the server.
    *   Subscribing to NATS subjects to listen for tasks (e.g., deploy container, delete container).
    *   Interacting directly with the local Docker daemon to manage containers, networks, and volumes.
    *   Reporting back the status of tasks to the server.

## 3. Communication

All communication between the server and agents is asynchronous and handled by NATS.

*   **NATS Server:** A central NATS server (or cluster) is required.
*   **Subjects:** Specific NATS subjects will be used for different types of messages:
    *   `knit.agent.heartbeat`: Agents publish their status and heartbeat here.
    *   `knit.tasks.{node-id}`: The server publishes node-specific tasks to these subjects (e.g., `knit.tasks.node-123`).
    *   `knit.tasks.broadcast`: The server publishes tasks for any available agent.
    *   `knit.task.status`: Agents publish the results of their tasks here.

## 4. Data Models (GORM / SQLite)

The server will use GORM with the `modernc/sqlite` driver (CGO-free) to persist its state.

*   `Node`: Represents a worker host.
    *   `ID`: Unique identifier.
    *   `Hostname`: Hostname of the node.
    *   `Status`: (e.g., "healthy", "unhealthy").
    *   `LastHeartbeat`: Timestamp of the last heartbeat.
*   `Deployment`: The specification for a set of containers.
    *   `ID`: Unique identifier.
    *   `Name`: User-defined name for the deployment.
    *   `Image`: Docker image to use.
    *   `RegistryCredentialsID`: Foreign key to private registry credentials.
    *   `NetworkAttachments`: List of networks to attach the container to.
    *   `Templates`: Configuration for file templates to be mounted into the container.
*   `ContainerInstance`: Represents a running container managed by Knit.
    *   `ID`: Docker's container ID.
    *   `NodeID`: The node it's running on.
    *   `DeploymentID`: The deployment it belongs to.
    *   `Status`: (e.g., "running", "stopped", "error").
*   `Network`: A Docker network managed by Knit.
    *   `ID`: Docker's network ID.
    *   `Name`: User-defined name.
    *   `Driver`: (e.g., "overlay").
    *   `Subnet`: IP range for the network.
*   `RegistryCredentials`: Stores credentials for private Docker registries.
    *   `ID`: Unique identifier.
    *   `URL`: Registry URL.
    *   `Username`: Username.
    *   `Password`: (Encrypted).

## 5. Core Workflows

### 5.1. Agent Registration

1.  A `knit-agent` starts on a host.
2.  It generates a unique node ID (or retrieves a previously generated one).
3.  It publishes a registration message with its hostname and other info to the `knit.agent.heartbeat` subject.
4.  The server, subscribed to this subject, receives the message and creates or updates the `Node` record in its database.

### 5.2. Container Deployment

1.  A user submits a `Deployment` specification to the server's `POST /deployments` API endpoint.
2.  The server validates the spec and stores it in the database.
3.  The server determines which node to deploy to (based on a scheduling algorithm, or broadcast).
4.  The server publishes a "deploy task" message to a NATS subject (e.g., `knit.tasks.broadcast`).
5.  An available agent receives the task.
6.  The agent processes the task:
    *   If `RegistryCredentialsID` is provided, it fetches the credentials.
    *   It authenticates with the private registry.
    *   It pulls the specified Docker image.
    *   If `Templates` are defined, it renders them (see below).
    *   It creates the container using the Docker API, attaching the specified networks and mounting the rendered template files.
    *   It starts the container.
7.  The agent publishes the result (success or failure, with container ID) to the `knit.task.status` subject.
8.  The server receives the status and updates the `ContainerInstance` record.

### 5.3. Configuration Templating (Nomad-style)

This feature allows for dynamic file creation inside containers.

1.  The `Deployment` specification includes a `templates` array. Each element contains:
    *   `Content`: The raw template content (using Go's `text/template` format).
    *   `Destination`: The absolute path where the file should be mounted inside the container (e.g., `/app/config.json`).

2.  When the agent receives the deployment task, it performs these steps for each template:
    *   Creates a temporary directory on the host (e.g., `/tmp/knit-templates-12345/`).
    *   For each template in the spec, it creates a file inside this directory.
    *   It parses the `Content` using Go's `text/template` engine.
    *   It executes the template, writing the rendered output to the temporary file. (Note: Currently, no data is passed to the template, but this can be extended).
    *   In the `docker create` command, it configures a bind mount from the temporary host file to the `Destination` path in the container.

This provides a powerful way to inject configuration, connection strings, or any other dynamic data into a container at runtime. The temporary directory on the host is not automatically cleaned up to allow for inspection and debugging.

## 6. Networking

Knit will use [wg-mesh](https://github.com/atvirokodosprendimai/wg-mesh) to create a secure, flat, peer-to-peer network for the entire cluster. This WireGuard-based mesh provides a robust foundation for node discovery, secure communication, and simplified network topology.

*   **Prerequisite:** `wg-mesh` is treated as a prerequisite that must be running on all nodes. Knit does not manage the `wg-mesh` lifecycle.

*   **Node Discovery:** The Knit Server connects to the `wg-mesh` JSON-RPC API via its Unix socket (`/var/run/wgmesh.sock` by default).
    *   Periodically, the server calls the `peers.list` RPC method to get a full list of all nodes in the mesh.
    *   It then "discovers" these nodes by creating a `Node` record in its own database for each peer, using the peer's WireGuard public key as the unique `NodeID`. This keeps Knit's view of the cluster in sync with the mesh topology.

*   **Secure Communication:** The Knit Server's embedded NATS and HTTP services are configured to bind to a specific IP address (via the `--nats-addr` and `--http-addr` flags). To secure the control plane, this should be the server's WireGuard IP.
    *   Agents are then configured with the server's WireGuard IP and NATS port (`--nats-url`) to ensure all communication (heartbeats, tasks) happens over the encrypted WireGuard tunnels.

*   **Container Networking:** While the control plane operates on the WireGuard mesh, container-to-container networking can still leverage Docker's native capabilities. For cross-host container communication, applications can be exposed on their host's WireGuard IP address. (This is a future roadmap item).
