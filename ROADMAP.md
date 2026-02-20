# Knit Development Roadmap

This document outlines the development phases for building the Knit container orchestrator.

## Phase 1: Core Infrastructure (Foundation)

The goal of this phase is to set up the project structure and establish the basic communication and data persistence layers.

- [ ] **Project Scaffolding:**
    - [ ] Create the standard Go project directory structure (`/cmd`, `/internal`, `/pkg`).
    - [ ] Initialize `go.mod`.
- [ ] **Initial Dependencies:**
    - [ ] Add core dependencies:
        - `github.com/go-chi/chi/v5` (Web framework)
        - `gorm.io/gorm` (ORM)
        - `gorm.io/driver/sqlite` with `github.com/glebarez/sqlite` (CGO-free SQLite)
        - `github.com/nats-io/nats.go` (NATS client)
        - `github.com/docker/docker` (Docker client)
- [ ] **Application Skeletons:**
    - [ ] Create `cmd/knit-server/main.go` with a basic Chi server setup.
    - [ ] Create `cmd/knit-agent/main.go` with a basic application loop.
- [ ] **Database & Models:**
    - [ ] Define initial GORM models in `/internal/db/models.go` for `Node`, `Deployment`, and `ContainerInstance`.
    - [ ] Implement a database initialization function for the server.
- [ ] **Communication:**
    - [ ] Implement a wrapper for NATS connection handling.
    - [ ] Server: Implement logic to listen for agent heartbeats.
    - [ ] Agent: Implement logic to send periodic heartbeats.

## Phase 2: Container Deployment

This phase focuses on the primary workflow: deploying a container based on a user's request.

- [ ] **API Endpoint:**
    - [ ] Create a `POST /deployments` endpoint in the Chi router.
    - [ ] Define the JSON structure for a deployment request.
- [ ] **Server Logic:**
    - [ ] Implement the handler to validate and store the deployment request in the database.
    - [ ] Logic to publish a "deploy task" to a NATS subject.
- [ ] **Agent Logic:**
    - [ ] Agent subscribes to the NATS "deploy task" subject.
    - [ ] Implement a Docker client wrapper in the agent.
    - [ ] Core logic to pull an image and create/start a Docker container based on the task details.
    - [ ] Implement status reporting back to the server via NATS.
- [ ] **Private Registries:**
    - [ ] Add data model for `RegistryCredentials`.
    - [ ] Implement API endpoints to manage credentials (securely).
    - [ ] Enhance agent logic to use credentials when pulling images.

## Phase 3: Networking with WireGuard Mesh

This phase implements the flat, secure networking model using `wg-mesh`.

- [ ] **wg-mesh Integration:**
    - [ ] Add `wg-mesh` as a dependency or submodule.
    - [ ] Develop logic for the Knit Server to act as the `wg-mesh` orchestrator or integrate with an external one.
- [ ] **Agent Integration:**
    - [ ] Implement logic within the Knit Agent to join the WireGuard mesh on startup. This includes generating keys and fetching configuration.
- [ ] **Secure Communication:**
    - [ ] Configure NATS client and server communication to use the WireGuard interface and IP addresses.
- [ ] **Container Network Exposure:**
    - [ ] Update the deployment specification to allow services running in containers to be exposed on the host's WireGuard IP address.

## Phase 4: Configuration Templating

This phase adds the Nomad-inspired feature of rendering configuration files from templates and mounting them into containers.

- [ ] **API & Data Model:**
    - [ ] Extend the `Deployment` model and API to accept a `templates` array, with fields for `content` and `destination`.
- [ ] **Agent-side Rendering:**
    - [ ] Implement a templating engine in the agent using Go's `text/template`.
    - [ ] Before creating a container, the agent renders the template content to a temporary file on the host.
- [ ] **Volume Mounting:**
    - [ ] The agent will configure a bind mount in the Docker create command to mount the temporary host file to the specified destination path inside the container.

## Phase 5: Dashboard & Usability

This phase focuses on making the system observable and easier to use.

- [ ] **Web Dashboard:**
    - [ ] Create a simple web UI using Go's `html/template` package, served by the Chi server.
    - [ ] The dashboard should list nodes, deployments, and their statuses.
    - [ ] Provide a simple form to create new deployments.
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
