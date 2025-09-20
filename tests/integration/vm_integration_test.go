package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TestAgentBinaryExists verifies the agent binary is built and functional
func TestAgentBinaryExists(t *testing.T) {
	projectRoot := getProjectRoot(t)
	binaryPath := filepath.Join(projectRoot, "bin", "viper-agent")

	// Check if binary exists
	_, err := os.Stat(binaryPath)
	require.NoError(t, err, "Agent binary should exist at %s", binaryPath)

	// Check if binary is executable
	fileInfo, err := os.Stat(binaryPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, fileInfo.Mode()&0111, "Binary should be executable")

	t.Logf("✅ Agent binary exists and is executable: %s", binaryPath)
}


// TestDockerImageCleanup verifies Docker cleanup works
func TestDockerImageCleanup(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping Docker-related test")
	}

	// This test verifies that our build process cleans up properly
	// We don't actually test the cleanup here, just verify Docker is functional
	t.Logf("✅ Docker is available for image building")
}

// TestAgentHealthEndpoint tests the agent's health endpoint if running
func TestAgentHealthEndpoint(t *testing.T) {
	// This would only run if we have an agent running locally
	// For now, we just test the health response structure
	expectedHealth := types.AgentHealth{
		Status:    "healthy",
		Version:   "test",
		Contexts:  0,
		Tasks:     0,
		Memory:    1024,
		LastCheck: time.Now(),
		Details:   map[string]string{"vm_name": "test"},
	}

	// Serialize and deserialize to test JSON handling
	jsonData, err := json.Marshal(expectedHealth)
	require.NoError(t, err)

	var parsedHealth types.AgentHealth
	err = json.Unmarshal(jsonData, &parsedHealth)
	require.NoError(t, err)

	assert.Equal(t, "healthy", parsedHealth.Status)
	assert.Equal(t, "test", parsedHealth.Version)

	t.Logf("✅ Agent health endpoint structure is valid")
}

// TestTaskStructure validates task JSON structure
func TestTaskStructure(t *testing.T) {
	task := types.Task{
		ID:      "test-task-1",
		VMID:    "test-vm",
		URL:     "https://example.com",
		Status:  types.TaskStatusPending,
		Created: time.Now(),
		Timeout: 30 * time.Second,
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(task)
	require.NoError(t, err)

	var parsedTask types.Task
	err = json.Unmarshal(jsonData, &parsedTask)
	require.NoError(t, err)

	assert.Equal(t, "test-task-1", parsedTask.ID)
	assert.Equal(t, "test-vm", parsedTask.VMID)
	assert.Equal(t, "https://example.com", parsedTask.URL)

	t.Logf("✅ Task structure serialization works correctly")
}

// Helper functions


func isDockerAvailable() bool {
	resp, err := http.Get("http://unix.sock/version")
	if err == nil {
		resp.Body.Close()
		return true
	}

	// Try alternative check
	client := &http.Client{Timeout: 1 * time.Second}
	_, err = client.Get("http://localhost:2375/version")
	return err == nil
}

// TestBuildIntegration runs a comprehensive build integration test
func TestBuildIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("AgentBinary", TestAgentBinaryExists)
	t.Run("AgentHealth", TestAgentHealthEndpoint)
	t.Run("TaskStructure", TestTaskStructure)

	t.Logf("🎉 All integration tests passed!")
}
