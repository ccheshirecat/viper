package config

import (
	"github.com/ccheshirecat/viper/internal/types"
)

// DefaultVMConfig holds the opinionated defaults for VM creation
type DefaultVMConfig struct {
	// Networking defaults
	BridgeName    string
	NetworkMode   types.NetworkMode
	NetworkCIDR   string
	GatewayIP     string
	Netmask       string
	DNS           []string

	// Resource defaults
	DefaultMemory int // MB
	DefaultCPU    int // CPU shares

	// Nomad defaults
	DefaultDatacenter string
}

// DefaultConfig returns the opinionated defaults for Viper VMs
func DefaultConfig() *DefaultVMConfig {
	return &DefaultVMConfig{
		// Networking - private subnet with automatic bridge creation
		BridgeName:    "viperbr0",
		NetworkMode:   types.NetworkModePrivateSubnet,
		NetworkCIDR:   "192.168.1.0/24",
		GatewayIP:     "192.168.1.1",
		Netmask:       "24",
		DNS:           []string{"8.8.8.8", "1.1.1.1"},

		// Resources - reasonable defaults for browser automation
		DefaultMemory: 2048, // 2GB
		DefaultCPU:    2000, // 2 CPU cores

		// Nomad defaults
		DefaultDatacenter: "viper",
	}
}

// IsNetworkModeValid checks if a network mode is supported
func IsNetworkModeValid(mode types.NetworkMode) bool {
	switch mode {
	case types.NetworkModePrivateSubnet, types.NetworkModeStaticIP, types.NetworkModeHostShared:
		return true
	default:
		return false
	}
}
