package spec

// DeploymentSpec defines the structure for a user's deployment request.
// This is what is sent to the API.
type DeploymentSpec struct {
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Registry  RegistryAuth      `json:"registry,omitempty"`
	Network   string            `json:"network,omitempty"`
	Templates []Template        `json:"templates,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Ports     []PortBinding     `json:"ports,omitempty"`
}

// RegistryAuth defines credentials for a private container registry.
type RegistryAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Template defines a file to be rendered and mounted into a container.
type Template struct {
	Content     string `json:"content"`
	Destination string `json:"destination"`
}

// PortBinding defines a host-to-container port mapping.
type PortBinding struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`
}
