package nomad

import (
	"context"
	"testing"
)

func TestJobGeneratorBasic(t *testing.T) {
	imagePaths := ImagePaths{
		Kernel:    "/path/to/vmlinuz",
		Initramfs: "/path/to/initramfs.gz",
		DiskImage: "/path/to/image.qcow2",
	}

	generator := NewVMJobGenerator("dc1", imagePaths)

	opts := VMCreateOptions{
		Name:       "test-vm",
		Memory:     2048,
		CPU:        2000,
		ImagePaths: imagePaths,
	}

	job, err := generator.GenerateVMJob(opts)
	if err != nil {
		t.Fatalf("Failed to generate job: %v", err)
	}

	if *job.ID != "test-vm" {
		t.Errorf("Expected job ID 'test-vm', got %s", *job.ID)
	}

	if *job.Type != "service" {
		t.Errorf("Expected job type 'service', got %s", *job.Type)
	}

	if len(job.TaskGroups) != 1 {
		t.Errorf("Expected 1 task group, got %d", len(job.TaskGroups))
	}

	taskGroup := job.TaskGroups[0]
	if *taskGroup.Count != 1 {
		t.Errorf("Expected task group count 1, got %d", *taskGroup.Count)
	}

	if len(taskGroup.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(taskGroup.Tasks))
	}

	task := taskGroup.Tasks[0]
	if task.Driver != "nomad-driver-ch" {
		t.Errorf("Expected driver 'nomad-driver-ch', got %s", task.Driver)
	}

	if *task.Resources.CPU != 2000 {
		t.Errorf("Expected CPU 2000, got %d", *task.Resources.CPU)
	}

	if *task.Resources.MemoryMB != 2048 {
		t.Errorf("Expected Memory 2048, got %d", *task.Resources.MemoryMB)
	}
}

func TestJobGeneratorHCL(t *testing.T) {
	imagePaths := ImagePaths{
		Kernel:    "/path/to/vmlinuz",
		Initramfs: "/path/to/initramfs.gz",
		DiskImage: "/path/to/image.qcow2",
	}

	generator := NewVMJobGenerator("dc1", imagePaths)

	opts := VMCreateOptions{
		Name:       "hcl-test-vm",
		Memory:     1024,
		CPU:        1000,
		ImagePaths: imagePaths,
	}

	hcl, err := generator.GenerateJobHCL(opts)
	if err != nil {
		t.Fatalf("Failed to generate HCL: %v", err)
	}

	if hcl == "" {
		t.Error("Generated HCL is empty")
	}

	// Basic checks for HCL content
	if !contains(hcl, "job \"hcl-test-vm\"") {
		t.Error("HCL should contain job name")
	}

	if !contains(hcl, "driver = \"nomad-driver-ch\"") {
		t.Error("HCL should contain driver name (nomad-driver-ch)")
	}

	if !contains(hcl, "/path/to/vmlinuz") {
		t.Error("HCL should contain kernel path")
	}

	if !contains(hcl, "viperbr0") {
		t.Error("HCL should contain default bridge name")
	}
}

func TestListVMs(t *testing.T) {
	// This test requires a real Nomad client, so we skip if we can't create one
	client, err := NewClient()
	if err != nil {
		t.Skipf("Skipping test - Nomad client creation failed: %v", err)
		return
	}

	vms, err := client.ListVMs(context.Background())

	// Either we get VMs (empty slice if no VMs) or an error (if Nomad is not running)
	if err != nil {
		t.Logf("ListVMs failed as expected (Nomad not running): %v", err)
	} else {
		t.Logf("ListVMs succeeded, found %d VMs", len(vms))
	}
}

func TestDestroyVM(t *testing.T) {
	// This test requires a real Nomad client, so we skip if we can't create one
	client, err := NewClient()
	if err != nil {
		t.Skipf("Skipping test - Nomad client creation failed: %v", err)
		return
	}

	err = client.DestroyVM(context.Background(), "test-vm")

	// We expect an error since VM doesn't exist, but method should exist
	if err == nil {
		t.Log("DestroyVM succeeded (unexpected if VM doesn't exist)")
	}
}

func TestStringPtr(t *testing.T) {
	tests := []string{"test", "another-test", ""}

	for _, test := range tests {
		ptr := stringPtr(test)
		if ptr == nil {
			t.Errorf("stringPtr returned nil for %s", test)
		}
		if *ptr != test {
			t.Errorf("Expected %s, got %s", test, *ptr)
		}
	}
}

func TestIntPtr(t *testing.T) {
	tests := []int{0, 1, 100, -1}

	for _, test := range tests {
		ptr := intPtr(test)
		if ptr == nil {
			t.Errorf("intPtr returned nil for %d", test)
		}
		if *ptr != test {
			t.Errorf("Expected %d, got %d", test, *ptr)
		}
	}
}

func TestResolveImagePaths(t *testing.T) {
	basePath := "/test/images"
	paths := ResolveImagePaths(basePath)

	expected := ImagePaths{
		Kernel:    "/test/images/vmlinuz",
		Initramfs: "/test/images/viper-initramfs.gz",
		DiskImage: "/test/images/viper-headless.qcow2",
	}

	if paths.Kernel != expected.Kernel {
		t.Errorf("Expected kernel %s, got %s", expected.Kernel, paths.Kernel)
	}

	if paths.Initramfs != expected.Initramfs {
		t.Errorf("Expected initramfs %s, got %s", expected.Initramfs, paths.Initramfs)
	}

	if paths.DiskImage != expected.DiskImage {
		t.Errorf("Expected disk image %s, got %s", expected.DiskImage, paths.DiskImage)
	}
}

func TestBridgeConfiguration(t *testing.T) {
	imagePaths := ImagePaths{
		Kernel:    "/path/to/vmlinuz",
		Initramfs: "/path/to/initramfs.gz",
	}

	// Test with default configuration
	generator := NewVMJobGenerator("dc1", imagePaths)
	_, config, err := generator.DebugGenerateJob(VMCreateOptions{
		Name:       "test-vm",
		ImagePaths: imagePaths,
	})
	if err != nil {
		t.Fatalf("Failed to generate job: %v", err)
	}

	// Check that bridge is set to default
	if config == nil {
		t.Fatal("Task config is nil")
	}

	taskConfig, ok := config["network_interface"]
	if !ok {
		t.Fatal("network_interface not found in config")
	}

	networkInterface, ok := taskConfig.(map[string]interface{})
	if !ok {
		t.Fatal("network_interface is not a map")
	}

	bridgeConfig, ok := networkInterface["bridge"]
	if !ok {
		t.Fatal("bridge config not found")
	}

	bridgeMap, ok := bridgeConfig.(map[string]interface{})
	if !ok {
		t.Fatal("bridge config is not a map")
	}

	bridgeName, ok := bridgeMap["name"]
	if !ok {
		t.Fatal("bridge name not found")
	}

	if bridgeName != "viperbr0" {
		t.Errorf("Expected bridge name 'viperbr0', got %s", bridgeName)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
