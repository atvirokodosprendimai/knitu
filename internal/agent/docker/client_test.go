package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	"github.com/atvirokodosprendimai/knitu/internal/spec"
)

func TestPrepareTemplates(t *testing.T) {
	// 1. Setup
	c, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create new Docker client: %v", err)
	}

	task := &messaging.DeployTask{
		DeploymentSpec: spec.DeploymentSpec{
			Templates: []spec.Template{
				{
					Content:     "Hello, {{ .Name }}!",
					Destination: "/test/hello.txt",
				},
			},
		},
	}

	templateData := struct {
		Name string
	}{
		Name: "Knit",
	}

	// 2. Execute the function
	mounts, err := c.prepareTemplates(task, templateData)
	if err != nil {
		t.Fatalf("c.prepareTemplates failed: %v", err)
	}

	// 3. Assertions
	if len(mounts) != 1 {
		t.Fatalf("Expected 1 mount, but got %d", len(mounts))
	}

	mount := mounts[0]
	// Clean up the temp file and its parent dir after the test
	defer os.RemoveAll(filepath.Dir(mount.Source))

	if mount.Target != "/test/hello.txt" {
		t.Errorf("Expected mount target to be '/test/hello.txt', but got '%s'", mount.Target)
	}

	if !strings.HasPrefix(filepath.Base(mount.Source), "template-") {
		t.Errorf("Expected mount source to be a temp file, but got '%s'", mount.Source)
	}

	// Verify the content of the rendered file
	content, err := os.ReadFile(mount.Source)
	if err != nil {
		t.Fatalf("Failed to read rendered template file: %v", err)
	}

	expectedContent := "Hello, Knit!"
	if string(content) != expectedContent {
		t.Errorf("Expected file content to be '%s', but got '%s'", expectedContent, string(content))
	}
}
