package git

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
)

func getBackend(t *testing.T) *Backend {
	url := os.Getenv("TEST_GIT_URL")
	if url == "" {
		t.Log("git tests require TEST_GIT_URL")
		t.Skip()
	}

	b := &Backend{Backend: backendSchema()}
	b.Backend.ConfigureFunc = b.configure

	// Get the backend
	backend.TestBackendConfig(t, b, map[string]interface{}{
		"url":          url,
		"path":         fmt.Sprintf("tf-unit/%s", time.Now().String()),
		"ssh_key_path": os.Getenv("TEST_SSH_KEY_PATH"),
	})

	return b
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	b := getBackend(t)

	// Test
	backend.TestBackend(t, b)
}
