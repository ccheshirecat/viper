package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/viper-org/viper/internal/types"
)

type AgentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAgentClient(vmName string) (*AgentClient, error) {
	baseURL := fmt.Sprintf("http://%s:8080", vmName)

	return &AgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *AgentClient) Health(ctx context.Context) (*types.AgentHealth, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	var health types.AgentHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

func (c *AgentClient) SpawnContext(ctx context.Context, contextID string) error {
	url := fmt.Sprintf("%s/spawn/%s", c.baseURL, contextID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("spawn context failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *AgentClient) ListContexts(ctx context.Context) ([]types.BrowserContext, error) {
	url := fmt.Sprintf("%s/contexts", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list contexts failed with status: %d", resp.StatusCode)
	}

	var contexts []types.BrowserContext
	if err := json.NewDecoder(resp.Body).Decode(&contexts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return contexts, nil
}

func (c *AgentClient) DestroyContext(ctx context.Context, contextID string) error {
	url := fmt.Sprintf("%s/contexts/%s", c.baseURL, contextID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("destroy context failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *AgentClient) AttachProfile(ctx context.Context, contextID string, profile types.Profile) error {
	url := fmt.Sprintf("%s/profile/%s", c.baseURL, contextID)

	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("attach profile failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *AgentClient) SubmitTask(ctx context.Context, task types.Task) (*types.TaskResult, error) {
	url := fmt.Sprintf("%s/task", c.baseURL)

	data, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("submit task failed with status: %d", resp.StatusCode)
	}

	var result types.TaskResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *AgentClient) GetTaskStatus(ctx context.Context, taskID string) (*types.Task, error) {
	url := fmt.Sprintf("%s/task/%s", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task status failed with status: %d", resp.StatusCode)
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &task, nil
}

func (c *AgentClient) GetTaskLogs(ctx context.Context, taskID string) (string, error) {
	url := fmt.Sprintf("%s/logs/%s", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get task logs failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(data), nil
}

func (c *AgentClient) GetTaskScreenshots(ctx context.Context, taskID string) ([]string, error) {
	url := fmt.Sprintf("%s/screenshots/%s", c.baseURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task screenshots failed with status: %d", resp.StatusCode)
	}

	var screenshots []string
	if err := json.NewDecoder(resp.Body).Decode(&screenshots); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return screenshots, nil
}