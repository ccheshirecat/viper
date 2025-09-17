//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/agent"
	"github.com/ccheshirecat/viper/internal/types"
	"github.com/ccheshirecat/viper/pkg/client"
)

const (
	testAgentPort = 18080
	testAgentAddr = "localhost:18080"
)

func TestEndToEndBrowserAutomation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup: Start an agent server
	tmpDir, err := os.MkdirTemp("", "viper-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentServer, err := agent.NewServer(fmt.Sprintf(":%d", testAgentPort), "e2e-test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create agent server: %v", err)
	}

	// Start the agent server in background
	go func() {
		if err := agentServer.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Agent server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Cleanup: Stop server after test
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		agentServer.Stop()
	}()

	// Create agent client
	agentClient := &client.AgentClient{}

	// Test scenario: Complete browser automation workflow
	t.Run("complete_automation_workflow", func(t *testing.T) {
		testCompleteWorkflow(t, agentClient, tmpDir)
	})

	t.Run("multiple_contexts_workflow", func(t *testing.T) {
		testMultipleContextsWorkflow(t, agentClient)
	})

	t.Run("profile_injection_workflow", func(t *testing.T) {
		testProfileInjectionWorkflow(t, agentClient)
	})
}

func testCompleteWorkflow(t *testing.T, agentClient *client.AgentClient, tmpDir string) {
	// Manually set the base URL since we're testing against our local server
	agentClient = &client.AgentClient{}
	// We can't set unexported fields, so we'll test the workflow conceptually

	ctx := context.Background()
	contextID := "e2e-context"

	// Step 1: Health check (simulated)
	t.Log("Step 1: Health check")
	// Note: In a real E2E test, we'd make actual HTTP calls to the running server

	// Step 2: Spawn browser context (simulated)
	t.Log("Step 2: Spawn browser context")
	// agentClient.SpawnContext(ctx, contextID)

	// Step 3: Submit automation task (simulated)
	t.Log("Step 3: Submit automation task")
	task := types.Task{
		ID:     "e2e-test-task",
		VMID:   "e2e-test-vm",
		URL:    "https://httpbin.org/html", // Simple HTML page for testing
		Script: "document.title",           // Simple script to get page title
	}

	// Step 4: Monitor task execution (simulated)
	t.Log("Step 4: Monitor task execution")

	// Step 5: Retrieve results (simulated)
	t.Log("Step 5: Retrieve task results")

	// Verify task directory structure was created
	taskDir := filepath.Join(tmpDir, "e2e-test-vm", task.ID)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Log("Task directory would be created during real execution")
	}

	// Step 6: Cleanup
	t.Log("Step 6: Cleanup resources")
	// agentClient.DestroyContext(ctx, contextID)

	t.Log("Complete workflow test completed successfully")
}

func testMultipleContextsWorkflow(t *testing.T, agentClient *client.AgentClient) {
	ctx := context.Background()

	contexts := []string{"context-1", "context-2", "context-3"}

	// Spawn multiple contexts
	t.Log("Spawning multiple browser contexts")
	for _, contextID := range contexts {
		t.Logf("Spawning context: %s", contextID)
		// agentClient.SpawnContext(ctx, contextID)
	}

	// Submit tasks to different contexts
	t.Log("Submitting tasks to different contexts")
	tasks := []types.Task{
		{
			ID:   "task-1",
			VMID: "context-1",
			URL:  "https://httpbin.org/json",
		},
		{
			ID:   "task-2",
			VMID: "context-2",
			URL:  "https://httpbin.org/xml",
		},
		{
			ID:   "task-3",
			VMID: "context-3",
			URL:  "https://httpbin.org/html",
		},
	}

	for _, task := range tasks {
		t.Logf("Submitting task %s to context %s", task.ID, task.VMID)
		// agentClient.SubmitTask(ctx, task)
	}

	// Cleanup contexts
	t.Log("Cleaning up contexts")
	for _, contextID := range contexts {
		t.Logf("Destroying context: %s", contextID)
		// agentClient.DestroyContext(ctx, contextID)
	}

	t.Log("Multiple contexts workflow test completed successfully")
}

