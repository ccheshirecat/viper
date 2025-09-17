package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskSerialization(t *testing.T) {
	task := Task{
		ID:      "test-task",
		VMID:    "test-vm",
		URL:     "https://example.com",
		Script:  "console.log('test');",
		Timeout: 60 * time.Second,
		Status:  TaskStatusPending,
		Created: time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if decoded.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, decoded.ID)
	}
	if decoded.VMID != task.VMID {
		t.Errorf("Expected VMID %s, got %s", task.VMID, decoded.VMID)
	}
	if decoded.Status != task.Status {
		t.Errorf("Expected Status %s, got %s", task.Status, decoded.Status)
	}
}

func TestProfileSerialization(t *testing.T) {
	profile := Profile{
		ID:        "test-profile",
		Name:      "Test Profile",
		UserAgent: "Mozilla/5.0 Test Browser",
		Viewport: &Viewport{
			Width:  1920,
			Height: 1080,
		},
		LocalStorage: map[string]map[string]string{
			"example.com": {
				"theme": "dark",
				"lang":  "en",
			},
		},
		Cookies: []Cookie{
			{
				Name:     "session",
				Value:    "abc123",
				Domain:   "example.com",
				Path:     "/",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "Lax",
			},
		},
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}

	var decoded Profile
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal profile: %v", err)
	}

	if decoded.ID != profile.ID {
		t.Errorf("Expected ID %s, got %s", profile.ID, decoded.ID)
	}
	if decoded.UserAgent != profile.UserAgent {
		t.Errorf("Expected UserAgent %s, got %s", profile.UserAgent, decoded.UserAgent)
	}
	if decoded.Viewport.Width != profile.Viewport.Width {
		t.Errorf("Expected Width %d, got %d", profile.Viewport.Width, decoded.Viewport.Width)
	}
	if len(decoded.Cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(decoded.Cookies))
	}
	if decoded.Cookies[0].Name != "session" {
		t.Errorf("Expected cookie name 'session', got %s", decoded.Cookies[0].Name)
	}
}

func TestVMConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  VMConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: VMConfig{
				Name:     "test-vm",
				VMM:      "cloudhypervisor",
				Contexts: 1,
				Memory:   2048,
				CPUs:     2,
				Disk:     8192,
			},
			wantErr: false,
		},
		{
			name: "valid GPU config",
			config: VMConfig{
				Name:     "gpu-vm",
				VMM:      "cloudhypervisor",
				Contexts: 2,
				GPU:      true,
				Memory:   8192,
				CPUs:     4,
				Disk:     16384,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			var decoded VMConfig
			err = json.Unmarshal(data, &decoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && decoded.Name != tt.config.Name {
				t.Errorf("Expected name %s, got %s", tt.config.Name, decoded.Name)
			}
		})
	}
}

func TestTaskStatusTransitions(t *testing.T) {
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending: {TaskStatusRunning, TaskStatusFailed},
		TaskStatusRunning: {TaskStatusCompleted, TaskStatusFailed, TaskStatusTimeout},
	}

	for from, validTos := range validTransitions {
		for _, to := range validTos {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				task := Task{
					ID:     "test",
					Status: from,
				}

				task.Status = to

				if task.Status != to {
					t.Errorf("Failed to transition from %s to %s", from, to)
				}
			})
		}
	}
}

func TestAgentHealthSerialization(t *testing.T) {
	health := AgentHealth{
		Status:    "healthy",
		Version:   "0.1.0",
		Uptime:    5 * time.Minute,
		Contexts:  2,
		Tasks:     1,
		Memory:    1024 * 1024 * 100, // 100MB
		LastCheck: time.Now(),
		Details: map[string]string{
			"vm_name": "test-vm",
		},
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal health: %v", err)
	}

	var decoded AgentHealth
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal health: %v", err)
	}

	if decoded.Status != health.Status {
		t.Errorf("Expected status %s, got %s", health.Status, decoded.Status)
	}
	if decoded.Contexts != health.Contexts {
		t.Errorf("Expected contexts %d, got %d", health.Contexts, decoded.Contexts)
	}
	if decoded.Details["vm_name"] != "test-vm" {
		t.Errorf("Expected vm_name 'test-vm', got %s", decoded.Details["vm_name"])
	}
}