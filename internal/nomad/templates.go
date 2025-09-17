package nomad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/ccheshirecat/viper/internal/types"
)

type TemplateParser struct {
	client       *nomadapi.Client
	templatesDir string
}

type JobParseRequest struct {
	JobHCL       string            `json:"JobHCL"`
	Canonicalize bool              `json:"Canonicalize"`
	Variables    map[string]string `json:"Variables,omitempty"`
}

func NewTemplateParser(client *nomadapi.Client) *TemplateParser {
	return &TemplateParser{
		client:       client,
		templatesDir: "jobs",
	}
}

func (tp *TemplateParser) ParseJobTemplate(config types.VMConfig) (*nomadapi.Job, error) {
	templatePath := tp.selectTemplate(config)

	hclContent, err := tp.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	job, err := tp.parseHCLToJob(hclContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HCL: %w", err)
	}

	job = tp.customizeJob(job, config)

	return job, nil
}

func (tp *TemplateParser) selectTemplate(config types.VMConfig) string {
	if config.GPU {
		gpuPath := filepath.Join("configs", "gpu-vm.nomad.hcl")
		if _, err := os.Stat(gpuPath); err == nil {
			return gpuPath
		}
	}
	return filepath.Join(tp.templatesDir, "example-vm.nomad.hcl")
}

func (tp *TemplateParser) loadTemplate(templatePath string) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}
	return string(content), nil
}

func (tp *TemplateParser) parseHCLToJob(hclContent string) (*nomadapi.Job, error) {
	parseReq := JobParseRequest{
		JobHCL:       hclContent,
		Canonicalize: true,
	}

	reqBytes, err := json.Marshal(parseReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parse request: %w", err)
	}

	// Get the Nomad server address from the client config
	config := tp.client.Address()
	parseURL := fmt.Sprintf("%s/v1/jobs/parse", config)

	resp, err := http.Post(parseURL, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to make parse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("parse request failed with status %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read parse response: %w", err)
	}

	var job nomadapi.Job
	if err := json.Unmarshal(respBytes, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (tp *TemplateParser) customizeJob(job *nomadapi.Job, config types.VMConfig) *nomadapi.Job {
	jobID := fmt.Sprintf("viper-vm-%s", config.Name)
	job.ID = &jobID
	job.Name = &jobID

	if len(job.TaskGroups) > 0 && len(job.TaskGroups[0].Tasks) > 0 {
		task := job.TaskGroups[0].Tasks[0]

		if task.Config == nil {
			task.Config = make(map[string]interface{})
		}

		task.Config["args"] = []string{
			"--listen=:8080",
			fmt.Sprintf("--vm-name=%s", config.Name),
			"--task-dir=/var/viper/tasks",
		}

		if config.GPU {
			existingArgs, ok := task.Config["args"].([]string)
			if ok {
				task.Config["args"] = append(existingArgs, "--gpu-enabled")
			}
		}

		if task.Resources != nil {
			if config.CPUs > 0 {
				task.Resources.CPU = intPtr(config.CPUs * 1000)
			}
			if config.Memory > 0 {
				task.Resources.MemoryMB = intPtr(config.Memory)
			}
			if config.Disk > 0 {
				task.Resources.DiskMB = intPtr(config.Disk)
			} else {
				// Default to 1GB if not specified to handle log storage
				task.Resources.DiskMB = intPtr(1024)
			}

			if config.GPU && task.Resources.Devices == nil {
				task.Resources.Devices = []*nomadapi.RequestedDevice{
					{
						Name:  "nvidia/gpu",
						Count: uintPtr(1),
					},
				}
			}
		}

		if len(task.Services) > 0 {
			serviceName := fmt.Sprintf("viper-agent-%s", config.Name)
			task.Services[0].Name = serviceName
		}

		if task.Env == nil {
			task.Env = make(map[string]string)
		}
		if config.CPUs > 0 {
			task.Env["GOMAXPROCS"] = fmt.Sprintf("%d", config.CPUs)
		}
		task.Env["GIN_MODE"] = "release"

		if config.GPU {
			task.Env["NVIDIA_VISIBLE_DEVICES"] = "all"
		}
	}

	if len(job.TaskGroups) > 0 && config.GPU {
		tg := job.TaskGroups[0]
		if tg.Constraints == nil {
			tg.Constraints = []*nomadapi.Constraint{}
		}

		found := false
		for _, constraint := range tg.Constraints {
			if constraint.LTarget == "${driver.nvidia.available}" {
				found = true
				break
			}
		}

		if !found {
			gpuConstraint := &nomadapi.Constraint{
				LTarget: "${driver.nvidia.available}",
				RTarget: "true",
				Operand: "=",
			}
			tg.Constraints = append(tg.Constraints, gpuConstraint)
		}
	}

	return job
}

func (tp *TemplateParser) ListAvailableTemplates() ([]string, error) {
	templates := []string{}

	dirs := []string{tp.templatesDir, "configs"}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".nomad.hcl") {
				templates = append(templates, filepath.Join(dir, entry.Name()))
			}
		}
	}

	return templates, nil
}

func (tp *TemplateParser) ValidateTemplate(templatePath string) error {
	hclContent, err := tp.loadTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	_, err = tp.parseHCLToJob(hclContent)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}