package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ccheshirecat/viper/internal/agent"
)

func main() {
	var (
		listen      = flag.String("listen", ":8080", "HTTP listen address")
		vmName      = flag.String("vm-name", "", "VM name identifier")
		taskDir     = flag.String("task-dir", "/var/viper/tasks", "Task storage directory")
		initNetwork = flag.Bool("init-network", true, "Initialize network interfaces (when running as PID 1)")
	)
	flag.Parse()

	// Detect if we're running as PID 1
	isPID1 := os.Getpid() == 1

	if isPID1 {
		log.Println("Running as PID 1 - initializing system...")
		if err := initializeSystem(); err != nil {
			log.Fatalf("Failed to initialize system: %v", err)
		}
	}

	// Initialize networking if requested (and detect VM name from network if not provided)
	if *initNetwork {
		if detectedVMName, err := initializeNetworking(); err != nil {
			log.Printf("Warning: Network initialization failed: %v", err)
		} else if *vmName == "" && detectedVMName != "" {
			*vmName = detectedVMName
			log.Printf("Detected VM name from network: %s", *vmName)
		}
	}

	// Auto-detect VM name from hostname if still not set
	if *vmName == "" {
		if hostname, err := os.Hostname(); err == nil {
			*vmName = hostname
			log.Printf("Using hostname as VM name: %s", *vmName)
		} else {
			log.Fatal("vm-name is required and could not be auto-detected")
		}
	}

	server, err := agent.NewServer(*listen, *vmName, *taskDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		log.Printf("Agent starting for VM %s on %s", *vmName, *listen)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	if isPID1 {
		// As PID 1, handle more signals
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGCHLD)
	} else {
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	}

	<-sigChan

	log.Println("Shutting down agent...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	if isPID1 {
		// As PID 1, we should cleanly shut down the system
		log.Println("Shutting down system...")
		time.Sleep(1 * time.Second) // Give processes time to clean up
	}
}

// initializeSystem performs basic system initialization when running as PID 1
func initializeSystem() error {
	// Mount essential filesystems
	if err := mountEssentialFilesystems(); err != nil {
		return fmt.Errorf("failed to mount filesystems: %w", err)
	}

	// Set up basic device nodes if needed
	if err := setupDeviceNodes(); err != nil {
		return fmt.Errorf("failed to setup device nodes: %w", err)
	}

	return nil
}

// mountEssentialFilesystems mounts proc, sys, etc.
func mountEssentialFilesystems() error {
	mounts := []struct {
		source, target, fstype, flags string
	}{
		{"proc", "/proc", "proc", ""},
		{"sysfs", "/sys", "sysfs", ""},
		{"devtmpfs", "/dev", "devtmpfs", ""},
		{"tmpfs", "/tmp", "tmpfs", ""},
		{"tmpfs", "/run", "tmpfs", ""},
	}

	for _, mount := range mounts {
		// Check if already mounted
		if _, err := os.Stat(mount.target); os.IsNotExist(err) {
			if err := os.MkdirAll(mount.target, 0755); err != nil {
				continue // Skip if we can't create the directory
			}
		}

		cmd := exec.Command("mount", "-t", mount.fstype, mount.source, mount.target)
		if err := cmd.Run(); err != nil {
			// Don't fail if mount fails - it might already be mounted
			log.Printf("Warning: Failed to mount %s: %v", mount.target, err)
		}
	}

	return nil
}

// setupDeviceNodes creates essential device nodes
func setupDeviceNodes() error {
	// Create basic device nodes if they don't exist
	devices := []struct {
		name, path   string
		major, minor int
		mode         os.FileMode
	}{
		{"null", "/dev/null", 1, 3, 0666},
		{"zero", "/dev/zero", 1, 5, 0666},
		{"random", "/dev/random", 1, 8, 0666},
		{"urandom", "/dev/urandom", 1, 9, 0666},
	}

	for _, dev := range devices {
		if _, err := os.Stat(dev.path); os.IsNotExist(err) {
			// Device doesn't exist, try to create it
			cmd := exec.Command("mknod", dev.path, "c", fmt.Sprintf("%d", dev.major), fmt.Sprintf("%d", dev.minor))
			cmd.Run() // Ignore errors - might not have permissions
		}
	}

	return nil
}

