package nomad

import (
	"fmt"
	"path/filepath"

	"github.com/ccheshirecat/viper/internal/types"
	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper/pointer"
)

// VMCreateOptions contains options for creating a VM
type VMCreateOptions struct {
	Name        string
	Memory      int  // MB
	CPU         int  // CPU shares (1000 = 1 core)
	NetworkMode types.NetworkMode
	StaticIP    string // Only used if NetworkMode is Static
	ImagePaths  ImagePaths
}

// ImagePaths contains paths to VM artifacts
type ImagePaths struct {
	Kernel    string
	Initramfs string
	DiskImage string // Optional
}

// VMJobGenerator generates Nomad jobs for Viper VMs
type VMJobGenerator struct {
	defaultDatacenter string
	defaultBridge     string
	defaultImagePaths ImagePaths
}

// NewVMJobGenerator creates a new job generator
func NewVMJobGenerator(datacenter, bridge string, imagePaths ImagePaths) *VMJobGenerator {
	return &VMJobGenerator{
		defaultDatacenter: datacenter,
		defaultBridge:     bridge,
		defaultImagePaths: imagePaths,
	}
}

// GenerateVMJob creates a Nomad job for a Viper VM
func (g *VMJobGenerator) GenerateVMJob(opts VMCreateOptions) (*nomadapi.Job, error) {
	// Use defaults if not specified
	if opts.CPU == 0 {
		opts.CPU = 1000 // 1 CPU core
	}
	if opts.Memory == 0 {
		opts.Memory = 1024 // 1GB RAM
	}
	if opts.ImagePaths.Kernel == "" {
		opts.ImagePaths = g.defaultImagePaths
	}

	job := &nomadapi.Job{
		ID:          pointer.Of(opts.Name),
		Name:        pointer.Of(opts.Name),
		Type:        pointer.Of("service"),
		Datacenters: []string{g.defaultDatacenter},
		TaskGroups: []*nomadapi.TaskGroup{
			{
				Name:  pointer.Of("browser"),
				Count: pointer.Of(1),
				Networks: []*nomadapi.NetworkResource{
					{
						Mode: "bridge",
						DynamicPorts: []*nomadapi.Port{
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
						Driver: "nomad-driver-ch", // Your driver name
						Config: g.generateTaskConfig(opts),
						Resources: &nomadapi.Resources{
							CPU:    pointer.Of(opts.CPU),
							MemoryMB: pointer.Of(opts.Memory),
						},
						Services: []*nomadapi.Service{
							{
								Name:      "viper-agent",
								PortLabel: "agent",
								Checks: []*nomadapi.ServiceCheck{
									{
										Type:     "tcp",
										PortLabel: "agent",
										Interval: 30000000000, // 30s in nanoseconds
										Timeout:  5000000000,  // 5s in nanoseconds
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

	// Network configuration based on mode
	switch opts.NetworkMode {
	case types.NetworkModePrivateSubnet:
		config["network_interface"] = map[string]interface{}{
			"bridge": map[string]interface{}{
				"name": g.defaultBridge,
				// IP will be auto-assigned from driver pool
			},
		}

	case types.NetworkModeStaticIP:
		if opts.StaticIP == "" {
			// Auto-generate static IP if not provided
			opts.StaticIP = "192.168.1.150" // TODO: Make this smarter
		}
		config["network_interface"] = map[string]interface{}{
			"bridge": map[string]interface{}{
				"name":       g.defaultBridge,
				"static_ip":  opts.StaticIP,
				"gateway":    "192.168.1.1",
				"netmask":    "24",
				"dns":        []string{"8.8.8.8", "1.1.1.1"},
			},
		}

	case types.NetworkModeHostShared:
		// For host networking, we'd need to check if your driver supports this
		// For now, fall back to private subnet
		config["network_interface"] = map[string]interface{}{
			"bridge": map[string]interface{}{
				"name": g.defaultBridge,
			},
		}
	}

	return config
}

// GenerateJobHCL creates HCL representation for debugging/manual use
func (g *VMJobGenerator) GenerateJobHCL(opts VMCreateOptions) (string, error) {
	networkConfig := ""
	serviceConfig := ""

	switch opts.NetworkMode {
	case types.NetworkModeStaticIP:
		if opts.StaticIP == "" {
			opts.StaticIP = "192.168.1.150"
		}
		networkConfig = fmt.Sprintf(`
        network_interface {
          bridge {
            name = "%s"
            static_ip = "%s"
            gateway = "192.168.1.1"
            netmask = "24"
            dns = ["8.8.8.8", "1.1.1.1"]
          }
        }`, g.defaultBridge, opts.StaticIP)

		serviceConfig = fmt.Sprintf(`
      service {
        name = "viper-agent-%s"
        address = "%s"
        port = 8080

        check {
          type = "tcp"
          address = "%s"
          port = 8080
          interval = "30s"
          timeout = "5s"
        }
      }`, opts.Name, opts.StaticIP, opts.StaticIP)

	default: // Private subnet
		networkConfig = fmt.Sprintf(`
        network_interface {
          bridge {
            name = "%s"
          }
        }`, g.defaultBridge)

		serviceConfig = `
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
	}

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
      driver = "nomad-driver-ch"

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
		g.defaultDatacenter,
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