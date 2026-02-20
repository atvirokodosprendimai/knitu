package db

import (
	"time"

	"gorm.io/gorm"
)

// Node represents a worker host in the cluster.
type Node struct {
	gorm.Model
	NodeID        string `gorm:"uniqueIndex"`
	Hostname      string
	Status        string
	LastHeartbeat time.Time
}

// Deployment is the specification for a set of containers.
type Deployment struct {
	gorm.Model
	Name                  string `gorm:"uniqueIndex"`
	Image                 string
	RegistryCredentialsID uint
	NetworkAttachments    string // Simplification for now, could be a separate table
	Templates             string // Simplification for now, JSON blob
}

// ContainerInstance represents a running container managed by Knit.
type ContainerInstance struct {
	gorm.Model
	ContainerID  string `gorm:"uniqueIndex"`
	NodeID       uint
	DeploymentID uint
	Status       string
}

// RegistryCredentials stores credentials for private Docker registries.
type RegistryCredentials struct {
	gorm.Model
	URL      string `gorm:"uniqueIndex"`
	Username string
	Password string // Should be encrypted
}

// Network represents a Docker network managed by Knit.
type Network struct {
	gorm.Model
	NetworkID string `gorm:"uniqueIndex"`
	Name      string `gorm:"uniqueIndex"`
	Driver    string
	Subnet    string
}
