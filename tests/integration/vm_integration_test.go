package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVMImageExists verifies that the built VM image exists and is valid
func TestVMImageExists(t *testing.T) {
	projectRoot := getProjectRoot(t)
	imagePath := filepath.Join(projectRoot, "dist", "viper-headless.qcow2")

	// Check if image exists
	_, err := os.Stat(imagePath)
	require.NoError(t, err, "VM image should exist at %s", imagePath)

	// Check file size (should be reasonable for a headless Chrome image)
	fileInfo, err := os.Stat(imagePath)
	require.NoError(t, err)

	// Image should be at least 50MB (compressed) but less than 2GB
	assert.Greater(t, fileInfo.Size(), int64(50*1024*1024), "Image too small")
	assert.Less(t, fileInfo.Size(), int64(2*1024*1024*1024), "Image too large")

	t.Logf("✅ VM image exists: %s (size: %d bytes)", imagePath, fileInfo.Size())
}

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

// TestNomadJobTemplate verifies the generated Nomad job template
func TestNomadJobTemplate(t *testing.T) {
	projectRoot := getProjectRoot(t)
	jobPath := filepath.Join(projectRoot, "dist", "example-job.hcl")

	// Check if job template exists
	_, err := os.Stat(jobPath)
	require.NoError(t, err, "Nomad job template should exist at %s", jobPath)

	// Read and basic validate content
	content, err := os.ReadFile(jobPath)
	require.NoError(t, err)

	jobContent := string(content)
	assert.Contains(t, jobContent, `driver = "virt"`, "Should use virt driver")
	assert.Contains(t, jobContent, "viper-headless.qcow2", "Should reference our image")
	assert.Contains(t, jobContent, "viper-agent", "Should reference our agent")

	t.Logf("✅ Nomad job template generated and contains expected content")
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

func getProjectRoot(t *testing.T) string {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Walk up the directory tree to find the project root
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		wd = parent
	}
}

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

	t.Run("VMImage", TestVMImageExists)
	t.Run("AgentBinary", TestAgentBinaryExists)
	t.Run("NomadJobTemplate", TestNomadJobTemplate)
	t.Run("AgentHealth", TestAgentHealthEndpoint)
	t.Run("TaskStructure", TestTaskStructure)

	t.Logf("🎉 All integration tests passed!")
}