package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
)

func TestAgentClientHealth(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		health := types.AgentHealth{
			Status:    "healthy",
			Version:   "0.1.0",
			Uptime:    5 * time.Minute,
			Contexts:  2,
			Tasks:     1,
			Memory:    1024 * 1024,
			LastCheck: time.Now(),
			Details: map[string]string{
				"vm_name": "test-vm",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}))
	defer server.Close()

	// Create client pointing to mock server
	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", health.Status)
	}

	if health.Version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got %s", health.Version)
	}

	if health.Details["vm_name"] != "test-vm" {
		t.Errorf("Expected vm_name 'test-vm', got %s", health.Details["vm_name"])
	}
}

func TestAgentClientSpawnContext(t *testing.T) {
	contextID := "test-context"
	called := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		expectedPath := "/spawn/" + contextID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "spawned",
			"id":     contextID,
		})
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	err := client.SpawnContext(ctx, contextID)
	if err != nil {
		t.Fatalf("SpawnContext failed: %v", err)
	}

	if !called {
		t.Error("Server handler was not called")
	}
}

func TestAgentClientSubmitTask(t *testing.T) {
	task := types.Task{
		ID:     "test-task",
		VMID:   "test-vm",
		URL:    "https://example.com",
		Script: "console.log('test');",
	}

	var receivedTask types.Task
	called := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		if r.URL.Path != "/task" {
			t.Errorf("Expected path /task, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Decode received task
		err := json.NewDecoder(r.Body).Decode(&receivedTask)
		if err != nil {
			t.Errorf("Failed to decode task: %v", err)
		}

		// Return task result
		result := types.TaskResult{
			TaskID: task.ID,
			Status: types.TaskStatusPending,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	result, err := client.SubmitTask(ctx, task)
	if err != nil {
		t.Fatalf("SubmitTask failed: %v", err)
	}

	if !called {
		t.Error("Server handler was not called")
	}

	if receivedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, receivedTask.ID)
	}

	if result.TaskID != task.ID {
		t.Errorf("Expected result task ID %s, got %s", task.ID, result.TaskID)
	}
}

func TestAgentClientAttachProfile(t *testing.T) {
	contextID := "test-context"
	profile := types.Profile{
		ID:        "test-profile",
		Name:      "Test Profile",
		UserAgent: "Mozilla/5.0 Test",
		Viewport: &types.Viewport{
			Width:  1920,
			Height: 1080,
		},
		LocalStorage: map[string]map[string]string{
			"example.com": {
				"theme": "dark",
			},
		},
	}

	var receivedProfile types.Profile
	called := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		expectedPath := "/profile/" + contextID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		err := json.NewDecoder(r.Body).Decode(&receivedProfile)
		if err != nil {
			t.Errorf("Failed to decode profile: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "profile attached",
			"id":     profile.ID,
		})
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	err := client.AttachProfile(ctx, contextID, profile)
	if err != nil {
		t.Fatalf("AttachProfile failed: %v", err)
	}

	if !called {
		t.Error("Server handler was not called")
	}

	if receivedProfile.ID != profile.ID {
		t.Errorf("Expected profile ID %s, got %s", profile.ID, receivedProfile.ID)
	}

	if receivedProfile.UserAgent != profile.UserAgent {
		t.Errorf("Expected UserAgent %s, got %s", profile.UserAgent, receivedProfile.UserAgent)
	}
}

func TestAgentClientGetTaskLogs(t *testing.T) {
	taskID := "test-task"
	expectedLogs := "Task started\nNavigating to URL\nTask completed\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/logs/" + taskID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(expectedLogs))
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	logs, err := client.GetTaskLogs(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskLogs failed: %v", err)
	}

	if logs != expectedLogs {
		t.Errorf("Expected logs %q, got %q", expectedLogs, logs)
	}
}

func TestAgentClientGetTaskScreenshots(t *testing.T) {
	taskID := "test-task"
	expectedScreenshots := []string{"1.png", "2.png", "final.png"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/screenshots/" + taskID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedScreenshots)
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()
	screenshots, err := client.GetTaskScreenshots(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskScreenshots failed: %v", err)
	}

	if len(screenshots) != len(expectedScreenshots) {
		t.Errorf("Expected %d screenshots, got %d", len(expectedScreenshots), len(screenshots))
	}

	for i, expected := range expectedScreenshots {
		if i >= len(screenshots) || screenshots[i] != expected {
			t.Errorf("Expected screenshot[%d] = %s, got %s", i, expected, screenshots[i])
		}
	}
}

func TestAgentClientErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedError string
	}{
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			expectedError: "health check failed with status: 500",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			expectedError: "health check failed with status: 404",
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			expectedError: "health check failed with status: 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &AgentClient{
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
				},
			}

			ctx := context.Background()
			_, err := client.Health(ctx)
			if err == nil {
				t.Error("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestAgentClientTimeout(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 50 * time.Millisecond, // Shorter than server delay
		},
	}

	ctx := context.Background()
	_, err := client.Health(ctx)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

func BenchmarkAgentClientHealth(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		health := types.AgentHealth{
			Status:  "healthy",
			Version: "0.1.0",
		}
		json.NewEncoder(w).Encode(health)
	}))
	defer server.Close()

	client := &AgentClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Health(ctx)
		if err != nil {
			b.Fatalf("Health check failed: %v", err)
		}
	}
}