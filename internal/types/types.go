package types

import "time"

// Task represents a browser automation task to be executed
type Task struct {
	ID        string        `json:"id"`
	VMID      string        `json:"vm_id"`
	URL       string        `json:"url"`
	Script    string        `json:"script,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Status    TaskStatus    `json:"status"`
	Created   time.Time     `json:"created"`
	Started   *time.Time    `json:"started,omitempty"`
	Completed *time.Time    `json:"completed,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusTimeout   TaskStatus = "timeout"
)

// Profile represents a browser profile configuration
type Profile struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name,omitempty"`
	Cookies      []Cookie                     `json:"cookies,omitempty"`
	LocalStorage map[string]map[string]string `json:"localStorage,omitempty"`
	UserAgent    string                       `json:"userAgent,omitempty"`
	Viewport     *Viewport                    `json:"viewport,omitempty"`
	Headers      map[string]string            `json:"headers,omitempty"`
}

// Cookie represents an HTTP cookie
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path,omitempty"`
	Expires  *int64 `json:"expires,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
}

// Viewport represents browser viewport dimensions
type Viewport struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

// VMConfig represents configuration for a microVM
type VMConfig struct {
	Name     string            `json:"name"`
	VMM      string            `json:"vmm"`      // cloudhypervisor, firecracker
	Contexts int               `json:"contexts"` // number of browser contexts
	GPU      bool              `json:"gpu"`      // enable GPU passthrough
	Memory   int               `json:"memory"`   // MB
	CPUs     int               `json:"cpus"`     // number of CPUs
	Disk     int               `json:"disk"`     // MB
	Network  string            `json:"network"`  // network configuration
	Labels   map[string]string `json:"labels"`   // arbitrary key-value labels
}

// VMStatus represents the current state of a microVM
type VMStatus struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`    // running, stopped, failed, etc.
	AgentURL string     `json:"agent_url"` // HTTP endpoint for agent
	Created  time.Time  `json:"created"`
	Started  *time.Time `json:"started,omitempty"`
	Stopped  *time.Time `json:"stopped,omitempty"`
	Health   string     `json:"health"`   // healthy, unhealthy, unknown
	Contexts []string   `json:"contexts"` // active browser context IDs
}

// BrowserContext represents an isolated browser session
type BrowserContext struct {
	ID       string     `json:"id"`
	VMID     string     `json:"vm_id"`
	Created  time.Time  `json:"created"`
	Profile  *Profile   `json:"profile,omitempty"`
	Active   bool       `json:"active"`
	LastUsed *time.Time `json:"last_used,omitempty"`
}

// TaskResult represents the outcome of a task execution
type TaskResult struct {
	TaskID      string            `json:"task_id"`
	Status      TaskStatus        `json:"status"`
	Screenshots []string          `json:"screenshots"` // file paths
	Logs        []string          `json:"logs"`        // file paths
	Metadata    map[string]string `json:"metadata"`    // arbitrary data
	Duration    time.Duration     `json:"duration"`
	Error       string            `json:"error,omitempty"`
}

// AgentHealth represents agent health status
type AgentHealth struct {
	Status    string            `json:"status"`     // healthy, unhealthy
	Version   string            `json:"version"`    // agent version
	Uptime    time.Duration     `json:"uptime"`     // how long agent has been running
	Contexts  int               `json:"contexts"`   // number of active contexts
	Tasks     int               `json:"tasks"`      // number of active tasks
	Memory    int64             `json:"memory"`     // memory usage in bytes
	LastCheck time.Time         `json:"last_check"` // last health check time
	Details   map[string]string `json:"details"`    // additional health info
}

// NetworkMode represents different VM networking strategies
type NetworkMode string

const (
	// NetworkModePrivateSubnet uses bridge networking with auto-assigned IPs
	NetworkModePrivateSubnet NetworkMode = "private_subnet"

	// NetworkModeStaticIP uses bridge networking with predefined static IPs
	NetworkModeStaticIP NetworkMode = "static_ip"

	// NetworkModeHostShared shares the host network namespace
	NetworkModeHostShared NetworkMode = "host_shared"
)

// VMNetworkConfig contains network configuration for a VM
type VMNetworkConfig struct {
	Mode     NetworkMode `json:"mode"`
	StaticIP string      `json:"static_ip,omitempty"`
	Bridge   string      `json:"bridge,omitempty"`
	Gateway  string      `json:"gateway,omitempty"`
	DNS      []string    `json:"dns,omitempty"`
}
