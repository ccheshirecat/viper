package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
)

func TestServerHealth(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var health types.AgentHealth
	err = json.Unmarshal(w.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
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

func TestServerSpawnContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	contextID := "test-context"
	req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "spawned" {
		t.Errorf("Expected status 'spawned', got %s", response["status"])
	}

	if response["id"] != contextID {
		t.Errorf("Expected id '%s', got %s", contextID, response["id"])
	}

	// Verify context was actually created
	server.mu.RLock()
	_, exists := server.contexts[contextID]
	server.mu.RUnlock()

	if !exists {
		t.Error("Context was not actually created")
	}
}

func TestServerListContexts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Spawn a context first
	contextID := "test-context"
	req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to spawn context")
	}

	// Now list contexts
	req = httptest.NewRequest("GET", "/contexts", nil)
	w = httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var contexts []types.BrowserContext
	err = json.Unmarshal(w.Body.Bytes(), &contexts)
	if err != nil {
		t.Fatalf("Failed to unmarshal contexts: %v", err)
	}

	if len(contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(contexts))
	}

	if contexts[0].ID != contextID {
		t.Errorf("Expected context ID '%s', got %s", contextID, contexts[0].ID)
	}

	if contexts[0].VMID != "test-vm" {
		t.Errorf("Expected VMID 'test-vm', got %s", contexts[0].VMID)
	}
}

func TestServerDestroyContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Spawn a context first
	contextID := "test-context"
	req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	// Now destroy it
	req = httptest.NewRequest("DELETE", "/contexts/"+contextID, nil)
	w = httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify context was actually destroyed
	server.mu.RLock()
	_, exists := server.contexts[contextID]
	server.mu.RUnlock()

	if exists {
		t.Error("Context was not actually destroyed")
	}
}

func TestServerAttachProfile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Spawn a context first
	contextID := "test-context"
	req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	// Create profile
	profile := types.Profile{
		ID:        "test-profile",
		Name:      "Test Profile",
		UserAgent: "Mozilla/5.0 Test Browser",
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

	profileData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}

	// Attach profile
	req = httptest.NewRequest("POST", "/profile/"+contextID, bytes.NewReader(profileData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify profile was attached
	server.mu.RLock()
	ctx, exists := server.contexts[contextID]
	server.mu.RUnlock()

	if !exists {
		t.Fatal("Context not found")
	}

	if ctx.Profile == nil {
		t.Error("Profile was not attached")
	}

	if ctx.Profile.ID != profile.ID {
		t.Errorf("Expected profile ID %s, got %s", profile.ID, ctx.Profile.ID)
	}
}

func TestServerSubmitTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Spawn a context first
	contextID := "test-vm" // Use vm name as context ID for this test
	req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	// Create task
	task := types.Task{
		ID:     "test-task",
		VMID:   "test-vm",
		URL:    "https://example.com",
		Script: "console.log('test');",
	}

	taskData, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	// Submit task
	req = httptest.NewRequest("POST", "/task", bytes.NewReader(taskData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result types.TaskResult
	err = json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result.TaskID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, result.TaskID)
	}

	// Verify task was stored
	server.mu.RLock()
	storedTask, exists := server.tasks[task.ID]
	server.mu.RUnlock()

	if !exists {
		t.Error("Task was not stored")
	}

	if storedTask.ID != task.ID {
		t.Errorf("Expected stored task ID %s, got %s", task.ID, storedTask.ID)
	}
}

func TestServerGetTaskStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Add a task directly to the server's task map
	task := &types.Task{
		ID:      "test-task",
		VMID:    "test-vm",
		Status:  types.TaskStatusCompleted,
		Created: time.Now(),
	}

	server.mu.Lock()
	server.tasks[task.ID] = task
	server.mu.Unlock()

	// Get task status
	req := httptest.NewRequest("GET", "/task/"+task.ID, nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var retrievedTask types.Task
	err = json.Unmarshal(w.Body.Bytes(), &retrievedTask)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrievedTask.ID)
	}

	if retrievedTask.Status != task.Status {
		t.Errorf("Expected status %s, got %s", task.Status, retrievedTask.Status)
	}
}

