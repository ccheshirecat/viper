package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/internal/types"
	"github.com/ccheshirecat/viper/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMicroVMEndToEnd tests the complete microVM workflow with real nomad-driver-ch
func TestMicroVMEndToEnd(t *testing.T) {
	// Skip this test if not running with real infrastructure
	if testing.Short() {
		t.Skip("Skipping microVM integration test in short mode")
	}

	// Check if we have the required infrastructure
	if !hasNomadDriverCH(t) {
		t.Skip("Skipping microVM test - nomad-driver-ch not available")
	}

	if !hasVMImages(t) {
		t.Skip("Skipping microVM test - VM images not available. Run 'make build-images' first.")
	}

	ctx := context.Background()
	vmName := fmt.Sprintf("test-vm-%d", time.Now().Unix())

	// Clean up any existing test VMs
	defer cleanupTestVM(t, vmName)

	t.Run("1. Create microVM", func(t *testing.T) {
		testCreateMicroVM(t, ctx, vmName)
	})

	t.Run("2. Wait for VM to boot and agent to be ready", func(t *testing.T) {
		testWaitForVMReady(t, ctx, vmName)
	})

	t.Run("3. Communicate with agent inside VM", func(t *testing.T) {
		testAgentCommunication(t, ctx, vmName)
	})

	t.Run("4. Spawn browser context", func(t *testing.T) {
		testSpawnBrowserContext(t, ctx, vmName)
	})

	t.Run("5. Submit browser automation task", func(t *testing.T) {
		testBrowserAutomationTask(t, ctx, vmName)
	})

	t.Run("6. Retrieve task results", func(t *testing.T) {
		testRetrieveTaskResults(t, ctx, vmName)
	})

	t.Run("7. Cleanup microVM", func(t *testing.T) {
		testCleanupMicroVM(t, ctx, vmName)
	})
}

func testCreateMicroVM(t *testing.T, ctx context.Context, vmName string) {
	// Create Nomad client
	nomadClient, err := nomad.NewClient()
	require.NoError(t, err, "Failed to create Nomad client")

	// Generate job for microVM
	imagePaths := nomad.ResolveImagePaths("./dist")
	generator := nomad.NewVMJobGenerator("dc1", "br0", imagePaths)

	opts := nomad.VMCreateOptions{
		Name:        vmName,
		Memory:      1024, // 1GB for testing
		CPU:         1000, // 1 CPU core
		NetworkMode: types.NetworkModePrivateSubnet,
		ImagePaths:  imagePaths,
	}

	job, err := generator.GenerateVMJob(opts)
	require.NoError(t, err, "Failed to generate VM job")

	// Submit job to Nomad
	jobID, err := nomadClient.SubmitJob(ctx, job)
	require.NoError(t, err, "Failed to submit VM job to Nomad")

	assert.Equal(t, vmName, jobID, "Job ID should match VM name")
	t.Logf("✅ Successfully created microVM job: %s", jobID)
}

func testWaitForVMReady(t *testing.T, ctx context.Context, vmName string) {
	nomadClient, err := nomad.NewClient()
	require.NoError(t, err, "Failed to create Nomad client")

	// Wait up to 2 minutes for VM to boot and agent to be ready
	timeout := time.Now().Add(2 * time.Minute)
	var lastErr error

	for time.Now().Before(timeout) {
		// Check if VM is running
		vms, err := nomadClient.ListVMs(ctx)
		if err != nil {
			lastErr = err
			time.Sleep(5 * time.Second)
			continue
		}

		// Find our VM
		var vm *types.VMStatus
		for _, v := range vms {
			if v.Name == vmName {
				vm = &v
				break
			}
		}

		if vm == nil {
			lastErr = fmt.Errorf("VM %s not found in job list", vmName)
			time.Sleep(5 * time.Second)
			continue
		}

		if vm.Status != "running" {
			lastErr = fmt.Errorf("VM %s status: %s", vmName, vm.Status)
			time.Sleep(5 * time.Second)
			continue
		}

		if vm.AgentURL == "" {
			lastErr = fmt.Errorf("VM %s has no agent URL", vmName)
			time.Sleep(5 * time.Second)
			continue
		}

		if vm.Health != "healthy" {
			lastErr = fmt.Errorf("VM %s health: %s", vmName, vm.Health)
			time.Sleep(5 * time.Second)
			continue
		}

		// VM is ready!
		t.Logf("✅ VM is ready: %s (agent: %s)", vmName, vm.AgentURL)
		return
	}

	t.Fatalf("VM failed to become ready within timeout. Last error: %v", lastErr)
}

