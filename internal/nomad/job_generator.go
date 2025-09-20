package nomad

import (
	"fmt"
	"path/filepath"

	"github.com/ccheshirecat/viper/internal/config"
	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper/pointer"
)

// VMCreateOptions contains options for creating a VM
type VMCreateOptions struct {
	Name       string
	Memory     int // MB
	CPU        int // CPU shares (1000 = 1 core)
	ImagePaths ImagePaths
}

// ImagePaths contains paths to VM artifacts
type ImagePaths struct {
	Kernel    string
	Initramfs string
	DiskImage string // Optional
}

// VMJobGenerator generates Nomad jobs for Viper VMs
type VMJobGenerator struct {
	config            *config.DefaultVMConfig
	defaultImagePaths ImagePaths
}

// NewVMJobGenerator creates a new job generator
func NewVMJobGenerator(datacenter string, imagePaths ImagePaths) *VMJobGenerator {
	cfg := config.DefaultConfig()
	if datacenter != "" {
		cfg.DefaultDatacenter = datacenter
	}

	return &VMJobGenerator{
		config:            cfg,
		defaultImagePaths: imagePaths,
	}
}

// GenerateVMJob creates a Nomad job for a Viper VM
func (g *VMJobGenerator) GenerateVMJob(opts VMCreateOptions) (*nomadapi.Job, error) {
	// Use defaults if not specified
	if opts.CPU == 0 {
		opts.CPU = g.config.DefaultCPU
	}
	if opts.Memory == 0 {
		opts.Memory = g.config.DefaultMemory
	}
	if opts.ImagePaths.Kernel == "" {
		opts.ImagePaths = g.defaultImagePaths
	}

	job := &nomadapi.Job{
		ID:          pointer.Of(opts.Name),
		Name:        pointer.Of(opts.Name),
		Type:        pointer.Of("service"),
		Datacenters: []string{g.config.DefaultDatacenter},
		TaskGroups: []*nomadapi.TaskGroup{
			{
				Name:  pointer.Of("browser"),
				Count: pointer.Of(1),
				Networks: []*nomadapi.NetworkResource{
					{
						Mode: "bridge",
						DynamicPorts: []nomadapi.Port{
							{
								Label: "agent",
								To:    8080, // Agent listens on 8080 inside VM
							},
						},
					},
				},
				Tasks: []*nomadapi.Task{
					{
						Name:   opts.Name + "-vm",
						Driver: "ch", // nomad-driver-ch registers as "ch"
						Config: g.generateTaskConfig(opts),
						Resources: &nomadapi.Resources{
							CPU:      pointer.Of(opts.CPU),
							MemoryMB: pointer.Of(opts.Memory),
						},
						Services: []*nomadapi.Service{
							{
								Name:      "viper-agent",
								PortLabel: "agent",
								Checks: []nomadapi.ServiceCheck{
									{
										Type:      "tcp",
										PortLabel: "agent",
										Interval:  30000000000, // 30s in nanoseconds
										Timeout:   5000000000,  // 5s in nanoseconds
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job, nil
}

// DebugGenerateJob generates a job and returns debug information about the configuration
func (g *VMJobGenerator) DebugGenerateJob(opts VMCreateOptions) (*nomadapi.Job, map[string]interface{}, error) {
	job, err := g.GenerateVMJob(opts)
	if err != nil {
		return nil, nil, err
	}

	// Extract the task config for debugging
	if len(job.TaskGroups) > 0 && len(job.TaskGroups[0].Tasks) > 0 {
		task := job.TaskGroups[0].Tasks[0]
		return job, task.Config, nil
	}

	return job, nil, nil
}

// generateTaskConfig creates the task configuration for nomad-driver-ch
func (g *VMJobGenerator) generateTaskConfig(opts VMCreateOptions) map[string]interface{} {
	config := map[string]interface{}{
		// Required: kernel and initramfs
		"kernel":    opts.ImagePaths.Kernel,
		"initramfs": opts.ImagePaths.Initramfs,

		// VM configuration
		"hostname": opts.Name,
		"cmdline":  "console=ttyS0 init=/usr/local/bin/viper-agent",
	}

	// Optional disk image
	if opts.ImagePaths.DiskImage != "" {
		config["image"] = opts.ImagePaths.DiskImage
	}

	// Opinionated networking: private subnet with automatic bridge
	config["network_interface"] = map[string]interface{}{
		"bridge": map[string]interface{}{
			"name": g.config.BridgeName,
			// Let nomad-driver-ch assign IP from our configured subnet
		},
	}

	return config
}

// GenerateJobHCL creates HCL representation for debugging/manual use
func (g *VMJobGenerator) GenerateJobHCL(opts VMCreateOptions) (string, error) {
	// Opinionated networking configuration
	networkConfig := fmt.Sprintf(`
        network_interface {
          bridge {
            name = "%s"
          }
        }`, g.config.BridgeName)

	// Service discovery with dynamic IP resolution
	serviceConfig := `
      service {
        name = "viper-agent"
        port = "agent"

        check {
          type = "tcp"
          port = "agent"
          interval = "30s"
          timeout = "5s"
        }
      }`

	diskConfig := ""
	if opts.ImagePaths.DiskImage != "" {
		diskConfig = fmt.Sprintf(`
        image = "%s"`, opts.ImagePaths.DiskImage)
	}

	hcl := fmt.Sprintf(`job "%s" {
  datacenters = ["%s"]
  type        = "service"

  group "browser" {
    count = 1

    task "%s-vm" {
      driver = "ch"

      config {
        kernel = "%s"
        initramfs = "%s"%s

        hostname = "%s"
        cmdline = "console=ttyS0 init=/usr/local/bin/viper-agent"
%s
      }

      resources {
        cpu    = %d
        memory = %d
      }
%s
    }

    network {
      port "agent" {
        to = 8080
      }
    }
  }
}`,
		opts.Name,
		g.config.DefaultDatacenter,
		opts.Name,
		opts.ImagePaths.Kernel,
		opts.ImagePaths.Initramfs,
		diskConfig,
		opts.Name,
		networkConfig,
		opts.CPU,
		opts.Memory,
		serviceConfig,
	)

	return hcl, nil
}

// ResolveImagePaths resolves relative paths to absolute paths
func ResolveImagePaths(basePath string) ImagePaths {
	return ImagePaths{
		Kernel:    filepath.Join(basePath, "vmlinuz"),
		Initramfs: filepath.Join(basePath, "viper-initramfs.gz"),
		DiskImage: filepath.Join(basePath, "viper-headless.qcow2"),
	}
}
