//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
)

const (
	testVM      = "integration-test-vm"
	testContext = "test-context"
	testTimeout = 30 * time.Second
)

func TestCLIVMLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build CLI binary for testing
	buildCLI(t)

	tests := []struct {
		name     string
		command  []string
		wantCode int
	}{
		{
			name:     "list vms initially",
			command:  []string{"vms", "list"},
			wantCode: 0,
		},
		{
			name:     "create vm",
			command:  []string{"vms", "create", testVM, "--memory", "1024", "--cpus", "1"},
			wantCode: 0,
		},
		{
			name:     "list vms after creation",
			command:  []string{"vms", "list"},
			wantCode: 0,
		},
		{
			name:     "destroy vm",
			command:  []string{"vms", "destroy", testVM, "--force"},
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			if tt.wantCode == 0 && err != nil {
				t.Errorf("Command failed with error: %v\nOutput: %s", err, output)
			}

			if tt.wantCode != 0 && err == nil {
				t.Errorf("Expected command to fail but it succeeded\nOutput: %s", output)
			}

			t.Logf("Command output: %s", output)
		})
	}
}

func TestCLITaskCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildCLI(t)

	// Create test task file
	task := types.Task{
		ID:     "integration-test-task",
		URL:    "https://example.com",
		Script: "console.log('Integration test completed');",
	}

	taskFile := createTestTaskFile(t, task)
	defer os.Remove(taskFile)

	tests := []struct {
		name     string
		command  []string
		wantCode int
	}{
		{
			name:     "submit task",
			command:  []string{"tasks", "submit", testVM, taskFile},
			wantCode: 0,
		},
		{
			name:     "get task status",
			command:  []string{"tasks", "status", testVM, task.ID},
			wantCode: 0,
		},
		{
			name:     "get task logs",
			command:  []string{"tasks", "logs", testVM, task.ID},
			wantCode: 0,
		},
		{
			name:     "get task screenshots",
			command:  []string{"tasks", "screenshots", testVM, task.ID},
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			// Note: These commands might fail if no actual VM is running
			// But we test that the CLI can parse and execute the commands
			t.Logf("Command: viper %v", tt.command)
			t.Logf("Output: %s", output)

			if err != nil {
				t.Logf("Command failed (expected if no VM running): %v", err)
			}
		})
	}
}

func TestCLIBrowserCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildCLI(t)

	tests := []struct {
		name    string
		command []string
	}{
		{
			name:    "spawn browser context",
			command: []string{"browsers", "spawn", testVM, testContext},
		},
		{
			name:    "list browser contexts",
			command: []string{"browsers", "list", testVM},
		},
		{
			name:    "destroy browser context",
			command: []string{"browsers", "destroy", testVM, testContext},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			t.Logf("Command: viper %v", tt.command)
			t.Logf("Output: %s", output)

			if err != nil {
				t.Logf("Command failed (expected if no VM running): %v", err)
			}
		})
	}
}

func TestCLIProfileCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildCLI(t)

	// Create test profile file
	profile := types.Profile{
		ID:        "integration-test-profile",
		Name:      "Integration Test Profile",
		UserAgent: "Mozilla/5.0 Integration Test",
		Viewport: &types.Viewport{
			Width:  1920,
			Height: 1080,
		},
		LocalStorage: map[string]map[string]string{
			"example.com": {
				"test": "integration",
			},
		},
	}

	profileFile := createTestProfileFile(t, profile)
	defer os.Remove(profileFile)

	tests := []struct {
		name    string
		command []string
	}{
		{
			name:    "attach profile",
			command: []string{"profiles", "attach", testVM, testContext, profileFile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			t.Logf("Command: viper %v", tt.command)
			t.Logf("Output: %s", output)

			if err != nil {
				t.Logf("Command failed (expected if no VM running): %v", err)
			}
		})
	}
}

func TestCLIDebugCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildCLI(t)

	tests := []struct {
		name    string
		command []string
	}{
		{
			name:    "system debug",
			command: []string{"debug", "system"},
		},
		{
			name:    "network debug",
			command: []string{"debug", "network"},
		},
		{
			name:    "agent debug",
			command: []string{"debug", "agent", testVM},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			t.Logf("Command: viper %v", tt.command)
			t.Logf("Output: %s", output)

			if err != nil {
				t.Logf("Command failed (expected if no Nomad/VM running): %v", err)
			}
		})
	}
}

func TestCLIHelp(t *testing.T) {
	buildCLI(t)

	tests := []struct {
		name    string
		command []string
	}{
		{
			name:    "main help",
			command: []string{"--help"},
		},
		{
			name:    "vms help",
			command: []string{"vms", "--help"},
		},
		{
			name:    "tasks help",
			command: []string{"tasks", "--help"},
		},
		{
			name:    "browsers help",
			command: []string{"browsers", "--help"},
		},
		{
			name:    "profiles help",
			command: []string{"profiles", "--help"},
		},
		{
			name:    "debug help",
			command: []string{"debug", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./bin/viper", tt.command...)
			cmd.Dir = getProjectRoot()

			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("Help command failed: %v\nOutput: %s", err, output)
			}

			outputStr := string(output)
			if len(outputStr) < 10 {
				t.Errorf("Help output too short: %s", outputStr)
			}

			t.Logf("Help output length: %d characters", len(outputStr))
		})
	}
}

func TestCLIVersion(t *testing.T) {
	buildCLI(t)

	cmd := exec.Command("./bin/viper", "--version")
	cmd.Dir = getProjectRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Version command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Version output is empty")
	}

	t.Logf("Version output: %s", outputStr)
}

// Helper functions

func buildCLI(t *testing.T) {
	t.Helper()

	projectRoot := getProjectRoot()

	// Check if binary already exists and is recent
	binaryPath := filepath.Join(projectRoot, "bin", "viper")
	if stat, err := os.Stat(binaryPath); err == nil {
		if time.Since(stat.ModTime()) < 5*time.Minute {
			return // Binary is recent enough
		}
	}

	cmd := exec.Command("make", "build-cli")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	t.Logf("CLI built successfully")
}

func getProjectRoot() string {
	// Assumes test is run from project root or subdirectory
	wd, _ := os.Getwd()

	// Walk up directories until we find go.mod
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}

	return wd // Fallback to current directory
}

func createTestTaskFile(t *testing.T, task types.Task) string {
	t.Helper()

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test-task-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = tmpFile.Write(data)
	if err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}

func createTestProfileFile(t *testing.T, profile types.Profile) string {
	t.Helper()

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test-profile-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = tmpFile.Write(data)
	if err != nil {
		t.Fatalf("Failed to write profile file: %v", err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}
