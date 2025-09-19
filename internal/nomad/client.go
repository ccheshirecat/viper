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
			vm.AgentURL = fmt.Sprintf("http://%s:8080", vmName)
			vm.Health = "healthy"
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
	return c.templateParser.ListAvailableTemplates()
}

func (c *Client) ValidateTemplate(templatePath string) error {
	return c.templateParser.ValidateTemplate(templatePath)
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