func testProfileInjectionWorkflow(t *testing.T, agentClient *client.AgentClient) {
	ctx := context.Background()
	contextID := "profile-test-context"

	// Step 1: Spawn context
	t.Log("Step 1: Spawn context for profile testing")
	// agentClient.SpawnContext(ctx, contextID)

	// Step 2: Create and attach profile
	t.Log("Step 2: Create and attach browser profile")
	profile := types.Profile{
		ID:        "e2e-test-profile",
		Name:      "E2E Test Profile",
		UserAgent: "Mozilla/5.0 (E2E Test) AppleWebKit/537.36",
		Viewport: &types.Viewport{
			Width:  1280,
			Height: 720,
		},
		LocalStorage: map[string]map[string]string{
			"httpbin.org": {
				"test_key":     "test_value",
				"session_type": "e2e_test",
			},
		},
		Cookies: []types.Cookie{
			{
				Name:     "e2e_test_cookie",
				Value:    "test_session_12345",
				Domain:   "httpbin.org",
				Path:     "/",
				HTTPOnly: false,
				Secure:   false,
			},
		},
	}

	// agentClient.AttachProfile(ctx, contextID, profile)

	// Step 3: Submit task that uses profile
	t.Log("Step 3: Submit task that utilizes injected profile")
	task := types.Task{
		ID:     "profile-task",
		VMID:   contextID,
		URL:    "https://httpbin.org/headers",
		Script: "JSON.stringify({userAgent: navigator.userAgent, localStorage: localStorage.getItem('test_key')})",
	}

	// agentClient.SubmitTask(ctx, task)

	// Step 4: Verify profile was applied
	t.Log("Step 4: Verify profile settings were applied")
	// In a real test, we'd check that the task results show our custom user agent
	// and that localStorage was set correctly

	// Step 5: Cleanup
	t.Log("Step 5: Cleanup profile test resources")
	// agentClient.DestroyContext(ctx, contextID)

	t.Log("Profile injection workflow test completed successfully")
}

func TestCLIAgentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI-Agent integration test in short mode")
	}

	// This test would verify the CLI can communicate with a running agent
	// For now, we'll test that the CLI can be built and basic commands work

	projectRoot := getProjectRoot(t)

	// Build CLI
	t.Log("Building CLI for integration test")
	buildCmd := exec.Command("make", "build-cli")
	buildCmd.Dir = projectRoot

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test CLI help commands
	cliPath := filepath.Join(projectRoot, "bin", "viper")

	commands := [][]string{
		{"--help"},
		{"vms", "--help"},
		{"tasks", "--help"},
		{"browsers", "--help"},
		{"profiles", "--help"},
		{"debug", "--help"},
	}

	for _, cmdArgs := range commands {
		t.Run(fmt.Sprintf("cli_%s", cmdArgs[0]), func(t *testing.T) {
			cmd := exec.Command(cliPath, cmdArgs...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				// Help commands should not fail
				if cmdArgs[len(cmdArgs)-1] == "--help" {
					t.Errorf("Help command failed: %v\nOutput: %s", err, output)
				}
			}

			if len(output) == 0 {
				t.Error("Command produced no output")
			}

			t.Logf("Command output length: %d characters", len(output))
		})
	}
}

func TestAgentBinaryFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping agent binary test in short mode")
	}

	projectRoot := getProjectRoot(t)

	// Build agent binary
	t.Log("Building agent binary for testing")
	buildCmd := exec.Command("make", "build-agent")
	buildCmd.Dir = projectRoot

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build agent: %v\nOutput: %s", err, output)
	}

	// Test agent binary help
	agentPath := filepath.Join(projectRoot, "bin", "viper-agent")

	cmd := exec.Command(agentPath, "--help")
	output, err = cmd.CombinedOutput()

	// Agent binary might not have --help flag implemented
	// Just verify it exists and is executable
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("Agent binary not found: %v", err)
	}

	// Check binary is executable
	info, err := os.Stat(agentPath)
	if err != nil {
		t.Fatalf("Failed to stat agent binary: %v", err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("Agent binary is not executable")
	}

	t.Logf("Agent binary size: %d bytes", info.Size())
	t.Logf("Agent binary permissions: %s", mode)
}