func TestServerGetTaskLogs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create task log file
	taskID := "test-task"
	taskDir := filepath.Join(tmpDir, "test-vm", taskID)
	err = os.MkdirAll(taskDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create task dir: %v", err)
	}

	logContent := "Task started\nNavigating to URL\nTask completed\n"
	logPath := filepath.Join(taskDir, "stdout.log")
	err = os.WriteFile(logPath, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	// Get task logs
	req := httptest.NewRequest("GET", "/logs/"+taskID, nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	retrievedLogs := w.Body.String()
	if retrievedLogs != logContent {
		t.Errorf("Expected logs %q, got %q", logContent, retrievedLogs)
	}
}

func TestServerGetTaskScreenshots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create task screenshots directory and files
	taskID := "test-task"
	screenshotsDir := filepath.Join(tmpDir, "test-vm", taskID, "screenshots")
	err = os.MkdirAll(screenshotsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create screenshots dir: %v", err)
	}

	screenshots := []string{"1.png", "2.png", "final.png"}
	for _, screenshot := range screenshots {
		path := filepath.Join(screenshotsDir, screenshot)
		err = os.WriteFile(path, []byte("fake png data"), 0644)
		if err != nil {
			t.Fatalf("Failed to write screenshot file: %v", err)
		}
	}

	// Get task screenshots
	req := httptest.NewRequest("GET", "/screenshots/"+taskID, nil)
	w := httptest.NewRecorder()

	server.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var retrievedScreenshots []string
	err = json.Unmarshal(w.Body.Bytes(), &retrievedScreenshots)
	if err != nil {
		t.Fatalf("Failed to unmarshal screenshots: %v", err)
	}

	if len(retrievedScreenshots) != len(screenshots) {
		t.Errorf("Expected %d screenshots, got %d", len(screenshots), len(retrievedScreenshots))
	}

	// Verify all screenshot names are present (order might vary)
	screenshotMap := make(map[string]bool)
	for _, screenshot := range retrievedScreenshots {
		screenshotMap[screenshot] = true
	}

	for _, expected := range screenshots {
		if !screenshotMap[expected] {
			t.Errorf("Expected screenshot %s not found", expected)
		}
	}
}

func TestServerErrorHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		path           string
		body           io.Reader
		expectedStatus int
	}{
		{
			name:           "spawn existing context",
			method:         "POST",
			path:           "/spawn/existing",
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "destroy non-existent context",
			method:         "DELETE",
			path:           "/contexts/non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "attach profile to non-existent context",
			method:         "POST",
			path:           "/profile/non-existent",
			body:           strings.NewReader(`{"id":"test"}`),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "get status of non-existent task",
			method:         "GET",
			path:           "/task/non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "get logs of non-existent task",
			method:         "GET",
			path:           "/logs/non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "get screenshots of non-existent task",
			method:         "GET",
			path:           "/screenshots/non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid JSON in task submission",
			method:         "POST",
			path:           "/task",
			body:           strings.NewReader(`{"invalid": json}`),
			expectedStatus: http.StatusBadRequest,
		},
	}

	// First, spawn a context for the conflict test
	req := httptest.NewRequest("POST", "/spawn/existing", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServerConcurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test concurrent context creation
	const numContexts = 10
	done := make(chan bool, numContexts)

	for i := 0; i < numContexts; i++ {
		go func(id int) {
			contextID := fmt.Sprintf("context-%d", id)
			req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
			w := httptest.NewRecorder()

			server.engine.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Context %d failed to spawn: status %d", id, w.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numContexts; i++ {
		<-done
	}

	// Verify all contexts were created
	server.mu.RLock()
	actualCount := len(server.contexts)
	server.mu.RUnlock()

	if actualCount != numContexts {
		t.Errorf("Expected %d contexts, got %d", numContexts, actualCount)
	}
}

func BenchmarkServerHealth(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkServerSpawnContext(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "viper-test-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := NewServer(":0", "test-vm", tmpDir)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		contextID := fmt.Sprintf("bench-context-%d", i)
		req := httptest.NewRequest("POST", "/spawn/"+contextID, nil)
		w := httptest.NewRecorder()

		server.engine.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}