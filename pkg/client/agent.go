package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/internal/types"
)

type AgentClient struct {
	baseURL     string
	vmName      string
	httpClient  *http.Client
	nomadClient *nomad.Client
}

// NewAgentClient creates an AgentClient that uses Nomad service discovery to resolve VM IPs
func NewAgentClient(vmName string) (*AgentClient, error) {
	nomadClient, err := nomad.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Nomad client for service discovery: %w", err)
	}

	client := &AgentClient{
		vmName:      vmName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		nomadClient: nomadClient,
	}

	// Attempt to resolve the VM's agent URL
	if err := client.RefreshAgentURL(context.Background()); err != nil {
		// Don't fail creation - allow lazy resolution
		client.baseURL = fmt.Sprintf("http://%s:8080", vmName) // Fallback
	}

	return client, nil
}

// NewAgentClientWithURL creates an AgentClient with a direct URL (for testing or direct connections)
func NewAgentClientWithURL(baseURL string) *AgentClient {
	return &AgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RefreshAgentURL refreshes the agent URL using Nomad service discovery
func (c *AgentClient) RefreshAgentURL(ctx context.Context) error {
	if c.nomadClient == nil {
		return fmt.Errorf("nomad client not available for service discovery")
	}

	url, err := c.nomadClient.ResolveVMAgentURL(ctx, c.vmName)
	if err != nil {
		return fmt.Errorf("failed to resolve agent URL for VM %s: %w", c.vmName, err)
	}

	c.baseURL = url
	return nil
}

func (c *AgentClient) Health(ctx context.Context) (*types.AgentHealth, error) {
	return c.performRequestWithRetry(ctx, func() (*types.AgentHealth, error) {
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
	})
}

// performRequestWithRetry performs a request with automatic URL refresh on failure for AgentHealth
func (c *AgentClient) performRequestWithRetry(ctx context.Context, operation func() (*types.AgentHealth, error)) (*types.AgentHealth, error) {
	// First attempt
	result, err := operation()
	if err == nil {
		return result, nil
	}

	// If we have a nomad client, try refreshing the URL and retry once
	if c.nomadClient != nil {
		if refreshErr := c.RefreshAgentURL(ctx); refreshErr == nil {
			// Retry with the refreshed URL
			result, retryErr := operation()
			if retryErr == nil {
				return result, nil
			}
			// If retry also fails, return the original error
		}
	}

	return nil, err
}

// performRequestWithRetryGeneric performs a request with automatic URL refresh on failure
func performRequestWithRetryGeneric[T any](c *AgentClient, ctx context.Context, operation func() (*T, error)) (*T, error) {
	// First attempt
	result, err := operation()
	if err == nil {
		return result, nil
	}

	// If we have a nomad client, try refreshing the URL and retry once
	if c.nomadClient != nil {
		if refreshErr := c.RefreshAgentURL(ctx); refreshErr == nil {
			// Retry with the refreshed URL
			result, retryErr := operation()
			if retryErr == nil {
				return result, nil
			}
			// If retry also fails, return the original error
		}
	}

	return nil, err
}

// performRequestWithRetrySlice performs a request with automatic URL refresh on failure for slices
func performRequestWithRetrySlice[T any](c *AgentClient, ctx context.Context, operation func() ([]T, error)) ([]T, error) {
	// First attempt
	result, err := operation()
	if err == nil {
		return result, nil
	}

	// If we have a nomad client, try refreshing the URL and retry once
	if c.nomadClient != nil {
		if refreshErr := c.RefreshAgentURL(ctx); refreshErr == nil {
			// Retry with the refreshed URL
			result, retryErr := operation()
			if retryErr == nil {
				return result, nil
			}
			// If retry also fails, return the original error
		}
	}

	return nil, err
}

// performRequestWithRetryVoid is similar but for operations that don't return data
func (c *AgentClient) performRequestWithRetryVoid(ctx context.Context, operation func() error) error {
	// First attempt
	err := operation()
	if err == nil {
		return nil
	}

	// If we have a nomad client, try refreshing the URL and retry once
	if c.nomadClient != nil {
		if refreshErr := c.RefreshAgentURL(ctx); refreshErr == nil {
			// Retry with the refreshed URL
			retryErr := operation()
			if retryErr == nil {
				return nil
			}
			// If retry also fails, return the original error
		}
	}

	return err
}

func (c *AgentClient) SpawnContext(ctx context.Context, contextID string) error {
	return c.performRequestWithRetryVoid(ctx, func() error {
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
	})
}

func (c *AgentClient) ListContexts(ctx context.Context) ([]types.BrowserContext, error) {
	return performRequestWithRetrySlice(c, ctx, func() ([]types.BrowserContext, error) {
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
	})
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