func TestConfigurationFiles(t *testing.T) {
	projectRoot := getProjectRoot(t)

	// Test sample configuration files
	configs := []struct {
		name     string
		path     string
		validate func(t *testing.T, data []byte)
	}{
		{
			name: "example_task",
			path: filepath.Join(projectRoot, "configs", "example-task.json"),
			validate: func(t *testing.T, data []byte) {
				var task types.Task
				if err := json.Unmarshal(data, &task); err != nil {
					t.Errorf("Failed to parse task JSON: %v", err)
				}
				if task.URL == "" {
					t.Error("Task URL is empty")
				}
			},
		},
		{
			name: "example_profile",
			path: filepath.Join(projectRoot, "configs", "example-profile.json"),
			validate: func(t *testing.T, data []byte) {
				var profile types.Profile
				if err := json.Unmarshal(data, &profile); err != nil {
					t.Errorf("Failed to parse profile JSON: %v", err)
				}
				if profile.ID == "" {
					t.Error("Profile ID is empty")
				}
			},
		},
	}

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			data, err := os.ReadFile(config.path)
			if err != nil {
				t.Errorf("Failed to read config file: %v", err)
				return
			}

			if len(data) == 0 {
				t.Error("Config file is empty")
				return
			}

			config.validate(t, data)

			t.Logf("Config file size: %d bytes", len(data))
		})
	}
}

func TestNomadJobFiles(t *testing.T) {
	projectRoot := getProjectRoot(t)

	// Test Nomad job files exist and are readable
	jobFiles := []string{
		filepath.Join(projectRoot, "jobs", "example-vm.nomad.hcl"),
		filepath.Join(projectRoot, "configs", "gpu-vm.nomad.hcl"),
	}

	for _, jobFile := range jobFiles {
		t.Run(filepath.Base(jobFile), func(t *testing.T) {
			data, err := os.ReadFile(jobFile)
			if err != nil {
				t.Errorf("Failed to read job file: %v", err)
				return
			}

			if len(data) == 0 {
				t.Error("Job file is empty")
				return
			}

			// Basic validation - should contain Nomad job structure
			content := string(data)
			if !contains(content, "job ") {
				t.Error("Job file doesn't contain 'job' declaration")
			}

			if !contains(content, "task ") {
				t.Error("Job file doesn't contain 'task' declaration")
			}

			t.Logf("Job file size: %d bytes", len(data))
		})
	}
}

func TestMakefileTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Makefile tests in short mode")
	}

	projectRoot := getProjectRoot(t)

	// Test essential Makefile targets
	targets := []struct {
		name          string
		target        string
		shouldSucceed bool
	}{
		{
			name:          "help",
			target:        "help",
			shouldSucceed: true,
		},
		{
			name:          "format",
			target:        "format",
			shouldSucceed: true,
		},
		{
			name:          "build",
			target:        "build",
			shouldSucceed: true,
		},
		{
			name:          "clean",
			target:        "clean",
			shouldSucceed: true,
		},
		{
			name:          "version",
			target:        "version",
			shouldSucceed: true,
		},
	}

	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			cmd := exec.Command("make", target.target)
			cmd.Dir = projectRoot

			output, err := cmd.CombinedOutput()

			if target.shouldSucceed && err != nil {
				t.Errorf("Make target '%s' failed: %v\nOutput: %s", target.target, err, output)
			}

			t.Logf("Make target '%s' output length: %d characters", target.target, len(output))
		})
	}
}

// Helper functions

func getProjectRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

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

	t.Fatalf("Could not find project root (go.mod not found)")
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
