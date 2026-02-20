# Knit Development Roadmap

This document outlines the development phases for building the Knit container orchestrator.

## Phase 1: Core Infrastructure (Foundation)

The goal of this phase is to set up the project structure and establish the basic communication and data persistence layers.

- [x] **Project Scaffolding:**
    - [x] Create the standard Go project directory structure (`/cmd`, `/internal`, `/pkg`).
    - [x] Initialize `go.mod`.
- [x] **Initial Dependencies:**
    - [x] Add core dependencies:
        - `github.com/go-chi/chi/v5` (Web framework)
        - `gorm.io/gorm` (ORM)
        - `github.com/glebarez/sqlite` (CGO-free SQLite)
        - `github.com/nats-io/nats.go` (NATS client)
        - `github.com/moby/moby` (Docker client)
- [x] **Application Skeletons:**
    - [x] Create `cmd/knit-server/main.go` with a basic Chi server setup.
    - [x] Create `cmd/knit-agent/main.go` with a basic application loop.
- [x] **Database & Models:**
    - [x] Define initial GORM models in `/internal/db/models.go`.
    - [x] Implement a database initialization function for the server.
- [x] **Communication:**
    - [x] Implement a wrapper for NATS connection handling.
    - [x] Server: Implement logic to listen for agent heartbeats.
    - [x] Agent: Implement logic to send periodic heartbeats.

## Phase 2: Container Deployment

This phase focuses on the primary workflow: deploying a container based on a user's request.

- [x] **API Endpoint:**
    - [x] Create a `POST /deployments` endpoint in the Chi router.
    - [x] Define the JSON structure for a deployment request.
- [x] **Server Logic:**
    - [x] Implement the handler to validate and store the deployment request in the database.
    - [x] Logic to publish a "deploy task" to a NATS subject.
- [x] **Agent Logic:**
    - [x] Agent subscribes to the NATS "deploy task" subject.
    - [x] Implement a Docker client wrapper in the agent.
    - [x] Core logic to pull an image and create/start a Docker container based on the task details.
    - [x] Implement status reporting back to the server via NATS.
- [ ] **Private Registries:**
    - [x] Add data model for `RegistryCredentials`.
    - [ ] Implement API endpoints to manage credentials (securely).
    - [x] Enhance agent logic to use credentials when pulling images.

## Phase 3: Networking with WireGuard Mesh

This phase implements the flat, secure networking model using `wg-mesh`.

- [x] **wg-mesh Integration:**
    - [ ] Add `wg-mesh` as a dependency or submodule.
    - [x] Develop logic for the Knit Server to discover nodes via the `wg-mesh` JSON-RPC socket.
- [ ] **Agent Integration:**
    - [ ] Implement logic within the Knit Agent to join the WireGuard mesh on startup. (Handled by `wg-mesh` itself).
    - [x] Agent can be configured with the server's WireGuard IP to connect.
- [x] **Secure Communication:**
    - [x] Configure NATS client and server communication to use the WireGuard interface and IP addresses via flags.
- [ ] **Container Network Exposure:**
    - [ ] Update the deployment specification to allow services running in containers to be exposed on the host's WireGuard IP address.

## Phase 4: Configuration Templating

This phase adds the Nomad-inspired feature of rendering configuration files from templates and mounting them into containers.

- [x] **API & Data Model:**
    - [x] Extend the `Deployment` model and API to accept a `templates` array.
- [x] **Agent-side Rendering:**
    - [x] Implement a templating engine in the agent using Go's `text/template`.
    - [x] Before creating a container, the agent renders the template content to a temporary file on the host.
- [x] **Volume Mounting:**
    - [x] The agent configures a bind mount in the Docker create command to mount the temporary host file to the specified destination path inside the container.

## Phase 5: Dashboard & Usability

This phase focuses on making the system observable and easier to use.

- [ ] **Web Dashboard:**
    - [x] Create a simple web UI using Go's `html/template` package, served by the Chi server.
    - [x] The dashboard lists nodes, deployments, and container instance statuses.
    - [x] Provide forms to deploy and undeploy workloads.
    - [ ] Replace periodic refresh with SSE/DataStar reactive updates.
- [ ] **CLI Client:**
    - [ ] Develop a separate `knit-cli` application.
    - [ ] The CLI will interact with the Knit server's REST API.
    - [ ] Implement commands like `knit deploy`, `knit status`, `knit list nodes`.

## Future Goals

- [ ] **Health Checks:** Implement container health checks and automated restarts.
- [ ] **Rolling Updates:** A strategy for updating deployments with zero downtime.
- [ ] **Secrets Management:** A secure way to provide secrets to containers.
- [ ] **Metrics & Logging:** Expose metrics for monitoring and provide a way to aggregate container logs.
- [ ] **Advanced Scheduling:** More sophisticated scheduling algorithms (e.g., resource-based).
