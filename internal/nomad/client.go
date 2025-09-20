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

func (c *Client) ListVMs(ctx context.Context) ([]types.VMStatus, error) {
	jobs, _, err := c.client.Jobs().List(&nomadapi.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	var vms []types.VMStatus
	for _, job := range jobs {
		// Use the new flexible job identification
		if !isViperJob(job.ID) {
			continue
		}

		vmName := extractVMNameFromJobID(job.ID)

		vm := types.VMStatus{
			Name:     vmName,
			Status:   job.Status,
			Health:   "unknown",
			Created:  time.Unix(0, job.SubmitTime),
			Contexts: []string{},
		}

		if job.Status == "running" {
			// Use service discovery to resolve actual agent URL
			if agentURL, err := c.resolveAgentURL(ctx, job.ID); err == nil {
				vm.AgentURL = agentURL
				vm.Health = "healthy"
			} else {
				// Log the error but don't fail the entire listing
				vm.Health = "unreachable"
				vm.AgentURL = "" // Clear any previous URL
			}
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

func (c *Client) DestroyVM(ctx context.Context, name string) error {
	// Try to find the actual job ID for this VM
	jobs, _, err := c.client.Jobs().List(&nomadapi.QueryOptions{})
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	var jobID string
	for _, job := range jobs {
		if isViperJob(job.ID) && extractVMNameFromJobID(job.ID) == name {
			jobID = job.ID
			break
		}
	}

	if jobID == "" {
		return fmt.Errorf("VM job not found: %s", name)
	}

	_, _, err = c.client.Jobs().Deregister(jobID, false, &nomadapi.WriteOptions{})
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

// Service discovery methods for VM IP resolution

// ResolveVMAgentURL resolves a VM name to its actual agent URL via Nomad service discovery
func (c *Client) ResolveVMAgentURL(ctx context.Context, vmName string) (string, error) {
	// Try multiple job ID patterns since we may have different naming conventions
	possibleJobIDs := []string{
		vmName,           // Direct name
		"viper-" + vmName, // Prefixed name
	}

	for _, jobID := range possibleJobIDs {
		if url, err := c.resolveAgentURL(ctx, jobID); err == nil {
			return url, nil
		}
	}

	return "", fmt.Errorf("no agent URL found for VM: %s", vmName)
}

// resolveAgentURL resolves the actual agent URL for a job via allocation inspection
func (c *Client) resolveAgentURL(ctx context.Context, jobID string) (string, error) {
	// Use allocation-based IP resolution as primary method
	return c.resolveAgentURLViaAllocations(ctx, jobID)
}

// resolveAgentURLViaAllocations attempts to resolve via allocation info
func (c *Client) resolveAgentURLViaAllocations(ctx context.Context, jobID string) (string, error) {
	// Get allocations for the job
	allocs, _, err := c.client.Jobs().Allocations(jobID, false, &nomadapi.QueryOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get allocations for job %s: %w", jobID, err)
	}

	// Find running allocation
	for _, alloc := range allocs {
		if alloc.ClientStatus == "running" {
			// Get detailed allocation info
			allocInfo, _, err := c.client.Allocations().Info(alloc.ID, &nomadapi.QueryOptions{})
			if err != nil {
				continue
			}

			// Extract network information
			if url := c.extractURLFromAllocation(allocInfo); url != "" {
				return url, nil
			}
		}
	}

	return "", fmt.Errorf("no running allocation found for job: %s", jobID)
}

// extractURLFromAllocation extracts the agent URL from allocation information
func (c *Client) extractURLFromAllocation(alloc *nomadapi.Allocation) string {
	// Check network information in the allocation
	if alloc.Resources != nil && len(alloc.Resources.Networks) > 0 {
		network := alloc.Resources.Networks[0]

		// Look for the agent port mapping
		for _, port := range network.DynamicPorts {
			if port.Label == "agent" {
				// Construct URL using network IP and mapped port
				if network.IP != "" {
					return fmt.Sprintf("http://%s:%d", network.IP, port.Value)
				}
			}
		}

		// Fallback: use network IP with default port
		if network.IP != "" {
			return fmt.Sprintf("http://%s:8080", network.IP)
		}
	}

	return ""
}

// Helper functions for job identification

// isViperJob checks if a job ID belongs to a Viper-managed VM
func isViperJob(jobID string) bool {
	// Current pattern is direct job names, but we can be more flexible
	return jobID != "" && !strings.HasPrefix(jobID, "system-") && !strings.HasPrefix(jobID, "nomad-")
}

// extractVMNameFromJobID extracts the VM name from a job ID
func extractVMNameFromJobID(jobID string) string {
	// Handle different naming patterns
	if strings.HasPrefix(jobID, "viper-") {
		return strings.TrimPrefix(jobID, "viper-")
	}
	return jobID
}
