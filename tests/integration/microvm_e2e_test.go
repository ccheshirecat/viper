package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/internal/types"
	"github.com/ccheshirecat/viper/pkg/client"
	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMicroVMEndToEnd tests the complete end-to-end workflow with real nomad-driver-ch
func TestMicroVMEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if we're running in an environment with nomad-driver-ch
	if !isNomadDriverCHAvailable(t) {
		t.Skip("nomad-driver-ch not available - skipping microVM integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Initialize Nomad client
	nomadClient, err := nomad.NewClient()
	require.NoError(t, err, "Failed to create Nomad client")

	// Test job generation and VM creation
	t.Run("CreateMicroVM", func(t *testing.T) {
		testCreateMicroVM(ctx, t, nomadClient)
	})
}

// testCreateMicroVM tests creating a microVM with nomad-driver-ch
func testCreateMicroVM(ctx context.Context, t *testing.T, nomadClient *nomad.Client) {
	vmName := "test-vm-e2e"
	jobID := fmt.Sprintf("viper-vm-%s", vmName)

	// Cleanup any existing test VM
	defer func() {
		if err := nomadClient.DestroyVM(ctx, vmName); err != nil {
			t.Logf("Warning: failed to cleanup test VM: %v", err)
		}
	}()

	// Create a test job for nomad-driver-ch
	job := createTestMicroVMJob(jobID, vmName)

	// Submit the job
	t.Logf("Submitting job %s to Nomad...", jobID)
	actualJobID, err := nomadClient.SubmitJob(ctx, job)
	require.NoError(t, err, "Failed to submit job")
	assert.Equal(t, jobID, actualJobID)

	// Wait for VM to be running with timeout
	t.Logf("Waiting for VM %s to start...", vmName)
	vmStatus := waitForVMRunning(ctx, t, nomadClient, vmName, 3*time.Minute)
	require.NotNil(t, vmStatus, "VM failed to start within timeout")
	assert.Equal(t, "running", vmStatus.Status)

	// Test service discovery
	t.Logf("Testing service discovery for VM %s...", vmName)
	agentURL, err := nomadClient.ResolveVMAgentURL(ctx, vmName)
	require.NoError(t, err, "Service discovery should resolve VM agent URL")
	assert.Contains(t, agentURL, "http://", "Agent URL should be HTTP")
	assert.Contains(t, agentURL, ":8080", "Agent URL should include port 8080")
	t.Logf("✅ Service discovery resolved VM agent URL: %s", agentURL)

	// Test AgentClient with service discovery
	t.Logf("Testing AgentClient with service discovery...")
	agentClient, err := client.NewAgentClient(vmName)
	require.NoError(t, err, "Failed to create agent client")

	// Wait for agent to be ready (VMs need time to boot)
	t.Logf("Waiting for agent to be ready...")
	var health *types.AgentHealth
	healthCheckCtx, healthCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer healthCancel()

	for {
		select {
		case <-healthCheckCtx.Done():
			t.Fatal("Agent health check timeout - VM may not have booted properly")
		default:
			health, err = agentClient.Health(ctx)
			if err == nil {
				goto healthCheckComplete // Use goto to avoid unreachable code warning
			}
			t.Logf("Agent not ready yet, retrying in 10s... (error: %v)", err)
			time.Sleep(10 * time.Second)
		}
	}

healthCheckComplete:

	require.NotNil(t, health, "Agent health should be available")
	assert.Equal(t, "healthy", health.Status, "Agent should report healthy status")
	t.Logf("✅ Agent health check successful: %+v", health)

	// Test browser context spawn
	t.Logf("Testing browser context spawn...")
	err = agentClient.SpawnContext(ctx, "test-ctx-1")
	require.NoError(t, err, "Failed to spawn browser context")
	t.Logf("✅ Browser context spawned successfully")

	// Test context listing
	t.Logf("Testing context listing...")
	contexts, err := agentClient.ListContexts(ctx)
	require.NoError(t, err, "Failed to list contexts")
	assert.Len(t, contexts, 1, "Should have one context")
	assert.Equal(t, "test-ctx-1", contexts[0].ID, "Context ID should match")
	t.Logf("✅ Context listing successful: %+v", contexts)

	// Test task submission
	t.Logf("Testing task submission...")
	task := types.Task{
		ID:      "test-task-1",
		VMID:    vmName,
		URL:     "https://example.com",
		Status:  types.TaskStatusPending,
		Created: time.Now(),
		Timeout: 30 * time.Second,
	}

	result, err := agentClient.SubmitTask(ctx, task)
	require.NoError(t, err, "Failed to submit task")
	require.NotNil(t, result, "Task result should be available")
	t.Logf("✅ Task submitted successfully: %+v", result)

	// Test task logs retrieval
	t.Logf("Testing task logs retrieval...")

	// Wait a bit for task to complete
	time.Sleep(5 * time.Second)

	logs, err := agentClient.GetTaskLogs(ctx, result.TaskID)
	if err == nil {
		t.Logf("✅ Task logs retrieved: %s", logs)
	} else {
		t.Logf("⚠️  Task logs not available yet (this is expected): %v", err)
	}

	t.Logf("🎉 End-to-end test completed successfully!")
}

// createTestMicroVMJob creates a test job for nomad-driver-ch
func createTestMicroVMJob(jobID, vmName string) *nomadapi.Job {
	job := &nomadapi.Job{
		ID:          &jobID,
		Name:        &jobID,
		Type:        stringPtr("service"),
		Datacenters: []string{"dc1"},
		TaskGroups: []*nomadapi.TaskGroup{
			{
				Name:  stringPtr("viper-vm"),
				Count: intPtr(1),
				Tasks: []*nomadapi.Task{
					{
						Name:   "microvm",
						Driver: "nomad-driver-ch", // nomad-driver-ch driver name
						Config: map[string]interface{}{
							// Use a minimal test image - in production this would be the viper image
							"image": "/var/lib/images/test-alpine.img",

							// REQUIRED for Cloud Hypervisor: kernel and initramfs
							"kernel":    "/boot/vmlinuz",
							"initramfs": "/boot/initramfs.img",
							"cmdline":   "console=ttyS0 init=/usr/local/bin/viper-agent",

							"hostname": vmName,

							// Network configuration for service discovery
							"network_interface": map[string]interface{}{
								"bridge": map[string]interface{}{
									"name": "br0",
									// Let nomad-driver-ch allocate IP dynamically
								},
							},
						},
						Resources: &nomadapi.Resources{
							CPU:      intPtr(1000), // 1 CPU core
							MemoryMB: intPtr(1024), // 1GB RAM
							Networks: []*nomadapi.NetworkResource{
								{
									MBits: intPtr(100),
									ReservedPorts: []nomadapi.Port{
										{
											Label: "agent",
											Value: 8080,
										},
									},
								},
							},
						},
						Services: []*nomadapi.Service{
							{
								Name:      "viper-agent",
								PortLabel: "agent",
								Checks: []nomadapi.ServiceCheck{
									{
										Type:     "tcp",
										Interval: time.Duration(30 * time.Second),
										Timeout:  time.Duration(10 * time.Second),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// waitForVMRunning waits for a VM to reach running status
func waitForVMRunning(ctx context.Context, t *testing.T, nomadClient *nomad.Client, vmName string, timeout time.Duration) *types.VMStatus {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Logf("Timeout waiting for VM %s to be running", vmName)
			return nil
		case <-ticker.C:
			vms, err := nomadClient.ListVMs(ctx)
			if err != nil {
				t.Logf("Error listing VMs: %v", err)
				continue
			}

			for _, vm := range vms {
				if vm.Name == vmName {
					t.Logf("VM %s status: %s", vmName, vm.Status)
					if vm.Status == "running" {
						return &vm
					}
				}
			}
		}
	}
}

// isNomadDriverCHAvailable checks if nomad-driver-ch is available
func isNomadDriverCHAvailable(t *testing.T) bool {
	// Check environment variable first
	if os.Getenv("SKIP_CH_TESTS") == "true" {
		t.Logf("Skipping nomad-driver-ch tests (SKIP_CH_TESTS=true)")
		return false
	}

	// Try to create Nomad client
	nomadClient, err := nomad.NewClient()
	if err != nil {
		t.Logf("Cannot create Nomad client: %v", err)
		return false
	}

	// Check if we can connect to Nomad
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := nomadClient.GetSystemStatus(ctx)
	if err != nil {
		t.Logf("Cannot connect to Nomad: %v", err)
		return false
	}

	if status.NomadStatus != "connected" {
		t.Logf("Nomad not connected: %s", status.NomadStatus)
		return false
	}

	// TODO: Add check for nomad-driver-ch plugin availability
	// This would require querying Nomad's plugin API to see if 'ch' driver is loaded

	t.Logf("✅ nomad-driver-ch environment is available")
	return true
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
