package nomad

import (
	"testing"

	"github.com/viper-org/viper/internal/types"
)

func TestBuildVMJob(t *testing.T) {
	client := &Client{}

	config := types.VMConfig{
		Name:     "test-vm",
		VMM:      "cloudhypervisor",
		Contexts: 2,
		Memory:   2048,
		CPUs:     2,
		Disk:     8192,
	}

	job := client.buildVMJob(config)

	expectedID := "viper-vm-test-vm"
	if *job.ID != expectedID {
		t.Errorf("Expected job ID %s, got %s", expectedID, *job.ID)
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
	if task.Name != "agent" {
		t.Errorf("Expected task name 'agent', got %s", task.Name)
	}

	if task.Driver != "exec" {
		t.Errorf("Expected driver 'exec', got %s", task.Driver)
	}

	expectedCPU := config.CPUs * 1000
	if *task.Resources.CPU != expectedCPU {
		t.Errorf("Expected CPU %d, got %d", expectedCPU, *task.Resources.CPU)
	}

	if *task.Resources.MemoryMB != config.Memory {
		t.Errorf("Expected Memory %d, got %d", config.Memory, *task.Resources.MemoryMB)
	}
}

func TestBuildVMJobWithGPU(t *testing.T) {
	client := &Client{}

	config := types.VMConfig{
		Name:     "gpu-vm",
		VMM:      "cloudhypervisor",
		Contexts: 2,
		GPU:      true,
		Memory:   8192,
		CPUs:     4,
		Disk:     16384,
	}

	job := client.buildVMJob(config)

	taskGroup := job.TaskGroups[0]
	task := taskGroup.Tasks[0]

	if len(task.Resources.Devices) != 1 {
		t.Errorf("Expected 1 GPU device, got %d", len(task.Resources.Devices))
	}

	device := task.Resources.Devices[0]
	if device.Name != "nvidia/gpu" {
		t.Errorf("Expected device name 'nvidia/gpu', got %s", device.Name)
	}

	if *device.Count != 1 {
		t.Errorf("Expected device count 1, got %d", *device.Count)
	}
}

func TestCreateVMJobGeneration(t *testing.T) {
	client := &Client{}

	config := types.VMConfig{
		Name:     "integration-test",
		VMM:      "cloudhypervisor",
		Contexts: 1,
		Memory:   1024,
		CPUs:     1,
		Disk:     4096,
		Labels: map[string]string{
			"env":     "test",
			"project": "viper",
		},
	}

	job := client.buildVMJob(config)

	// Verify job structure
	if job.TaskGroups == nil || len(job.TaskGroups) == 0 {
		t.Fatal("Job should have task groups")
	}

	taskGroup := job.TaskGroups[0]
	if taskGroup.Tasks == nil || len(taskGroup.Tasks) == 0 {
		t.Fatal("Task group should have tasks")
	}

	task := taskGroup.Tasks[0]
	if task.Config == nil {
		t.Fatal("Task should have config")
	}

	// Verify command configuration
	command, ok := task.Config["command"].(string)
	if !ok || command != "/usr/local/bin/viper-agent" {
		t.Errorf("Expected command '/usr/local/bin/viper-agent', got %v", command)
	}

	args, ok := task.Config["args"].([]string)
	if !ok {
		t.Fatal("Expected args to be []string")
	}

	expectedArgs := []string{
		"--listen=:8080",
		"--vm-name=integration-test",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, expected := range expectedArgs {
		if i >= len(args) || args[i] != expected {
			t.Errorf("Expected arg[%d] = %s, got %s", i, expected, args[i])
		}
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

// Benchmark tests for performance validation
func BenchmarkBuildVMJob(b *testing.B) {
	client := &Client{}
	config := types.VMConfig{
		Name:     "benchmark-vm",
		VMM:      "cloudhypervisor",
		Contexts: 2,
		Memory:   2048,
		CPUs:     2,
		Disk:     8192,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.buildVMJob(config)
	}
}

// Test edge cases and error conditions
func TestBuildVMJobEdgeCases(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name   string
		config types.VMConfig
	}{
		{
			name: "minimal config",
			config: types.VMConfig{
				Name: "minimal",
			},
		},
		{
			name: "zero values",
			config: types.VMConfig{
				Name:     "zero-vm",
				Contexts: 0,
				Memory:   0,
				CPUs:     0,
			},
		},
		{
			name: "large values",
			config: types.VMConfig{
				Name:     "large-vm",
				Contexts: 100,
				Memory:   32768,
				CPUs:     16,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := client.buildVMJob(tt.config)

			// Should not panic and should produce valid job
			if job == nil {
				t.Error("buildVMJob returned nil")
			}

			expectedID := "viper-vm-" + tt.config.Name
			if *job.ID != expectedID {
				t.Errorf("Expected job ID %s, got %s", expectedID, *job.ID)
			}
		})
	}
}