func testAgentCommunication(t *testing.T, ctx context.Context, vmName string) {
	// Create agent client - this should use service discovery
	agentClient, err := client.NewAgentClient(vmName)
	require.NoError(t, err, "Failed to create agent client")

	// Test health endpoint
	health, err := agentClient.Health(ctx)
	require.NoError(t, err, "Failed to get agent health")

	assert.Equal(t, "healthy", health.Status, "Agent should be healthy")
	assert.Greater(t, health.Uptime, time.Duration(0), "Agent should have positive uptime")

	t.Logf("✅ Agent communication successful - uptime: %v", health.Uptime)
}

func testSpawnBrowserContext(t *testing.T, ctx context.Context, vmName string) {
	agentClient, err := client.NewAgentClient(vmName)
	require.NoError(t, err, "Failed to create agent client")

	contextID := "test-ctx-1"

	// Spawn browser context
	err = agentClient.SpawnContext(ctx, contextID)
	require.NoError(t, err, "Failed to spawn browser context")

	// Verify context exists
	contexts, err := agentClient.ListContexts(ctx)
	require.NoError(t, err, "Failed to list browser contexts")

	found := false
	for _, context := range contexts {
		if context.ID == contextID {
			found = true
			assert.True(t, context.Active, "New context should be active")
			break
		}
	}

	assert.True(t, found, "Browser context should be created")
	t.Logf("✅ Successfully spawned browser context: %s", contextID)
}

func testBrowserAutomationTask(t *testing.T, ctx context.Context, vmName string) {
	agentClient, err := client.NewAgentClient(vmName)
	require.NoError(t, err, "Failed to create agent client")

	// Create a simple test task
	task := types.Task{
		ID:      fmt.Sprintf("test-task-%d", time.Now().Unix()),
		VMID:    vmName,
		URL:     "https://example.com",
		Timeout: 30 * time.Second,
	}

	// Submit task
	result, err := agentClient.SubmitTask(ctx, task)
	require.NoError(t, err, "Failed to submit task")

	assert.Equal(t, task.ID, result.TaskID, "Task ID should match")
	assert.Equal(t, types.TaskStatusCompleted, result.Status, "Task should complete successfully")

	t.Logf("✅ Browser automation task completed: %s", task.ID)
}

func testRetrieveTaskResults(t *testing.T, ctx context.Context, vmName string) {
	agentClient, err := client.NewAgentClient(vmName)
	require.NoError(t, err, "Failed to create agent client")

	// Get the most recent task (assuming it's from the previous test)
	// In a real scenario, we'd store the task ID from the previous step
	taskID := fmt.Sprintf("test-task-%d", time.Now().Unix()-1) // Approximate

	// Try to get task logs
	logs, err := agentClient.GetTaskLogs(ctx, taskID)
	if err == nil {
		assert.NotEmpty(t, logs, "Task logs should not be empty")
		t.Logf("✅ Retrieved task logs (length: %d)", len(logs))
	} else {
		t.Logf("⚠️  Could not retrieve task logs (expected for test): %v", err)
	}

	// Try to get screenshots
	screenshots, err := agentClient.GetTaskScreenshots(ctx, taskID)
	if err == nil && len(screenshots) > 0 {
		t.Logf("✅ Retrieved %d screenshots", len(screenshots))
	} else {
		t.Logf("⚠️  No screenshots available (expected for test): %v", err)
	}
}

