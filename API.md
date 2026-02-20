# Knit API Documentation

This document provides details on the RESTful API for the Knit orchestrator.

**Base URL:** `http://<knit-server-ip>:8080`

## Authentication

Authentication is not yet implemented. All endpoints are currently open.

---

## Endpoints

### Health Check

#### `GET /ping`

Checks if the Knit server is running and responsive.

**Request:**
*   None

**Response (200 OK):**
*   **Content-Type:** `application/json`
```json
{
  "message": "pong"
}
```

---

### Deployments

#### `POST /deployments`

Creates a new deployment. The server accepts the deployment specification, saves it, and publishes a task for an agent to execute. The API returns immediately with a `202 Accepted` status if the task is successfully dispatched.

**Request Body:**
*   **Content-Type:** `application/json`

| Field       | Type     | Description                                                                                             | Required |
|-------------|----------|---------------------------------------------------------------------------------------------------------|----------|
| `name`      | `string` | A unique name for the deployment.                                                                       | Yes      |
| `image`     | `string` | The Docker image to pull (e.g., `nginx:latest`).                                                        | Yes      |
| `registry`  | `object` | (Optional) Credentials for a private registry.                                                          | No       |
| `env`       | `object` | (Optional) A map of environment variables to set in the container (e.g., `"VAR": "value"`).              | No       |
| `ports`     | `array`  | (Optional) A list of port mappings.                                                                     | No       |
| `templates` | `array`  | (Optional) A list of file templates to render and mount into the container.                             | No       |
| `network`   | `string` | (Optional) The name of a pre-existing Docker network to attach the container to.                        | No       |

**Registry Object:**
| Field      | Type     | Description | Required |
|------------|----------|-------------|----------|
| `username` | `string` | Username for the private registry. | Yes |
| `password` | `string` | Password for the private registry. | Yes |

**Port Object:**
| Field           | Type     | Description | Required |
|-----------------|----------|-------------|----------|
| `host_port`     | `integer`| The port on the host machine. | Yes |
| `container_port`| `integer`| The port inside the container. | Yes |

**Template Object:**
| Field           | Type     | Description | Required |
|-----------------|----------|-------------|----------|
| `content`       | `string` | The content of the template (Go template format). | Yes |
| `destination`   | `string` | The absolute path to mount the file inside the container. | Yes |


**Example Request with Template:**
```json
{
  "name": "my-config-app",
  "image": "busybox:latest",
  "templates": [
    {
      "destination": "/etc/config.json",
      "content": "{\n  \"greeting\": \"Hello from Knit\",\n  \"port\": {{ .Env.PORT | default 8080 }}\n}"
    },
    {
      "destination": "/etc/another_file.txt",
      "content": "This is a static file."
    }
  ]
}
```

**Responses:**

*   **202 Accepted:** The deployment task was successfully accepted and published. The response body contains the database record for the new deployment.
    ```json
    {
      "ID": 1,
      "CreatedAt": "2023-10-27T10:00:00Z",
      "UpdatedAt": "2023-10-27T10:00:00Z",
      "DeletedAt": null,
      "Name": "my-config-app",
      "Image": "busybox:latest",
      "RegistryCredentialsID": 0,
      "NetworkAttachments": "",
      "Templates": "[{\"destination\":\"/etc/config.json\",\"content\":\"{\\n  \\\"greeting\\\": \\\"Hello from Knit\\\",\\n  \\\"port\\\": {{ .Env.PORT | default 8080 }}\\n}\"},{\"destination\":\"/etc/another_file.txt\",\"content\":\"This is a static file.\"}]"
    }
    ```

*   **400 Bad Request:** The request body is malformed or contains invalid JSON.
    ```text
    Invalid request body: <error details>
    ```

*   **500 Internal Server Error:** The server encountered an error while saving the deployment or publishing the task.
    ```text
    Failed to save deployment: <error details>
    ```
