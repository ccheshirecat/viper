package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ccheshirecat/viper/internal/nomad"
	"github.com/ccheshirecat/viper/pkg/client"
)

// Example script demonstrating the new VM IP discovery functionality
func main() {
	fmt.Println("🔍 Viper Service Discovery Test")
	fmt.Println("================================")

	// Test Nomad connectivity
	fmt.Println("1. Testing Nomad connectivity...")
	nomadClient, err := nomad.NewClient()
	if err != nil {
		log.Fatalf("❌ Failed to create Nomad client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := nomadClient.GetSystemStatus(ctx)
	if err != nil {
		log.Fatalf("❌ Nomad not accessible: %v", err)
	}

	fmt.Printf("✅ Nomad connected: %s\n", status.NomadDetails)

	// List current VMs
	fmt.Println("\n2. Listing current VMs...")
	vms, err := nomadClient.ListVMs(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to list VMs: %v", err)
	}

	if len(vms) == 0 {
		fmt.Println("ℹ️  No VMs currently running")
		fmt.Println("\nTo test service discovery:")
		fmt.Println("1. Run: viper vms create test-vm")
		fmt.Println("2. Wait for VM to boot")
		fmt.Println("3. Run this script again")
		return
	}

	fmt.Printf("✅ Found %d VMs:\n", len(vms))

	// Test service discovery for each VM
	for _, vm := range vms {
		fmt.Printf("\n3. Testing service discovery for VM: %s\n", vm.Name)
		fmt.Printf("   Status: %s, Health: %s\n", vm.Status, vm.Health)

		if vm.Status == "running" && vm.AgentURL != "" {
			fmt.Printf("   Agent URL: %s\n", vm.AgentURL)

			// Test direct communication with agent
			fmt.Println("   Testing agent communication...")

			// Create agent client using new service discovery
			agentClient, err := client.NewAgentClient(vm.Name)
			if err != nil {
				fmt.Printf("   ❌ Failed to create agent client: %v\n", err)
				continue
			}

			// Test health endpoint
			health, err := agentClient.Health(ctx)
			if err != nil {
				fmt.Printf("   ❌ Failed to get agent health: %v\n", err)
				continue
			}

			fmt.Printf("   ✅ Agent healthy - uptime: %v, contexts: %d\n",
				health.Uptime, health.Contexts)

			// Test URL refresh capability
			fmt.Println("   Testing URL refresh...")
			if err := agentClient.RefreshAgentURL(ctx); err != nil {
				fmt.Printf("   ❌ Failed to refresh URL: %v\n", err)
			} else {
				fmt.Printf("   ✅ URL refresh successful\n")
			}

		} else {
			fmt.Printf("   ⚠️  VM not ready for communication\n")
		}
	}

	fmt.Println("\n🎉 Service discovery test completed!")
	fmt.Println("\nKey improvements:")
	fmt.Println("• CLI now automatically discovers VM IPs via Nomad")
	fmt.Println("• No more hardcoded hostnames or IP addresses")
	fmt.Println("• Support for dynamic IP assignment in microVMs")
	fmt.Println("• Graceful handling of VM IP changes")
}