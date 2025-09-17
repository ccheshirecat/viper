package nomad

import (
	"context"
	"fmt"
	"strings"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/viper-org/viper/internal/types"
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

	return &Client{client: client}, nil
}

func (c *Client) CreateVM(ctx context.Context, config types.VMConfig) (*types.VMStatus, error) {
	job := c.buildVMJob(config)

	_, _, err := c.client.Jobs().Register(job, &nomadapi.WriteOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to register VM job: %w", err)
	}

	status := &types.VMStatus{
		Name:    config.Name,
		Status:  "pending",
		Health:  "unknown",
		Created: time.Now(),
		Contexts: []string{},
	}

	return status, nil
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

func (c *Client) buildVMJob(config types.VMConfig) *nomadapi.Job {
	jobID := fmt.Sprintf("viper-vm-%s", config.Name)

	job := &nomadapi.Job{
		ID:          &jobID,
		Name:        &jobID,
		Type:        stringPtr("service"),
		Datacenters: []string{"dc1"},
		TaskGroups: []*nomadapi.TaskGroup{
			{
				Name:  stringPtr("vm-group"),
				Count: intPtr(1),
				Tasks: []*nomadapi.Task{
					{
						Name:   "agent",
						Driver: "exec",
						Config: map[string]interface{}{
							"command": "/usr/local/bin/viper-agent",
							"args":    []string{"--listen=:8080", fmt.Sprintf("--vm-name=%s", config.Name)},
						},
						Resources: &nomadapi.Resources{
							CPU:      intPtr(config.CPUs * 1000),
							MemoryMB: intPtr(config.Memory),
							Networks: []*nomadapi.NetworkResource{
								{
									Mode: "host",
									DynamicPorts: []nomadapi.Port{
										{Label: "http", Value: 8080},
									},
								},
							},
						},
						Services: []*nomadapi.Service{
							{
								Name:      fmt.Sprintf("viper-agent-%s", config.Name),
								PortLabel: "http",
								Checks: []nomadapi.ServiceCheck{
									{
										Type:     "http",
										Path:     "/health",
										Interval: time.Duration(10 * time.Second),
										Timeout:  time.Duration(3 * time.Second),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if config.GPU {
		job.TaskGroups[0].Tasks[0].Resources.Devices = []*nomadapi.RequestedDevice{
			{
				Name:  "nvidia/gpu",
				Count: uintPtr(1),
			},
		}
	}

	return job
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