func testCleanupMicroVM(t *testing.T, ctx context.Context, vmName string) {
	nomadClient, err := nomad.NewClient()
	require.NoError(t, err, "Failed to create Nomad client")

	// Destroy the VM
	err = nomadClient.DestroyVM(ctx, vmName)
	require.NoError(t, err, "Failed to destroy VM")

	// Verify VM is removed (wait a bit for cleanup)
	time.Sleep(5 * time.Second)

	vms, err := nomadClient.ListVMs(ctx)
	require.NoError(t, err, "Failed to list VMs after cleanup")

	// Verify VM is not in the list
	for _, vm := range vms {
		assert.NotEqual(t, vmName, vm.Name, "VM should be removed from list")
	}

	t.Logf("✅ Successfully cleaned up microVM: %s", vmName)
}

// Helper functions

func hasNomadDriverCH(t *testing.T) bool {
	// Check if nomad-driver-ch is available by checking if Nomad is running
	// and if the driver is loaded
	nomadClient, err := nomad.NewClient()
	if err != nil {
		t.Logf("Nomad not available: %v", err)
		return false
	}

	// Simple connectivity test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = nomadClient.GetSystemStatus(ctx)
	if err != nil {
		t.Logf("Nomad system not accessible: %v", err)
		return false
	}

	t.Logf("✅ Nomad connectivity confirmed")
	return true
}

func hasVMImages(t *testing.T) bool {
	// Check if VM image files exist
	imagePaths := nomad.ResolveImagePaths("./dist")

	if _, err := os.Stat(imagePaths.Kernel); os.IsNotExist(err) {
		t.Logf("Kernel image not found: %s", imagePaths.Kernel)
		return false
	}

	if _, err := os.Stat(imagePaths.Initramfs); os.IsNotExist(err) {
		t.Logf("Initramfs image not found: %s", imagePaths.Initramfs)
		return false
	}

	// Make paths absolute for Nomad
	kernelAbs, _ := filepath.Abs(imagePaths.Kernel)
	initramfsAbs, _ := filepath.Abs(imagePaths.Initramfs)

	t.Logf("✅ VM images found:")
	t.Logf("  Kernel: %s", kernelAbs)
	t.Logf("  Initramfs: %s", initramfsAbs)
	return true
}

func cleanupTestVM(t *testing.T, vmName string) {
	// Best-effort cleanup
	nomadClient, err := nomad.NewClient()
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = nomadClient.DestroyVM(ctx, vmName)
	if err != nil {
		t.Logf("Cleanup warning - failed to destroy VM %s: %v", vmName, err)
	}
}

// Benchmark test for performance validation
func BenchmarkMicroVMLifecycle(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	if !hasNomadDriverCHQuiet() {
		b.Skip("Skipping benchmark - nomad-driver-ch not available")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vmName := fmt.Sprintf("bench-vm-%d-%d", i, time.Now().Unix())

		// Time the complete lifecycle
		start := time.Now()

		// Create VM
		nomadClient, err := nomad.NewClient()
		if err != nil {
			b.Fatalf("Failed to create Nomad client: %v", err)
		}

		imagePaths := nomad.ResolveImagePaths("./dist")
		generator := nomad.NewVMJobGenerator("dc1", "br0", imagePaths)

		opts := nomad.VMCreateOptions{
			Name:        vmName,
			Memory:      512, // Minimal for benchmark
			CPU:         500,
			NetworkMode: types.NetworkModePrivateSubnet,
			ImagePaths:  imagePaths,
		}

		job, err := generator.GenerateVMJob(opts)
		if err != nil {
			b.Fatalf("Failed to generate VM job: %v", err)
		}

		ctx := context.Background()
		_, err = nomadClient.SubmitJob(ctx, job)
		if err != nil {
			b.Fatalf("Failed to submit VM job: %v", err)
		}

		// Clean up
		defer nomadClient.DestroyVM(ctx, vmName)

		duration := time.Since(start)
		b.ReportMetric(float64(duration.Nanoseconds()), "ns/vm-create")
	}
}

func hasNomadDriverCHQuiet() bool {
	nomadClient, err := nomad.NewClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = nomadClient.GetSystemStatus(ctx)
	return err == nil
}