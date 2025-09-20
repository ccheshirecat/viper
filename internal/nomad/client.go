package nomad

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
	nomadapi "github.com/hashicorp/nomad/api"
)

type Client struct {
	client *nomadapi.Client
}

type SystemStatus struct {
	NomadStatus  string
	NomadDetails string
	VMCount      int
	VMDetails    string
	ActiveTasks  int
	TaskDetails  string
}

func NewClient() (*Client, error) {
	config := nomadapi.DefaultConfig()

	client, err := nomadapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Nomad client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// SubmitJob submits a job to Nomad and returns the job ID
func (c *Client) SubmitJob(ctx context.Context, job *nomadapi.Job) (string, error) {
	_, _, err := c.client.Jobs().Register(job, &nomadapi.WriteOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to register job: %w", err)
	}

	return *job.ID, nil
}

// Legacy CreateVM method removed - use job_generator.go instead

// ResolveVMAgentURL resolves a VM name to its agent URL using Nomad service discovery
func (c *Client) ResolveVMAgentURL(ctx context.Context, vmName string) (string, error) {
	// Try multiple job ID patterns to handle different naming conventions
	jobIDPatterns := []string{
		fmt.Sprintf("viper-vm-%s", vmName),
		vmName,
		fmt.Sprintf("viper-%s", vmName),
	}

	for _, jobID := range jobIDPatterns {
		url, err := c.resolveJobAgentURL(ctx, jobID)
		if err == nil {
			return url, nil
		}
	}

	return "", fmt.Errorf("failed to resolve agent URL for VM %s: no running allocations found", vmName)
}

// resolveJobAgentURL resolves a specific job ID to its agent URL
func (c *Client) resolveJobAgentURL(ctx context.Context, jobID string) (string, error) {
	// Get job allocations
	allocs, _, err := c.client.Jobs().Allocations(jobID, false, &nomadapi.QueryOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get allocations for job %s: %w", jobID, err)
	}

	// Find a running allocation
	for _, alloc := range allocs {
		if alloc.ClientStatus != "running" {
			continue
		}

		// Get detailed allocation info to extract network information
		allocDetail, _, err := c.client.Allocations().Info(alloc.ID, &nomadapi.QueryOptions{})
		if err != nil {
			continue // Try next allocation
		}

		// Extract IP address from allocation resources
		if allocDetail.Resources != nil && allocDetail.Resources.Networks != nil {
			for _, network := range allocDetail.Resources.Networks {
				if network.IP != "" {
					// Found a network with IP - construct agent URL
					return fmt.Sprintf("http://%s:8080", network.IP), nil
				}
			}
		}

		// Fallback: Check allocation network status for dynamic IPs
		if allocDetail.NetworkStatus != nil && allocDetail.NetworkStatus.Address != "" {
			return fmt.Sprintf("http://%s:8080", allocDetail.NetworkStatus.Address), nil
		}

		// Additional fallback: Try to extract IP from task states
		for _, taskState := range allocDetail.TaskStates {
			// Skip tasks that aren't running
			if taskState.State != "running" {
				continue
			}

			// Check if there are any network-related events that contain IPs
			for _, event := range taskState.Events {
				if event.Type == "Driver" && strings.Contains(event.DisplayMessage, "IP:") {
					// Extract IP from driver message (format varies by driver)
					if ip := extractIPFromMessage(event.DisplayMessage); ip != "" {
						return fmt.Sprintf("http://%s:8080", ip), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no running allocations found for job %s", jobID)
}

// extractIPFromMessage attempts to extract an IP address from a driver message
func extractIPFromMessage(message string) string {
	// Simple regex-like extraction for common IP patterns in driver messages
	// This handles various formats that nomad-driver-ch might use
	parts := strings.Fields(message)
	for i, part := range parts {
		if part == "IP:" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "ip=") {
			return strings.TrimPrefix(part, "ip=")
		}
		if strings.Contains(part, "allocated_ip:") {
			return strings.Split(part, ":")[1]
		}
		// Check if this looks like an IP address
		if isValidIPAddress(part) {
			return part
		}
	}
	return ""
}

// isValidIPAddress performs basic IP address validation
func isValidIPAddress(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func (c *Client) ListVMs(ctx context.Context) ([]types.VMStatus, error) {
	jobs, _, err := c.client.Jobs().List(&nomadapi.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	var vms []types.VMStatus
	for _, job := range jobs {
		if !strings.HasPrefix(job.ID, "viper-vm-") {
			continue
		}

		vmName := strings.TrimPrefix(job.ID, "viper-vm-")

		vm := types.VMStatus{
			Name:     vmName,
			Status:   job.Status,
			Health:   "unknown",
			Created:  time.Unix(0, job.SubmitTime),
			Contexts: []string{},
		}

		if job.Status == "running" {
			// Use service discovery to resolve the actual agent URL
			if agentURL, err := c.ResolveVMAgentURL(ctx, vmName); err == nil {
				vm.AgentURL = agentURL
				vm.Health = "healthy"
			} else {
				vm.AgentURL = fmt.Sprintf("http://%s:8080", vmName) // Fallback
				vm.Health = "unreachable"
			}
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

func (c *Client) DestroyVM(ctx context.Context, name string) error {
	jobID := fmt.Sprintf("viper-vm-%s", name)

	_, _, err := c.client.Jobs().Deregister(jobID, false, &nomadapi.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to deregister VM job: %w", err)
	}

	return nil
}

func (c *Client) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	leader, err := c.client.Status().Leader()
	if err != nil {
		return nil, fmt.Errorf("failed to get Nomad leader: %w", err)
	}

	status := &SystemStatus{
		NomadStatus:  "connected",
		NomadDetails: fmt.Sprintf("Leader: %s", leader),
	}

	jobs, _, err := c.client.Jobs().List(&nomadapi.QueryOptions{})
	if err == nil {
		vmCount := 0
		for _, job := range jobs {
			if strings.HasPrefix(job.ID, "viper-vm-") {
				vmCount++
			}
		}
		status.VMCount = vmCount
		status.VMDetails = fmt.Sprintf("%d managed VMs", vmCount)
	}

	return status, nil
}

func (c *Client) ListAvailableTemplates() ([]string, error) {
	return []string{"basic-vm", "gpu-vm", "minimal-vm"}, nil
}

func (c *Client) ValidateTemplate(templatePath string) error {
	return fmt.Errorf("template validation not yet implemented")
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func uintPtr(u uint64) *uint64 {
	return &u
}