// initializeNetworking sets up network interfaces
func initializeNetworking() (string, error) {
	// Bring up loopback interface
	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		log.Printf("Warning: Failed to bring up loopback: %v", err)
	}

	// Look for eth0 interface and try to bring it up
	if _, err := net.InterfaceByName("eth0"); err == nil {
		log.Println("Found eth0 interface, attempting to configure...")

		// Try to bring up eth0
		if err := exec.Command("ip", "link", "set", "eth0", "up").Run(); err != nil {
			log.Printf("Warning: Failed to bring up eth0: %v", err)
		}

		// Try DHCP first (this will work with most nomad-driver-ch configurations)
		if err := tryDHCP("eth0"); err != nil {
			log.Printf("DHCP failed, checking for static configuration: %v", err)

			// If DHCP fails, check environment for static IP configuration
			if staticIP := os.Getenv("VM_IP"); staticIP != "" {
				if err := configureStaticIP("eth0", staticIP); err != nil {
					return "", fmt.Errorf("failed to configure static IP: %w", err)
				}
			}
		}

		// Try to detect VM name from network configuration
		if vmName := detectVMNameFromNetwork(); vmName != "" {
			return vmName, nil
		}
	}

	return "", nil
}

// tryDHCP attempts to get an IP address via DHCP
func tryDHCP(iface string) error {
	// Try different DHCP clients
	clients := []string{"dhclient", "udhcpc", "dhcpcd"}

	for _, client := range clients {
		if _, err := exec.LookPath(client); err == nil {
			log.Printf("Trying DHCP with %s...", client)

			var cmd *exec.Cmd
			switch client {
			case "dhclient":
				cmd = exec.Command("dhclient", iface)
			case "udhcpc":
				cmd = exec.Command("udhcpc", "-i", iface, "-n")
			case "dhcpcd":
				cmd = exec.Command("dhcpcd", iface)
			}

			if err := cmd.Run(); err == nil {
				log.Printf("Successfully configured %s via DHCP", iface)
				return nil
			} else {
				log.Printf("%s failed: %v", client, err)
			}
		}
	}

	return fmt.Errorf("no working DHCP client found")
}

// configureStaticIP configures a static IP address
func configureStaticIP(iface, ipConfig string) error {
	// Expected format: IP/CIDR or just IP
	if !strings.Contains(ipConfig, "/") {
		ipConfig += "/24" // Default to /24
	}

	log.Printf("Configuring static IP %s on %s", ipConfig, iface)
	if err := exec.Command("ip", "addr", "add", ipConfig, "dev", iface).Run(); err != nil {
		return fmt.Errorf("failed to add IP address: %w", err)
	}

	// Add default route if gateway is specified
	if gateway := os.Getenv("VM_GATEWAY"); gateway != "" {
		log.Printf("Adding default route via %s", gateway)
		if err := exec.Command("ip", "route", "add", "default", "via", gateway).Run(); err != nil {
			log.Printf("Warning: Failed to add default route: %v", err)
		}
	}

	return nil
}

// detectVMNameFromNetwork tries to detect VM name from network configuration
func detectVMNameFromNetwork() string {
	// Try to get hostname from DHCP lease or reverse DNS
	if hostname, err := os.Hostname(); err == nil && hostname != "localhost" {
		return hostname
	}

	// Try to extract from IP address pattern (e.g., viper-vm-192-168-1-100)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				// Convert IP to hostname pattern
				ipStr := strings.ReplaceAll(ipNet.IP.String(), ".", "-")
				return fmt.Sprintf("viper-vm-%s", ipStr)
			}
		}
	}

	return ""
}
