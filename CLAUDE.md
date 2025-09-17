# Viper Engineering Doctrine

*(Every line of code is written as if it will ship to production tomorrow and must outlive its author)*

---

## **1. Production-First Philosophy**

- Every single line of code written must be with the **sole intention of being complete and production-ready** from the beginning.
- **No placeholders. No shortcuts. No “// TODO: implement later”.**
- If it’s in the codebase, it is expected to **work as intended, safely, reliably**.
- If you cannot implement something at production-level today → **do not add a stub**. Write it down in the project planner/issues instead.

---

## **2. Ownership & Accountability**

- Write every line as if **nobody else will ever touch this project again**.
- Take **full ownership** of the design, code, tests, and documentation.
- Every decision made in code must be backed by a clear reason that can withstand external review.
- Treat this as if you are the **sole engineer responsible for the entire product**, even if others are working alongside you.

---

## **3. No Dogma, Only Value**

- We do not follow “best practices” for their own sake.
- We follow what is **objectively valuable** for:
    - Security
    - Reliability
    - Maintainability
    - Transparency
- Security is not about looking secure. Security is about **being secure**. Don’t implement pointless ceremony—implement real defenses.

---

## **4. No Lazy MVP Excuses**

- Complexity is not an excuse to take shortcuts.
- This project was chosen precisely because it shortens the path to MVP—**so we have no margin left to allow laziness**.
- We would rather take longer to implement properly than ship a “fast but broken” version.

---

## **5. Professionalism & Open Source Standards**

- This project is open source. The codebase must be something we are proud to sign our names to.
- Code, comments, and commit history must be **professional, clear, and polished**.
- Every public interface (CLI, API, config) is part of the product contract. Treat it with care.

---

## **6. Documentation & Work Planning**

- Every step must be documented as if you could **die tomorrow and someone else must continue seamlessly**.
- This includes:
    - Detailed task planner / issue tracker
    - Explicit design decisions recorded in docs/ or issues
    - Comments that explain *why*, not *what*
- Work should proceed with a **detailed checklist**:
    - Define → Implement → Verify → Document → Mark complete
- Documentation is as much a part of the deliverable as the code.

---

## **7. Code Quality Requirements**

- Code must be **organized, consistent, and modular**.
- Tests are **mandatory** for anything critical (VM lifecycle, task execution, agent API).
- Comments must be **informative and minimal**: explain reasoning, not boilerplate.
- Do not write for “it compiles”; write for **robustness and longevity**.

---

## **8. Review & Enforcement**

- Any shortcut, placeholder, or lazy implementation is a **fundamental violation of this doctrine**.
- Code that violates doctrine must be rejected, no exceptions.
- The doctrine itself may evolve, but only through explicit, documented discussion.

---

## **9. Guiding Mental Model**

- *“This is not a prototype. This is not a demo. This is not just for us. This is the real thing, from day one.”*
- The end state is always:
    - Solid,
    - Reliable,
    - Secure,
    - Professional,
    - Open source–ready.

---

### **✅ TL;DR**

If you write it, it must:

- Be **production-grade**.
- Be **secure in reality, not appearance**.
- Be **owned fully, as if no one else exists**.
- Be **documented and accountable**.
- Be **worthy of open source release**.

---

# Viper: The Full Architecture Blueprint

---

## **1️⃣ CLI Tree (High-Level)**

```
viper
├── vms
│   ├── create <name> [--vmm cloudhypervisor/firecracker] [--contexts N] [--gpu]   # create new microVM
│   ├── list                 # show all managed VMs + status
│   └── destroy <name>       # remove VM
│
├── tasks
│   ├── submit <vm> <task.json>  # submit automation task to VM agent
│   ├── logs <vm> <task-id>      # task logs
│   └── screenshots <vm> <task-id> # list screenshots
│
├── browsers
│   └── spawn <vm> <context-id>  # spawn browser contexts inside VM
│
├── profiles
│   └── attach <vm> <context-id> <profile.json>  # inject preloaded browser profile
└── debug
    ├── system                     # Nomad + VM health + agent diagnostics
    ├── network                    # connectivity, proxy, BGP
    └── agent <vm>                 # debug agent inside VM
```

✅ **Design notes:**
- Each VM = 1 agent binary + multiple browser contexts
- Tasks are submitted to agent via HTTP API
- `vms create` registers Nomad job spec (HCL parsed via API)
- CLI uses Nomad API directly for VM management
- Thin CLI abstracts orchestration, fetches logs/screenshots from agent

---

## **2️⃣ Architecture Diagram**

```
┌───────────────────────────────┐
│         Viper CLI             │
│  Nomad API + Agent HTTP       │
└─────────────┬─────────────────┘
              │ Nomad API
              ▼
┌───────────────────────────────┐
│        Nomad Scheduler        │
│ - Job registration/scheduling │
│ - Resource & GPU constraints  │
│ - Multi-host orchestration    │
│ - Health checks / restart     │
└─────────────┬─────────────────┘
              │ Schedules VM jobs
              ▼
┌───────────────────────────────┐
│   Cloud Hypervisor VM         │
│ - Minimal rootfs (agent PID 1)│
│ - GPU passthrough (optional)  │
│ - Network: host IP accessible │
└─────────────┬─────────────────┘
              │ Runs agent on :8080
              ▼
┌───────────────────────────────┐
│       Agent (Go + Gin)        │
│ - chromedp browser contexts   │
│ - Task queue / execution      │
│ - Logs, screenshots storage   │
│ - Profile injection           │
└─────────────┬─────────────────┘
              │ HTTP endpoints
              ▼
┌───────────────────────────────┐
│   Browser Contexts (chromedp) │
│ - Isolated automation sessions│
│ - Navigate, script, capture   │
└───────────────────────────────┘
```

---

## **3️⃣ RootFS / Build Strategy**

- Use **Packer** for reproducible minimal rootfs (Alpine base + Chromium + agent)
- Preload GPU drivers if enabled
- Nomad artifacts pull rootfs image
- Fast boot via tmpfs/snapshots

**Commands:**
```
viper rootfs build cloudhypervisor --gpu
viper rootfs release cloudhypervisor
```

- Builds produce .qcow2 for Cloud Hypervisor
- Hashes ensure integrity in Nomad store

---

## **4️⃣ Task Flow**

1. `viper vms create my-vm` → registers Nomad job (boots VM + agent)
2. `viper browsers spawn my-vm ctx-1` → POST /spawn/ctx-1 (new chromedp context)
3. `viper profiles attach my-vm ctx-1 profile.json` → POST /profile/ctx-1 (inject UA, localStorage, cookies)
4. `viper tasks submit my-vm task.json` → POST /task (navigate URL, capture screenshot, log)
5. `viper tasks logs my-vm task-123` → GET /logs/task-123 (stdout)
6. `viper tasks screenshots my-vm task-123` → GET /screenshots/task-123 (list PNGs)

**Notes:**
- Tasks store in `/var/viper/tasks/<vm>/<task-id>/` (logs, screenshots, metadata.json)
- Contexts keyed by ID (spawned separately from VM)
- Multi-task: spawn multiple contexts, submit to VM (agent routes to free context)

---

## **5️⃣ Scaling & Robustness**

- **Horizontal:** Nomad scales VMs/agents automatically
- **Vertical:** Multiple contexts per VM (GPU for rendering)
- **Health:** Nomad checks + agent /health endpoint
- **Recovery:** Auto-restart failed tasks/contexts/VMs
- **Timeouts:** 60s default per task, configurable
- **Storage:** Persistent per-VM task dirs; optional S3 offload

---

## **6️⃣ Why This Feels “High-Class”**

- **Direct Nomad API:** No shell exec, clean Go integration
- **Drop-in CLI:** Matches existing structure, upgraded internals
- **Agent Alignment:** HTTP endpoints match CLI commands 1:1
- **Production-Ready:** Logging, screenshots, profiles out-of-box
- **Extensible:** Add gRPC, more VMMs, advanced chromedp actions
- **Developer-Friendly:** Reproducible builds, clear task isolation

---

💡 **TL;DR:**

Viper = **Nomad-orchestrated microVMs + chromedp agent + robust CLI**

- VM boot → agent API → isolated browser tasks
- GPU-ready, multi-context, profile injection
- Minimal rootfs → sub-second boots
- CLI handles full lifecycle + artifacts

---

## **1️⃣ Nomad Job Spec Example (viper-vm.nomad.hcl)**

```
job "viper-vm" {
  datacenters = ["dc1"]
  type        = "service"

  group "vm-group" {
    count = 1

    task "cloudhypervisor-vm" {
      driver = "exec"  # Or docker/raw_exec for agent

      config {
        command = "/agent"
        args    = ["--listen=:8080"]
      }

      resources {
        cpu    = 2000
        memory = 2048
        network {
          mbits = 100
          port "http" {
            static = 8080
          }
        }
      }

      # Artifact for rootfs/agent binary
      artifacts {
        source = "https://your-store/rootfs.qcow2"
      }
    }
  }
}
```

**Notes:**
- Use `libvirt` driver for full VMM (Cloud Hypervisor integration)
- Expose port 8080 for CLI → agent HTTP
- GPU: Add device passthrough via Nomad constraints

---

## **2️⃣ Agent Implementation (Go + Gin + chromedp)**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/chromedp/chromedp"
    "github.com/gin-gonic/gin"
)

type Task struct {
    ID     string `json:"id"`
    VMID   string `json:"vm_id"`
    URL    string `json:"url"`
    Script string `json:"script,omitempty"`
}

type Profile struct {
    ID           string                       `json:"id"`
    Cookies      []map[string]string          `json:"cookies"`
    LocalStorage map[string]map[string]string `json:"localStorage"`
    UserAgent    string                       `json:"userAgent"`
}

type Agent struct {
    mu       sync.Mutex
    contexts map[string]context.Context
    cancels  map[string]context.CancelFunc
    taskDir  string
}

func NewAgent() *Agent {
    return &Agent{
        contexts: make(map[string]context.Context),
        cancels:  make(map[string]context.CancelFunc),
        taskDir:  "/var/viper/tasks",
    }
}

// Spawn a new browser context
func (a *Agent) SpawnContext(id string) {
    ctx, cancel := chromedp.NewContext(context.Background())
    a.mu.Lock()
    a.contexts[id] = ctx
    a.cancels[id] = cancel
    a.mu.Unlock()
    log.Printf("Spawned context %s", id)
}

// Run a task in a context (assumes context ID matches VMID for simplicity; enhance for multi-context routing)
func (a *Agent) RunTask(t Task) error {
    a.mu.Lock()
    ctx, ok := a.contexts[t.VMID]
    a.mu.Unlock()
    if !ok {
        return fmt.Errorf("context %s not found", t.VMID)
    }

    logDir := filepath.Join(a.taskDir, t.VMID, t.ID)
    os.MkdirAll(filepath.Join(logDir, "screenshots"), 0755)

    logFile, _ := os.Create(filepath.Join(logDir, "stdout.log"))
    defer logFile.Close()

    var screenshot []byte
    timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()

    err := chromedp.Run(timeoutCtx,
        chromedp.Navigate(t.URL),
        chromedp.CaptureScreenshot(&screenshot),
    )
    if err != nil {
        return err
    }

    ioutil.WriteFile(filepath.Join(logDir, "screenshots", "1.png"), screenshot, 0644)

    meta := map[string]string{"url": t.URL, "status": "done"}
    data, _ := json.Marshal(meta)
    ioutil.WriteFile(filepath.Join(logDir, "metadata.json"), data, 0644)

    logFile.WriteString(fmt.Sprintf("Task %s completed\n", t.ID))
    return nil
}

// Attach profile to context
func (a *Agent) AttachProfile(ctxID string, p Profile) error {
    a.mu.Lock()
    ctx, ok := a.contexts[ctxID]
    a.mu.Unlock()
    if !ok {
        return fmt.Errorf("context %s not found", ctxID)
    }

    if p.UserAgent != "" {
        chromedp.Run(ctx, chromedp.Emulate(chromedp.UserAgent(p.UserAgent)))
    }

    for domain, kv := range p.LocalStorage {
        for key, value := range kv {
            expr := fmt.Sprintf(`localStorage.setItem("%s", "%s")`, key, value)
            chromedp.Run(ctx, chromedp.Evaluate(expr, nil))
            log.Printf("Injected localStorage %s=%s for %s", key, value, domain)
        }
    }

    return nil
}

func main() {
    agent := NewAgent()
    r := gin.Default()

    // POST /spawn/:id - Spawn browser context
    r.POST("/spawn/:id", func(c *gin.Context) {
        id := c.Param("id")
        agent.SpawnContext(id)
        c.JSON(http.StatusOK, gin.H{"status": "spawned", "id": id})
    })

    // POST /task - Submit and run task asynchronously
    r.POST("/task", func(c *gin.Context) {
        var t Task
        if err := c.ShouldBindJSON(&t); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        go func() {
            if err := agent.RunTask(t); err != nil {
                log.Println("Task error:", err)
            }
        }()
        c.JSON(http.StatusOK, gin.H{"status": "started", "id": t.ID})
    })

    // GET /logs/:taskid - Fetch task logs
    r.GET("/logs/:taskid", func(c *gin.Context) {
        taskID := c.Param("taskid")
        logPath := filepath.Join(agent.taskDir, "default-vm", taskID, "stdout.log")
        data, err := ioutil.ReadFile(logPath)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
            return
        }
        c.Data(http.StatusOK, "text/plain", data)
    })

    // GET /screenshots/:taskid - List screenshot files
    r.GET("/screenshots/:taskid", func(c *gin.Context) {
        taskID := c.Param("taskid")
        shotsPath := filepath.Join(agent.taskDir, "default-vm", taskID, "screenshots")
        files, _ := ioutil.ReadDir(shotsPath)
        names := []string{}
        for _, f := range files {
            names = append(names, f.Name())
        }
        c.JSON(http.StatusOK, names)
    })

    // POST /profile/:ctxid - Attach profile to context
    r.POST("/profile/:ctxid", func(c *gin.Context) {
        ctxID := c.Param("ctxid")
        var p Profile
        if err := c.ShouldBindJSON(&p); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        if err := agent.AttachProfile(ctxID, p); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"status": "profile attached", "id": p.ID})
    })

    // GET /health - For Nomad checks
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "healthy"})
    })

    log.Println("Agent listening on :8080")
    r.Run(":8080")
}
```

**Notes:**
- Contexts managed by ID (spawn first, then attach/run tasks)
- Tasks route to VMID-keyed context (enhance for ctxID routing in production)
- Storage: `/var/viper/tasks/<vm>/<task-id>/` for isolation
- Expand: Add cookie injection, JS evaluation, error handling

---

## **3️⃣ Packer Template (Minimal RootFS)**

```json
{
  "builders": [
    {
      "type": "qemu",
      "iso_url": "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/alpine-minirootfs-3.21.0-x86_64.tar.gz",
      "iso_checksum": "auto",
      "output_directory": "out/cloudhypervisor-rootfs",
      "format": "qcow2",
      "accelerator": "kvm",
      "disk_size": 1024,
      "headless": true,
      "ssh_username": "root",
      "ssh_password": "root",
      "shutdown_command": "poweroff"
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "inline": [
        "apk add --no-cache chromium bash curl",
        "mkdir -p /agent",
        "cp /tmp/agent /agent/agent",
        "chmod +x /agent/agent"
      ]
    }
  ]
}
```

**Usage:**
```
packer build cloudhypervisor.json
```

- Builds Alpine + Chromium + agent binary
- Output: rootfs.qcow2 for Nomad/libvirt

---

## **4️⃣ Viper CLI Implementation (Go + Nomad API + Cobra)**

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "time"

    nomad "github.com/hashicorp/nomad/api"
    "github.com/spf13/cobra"
)

type Task struct {
    ID     string `json:"id"`
    VMID   string `json:"vm_id"`
    URL    string `json:"url"`
    Script string `json:"script,omitempty"`
}

type Profile struct {
    ID           string                       `json:"id"`
    Cookies      []map[string]string          `json:"cookies"`
    LocalStorage map[string]map[string]string `json:"localStorage"`
    UserAgent    string                       `json:"userAgent"`
}

var nomadClient *nomad.Client

func main() {
    // Initialize Nomad API client
    conf := nomad.DefaultConfig()
    client, err := nomad.NewClient(conf)
    if err != nil {
        log.Fatalf("Error creating Nomad client: %v", err)
    }
    nomadClient = client

    rootCmd := &cobra.Command{
        Use:   "viper",
        Short: "Viper: microVM + browser automation CLI",
    }

    rootCmd.AddCommand(vmCmd())
    rootCmd.AddCommand(taskCmd())
    rootCmd.AddCommand(browserCmd())
    rootCmd.AddCommand(profileCmd())

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func vmCmd() *cobra.Command {
    vm := &cobra.Command{
        Use:   "vms",
        Short: "Manage microVMs",
    }

    // create
    create := &cobra.Command{
        Use:   "create [name]",
        Short: "Create a new VM",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            name := args[0]
            // For MVP: assume job file exists under jobs/<name>.nomad.hcl
            // Note: Parse HCL to nomad.Job (use hashicorp/nomad/hcl or similar; placeholder uses raw)
            jobFile := fmt.Sprintf("jobs/%s.nomad.hcl", name)
            data, err := ioutil.ReadFile(jobFile)
            if err != nil {
                log.Fatalf("failed to read job file: %v", err)
            }

            // TODO: Proper HCL parsing to nomad.Job
            job := &nomad.Job{
                Name:        name,
                Type:        nomad.JobTypeService,
                Datacenters: []string{"dc1"},
            }
            // Simulate: in prod, parse HCL data to populate job

            _, _, err = nomadClient.Jobs().Register(job, nil)
            if err != nil {
                log.Fatalf("failed to register job: %v", err)
            }
            fmt.Printf("VM %s created\n", name)
        },
    }
    vm.AddCommand(create)

    // list
    list := &cobra.Command{
        Use:   "list",
        Short: "List running VMs",
        Run: func(cmd *cobra.Command, args []string) {
            jobs, _, err := nomadClient.Jobs().List(nil)
            if err != nil {
                log.Fatal(err)
            }
            for _, j := range jobs {
                fmt.Printf("%s\t%s\t%s\n", j.ID, j.Type, j.Status)
            }
        },
    }
    vm.AddCommand(list)

    // destroy
    destroy := &cobra.Command{
        Use:   "destroy [name]",
        Short: "Destroy VM",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            name := args[0]
            _, _, err := nomadClient.Jobs().Deregister(name, false, nil)
            if err != nil {
                log.Fatal(err)
            }
            fmt.Printf("VM %s destroyed\n", name)
        },
    }
    vm.AddCommand(destroy)

    return vm
}

func taskCmd() *cobra.Command {
    task := &cobra.Command{
        Use:   "tasks",
        Short: "Manage tasks inside microVMs",
    }

    // submit task
    submit := &cobra.Command{
        Use:   "submit [vm] [task.json]",
        Short: "Submit a task to a VM agent",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            vm := args[0]
            taskFile := args[1]

            data, err := ioutil.ReadFile(taskFile)
            if err != nil {
                log.Fatal(err)
            }

            var t Task
            if err := json.Unmarshal(data, &t); err != nil {
                log.Fatal(err)
            }
            t.VMID = vm

            agentURL := fmt.Sprintf("http://%s:8080/task", vm)
            body, _ := json.Marshal(t)
            resp, err := http.Post(agentURL, "application/json", bytes.NewReader(body))
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()
            fmt.Printf("Task submitted: %s\n", t.ID)
        },
    }
    task.AddCommand(submit)

    // logs
    logs := &cobra.Command{
        Use:   "logs [vm] [task-id]",
        Short: "Fetch task logs",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            vm := args[0]
            taskID := args[1]
            agentURL := fmt.Sprintf("http://%s:8080/logs/%s", vm, taskID)

            resp, err := http.Get(agentURL)
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()

            data, _ := ioutil.ReadAll(resp.Body)
            fmt.Println(string(data))
        },
    }
    task.AddCommand(logs)

    // screenshots
    screenshots := &cobra.Command{
        Use:   "screenshots [vm] [task-id]",
        Short: "List screenshots for a task",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            vm := args[0]
            taskID := args[1]
            agentURL := fmt.Sprintf("http://%s:8080/screenshots/%s", vm, taskID)

            resp, err := http.Get(agentURL)
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()

            data, _ := ioutil.ReadAll(resp.Body)
            fmt.Println(string(data))
        },
    }
    task.AddCommand(screenshots)

    return task
}

func browserCmd() *cobra.Command {
    browsers := &cobra.Command{
        Use:   "browsers",
        Short: "Manage browser contexts inside VMs",
    }

    spawn := &cobra.Command{
        Use:   "spawn [vm] [context-id]",
        Short: "Spawn a new browser context inside VM",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            vm := args[0]
            ctxID := args[1]
            agentURL := fmt.Sprintf("http://%s:8080/spawn/%s", vm, ctxID)
            resp, err := http.Post(agentURL, "application/json", nil)
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()
            fmt.Printf("Spawned browser context %s in VM %s\n", ctxID, vm)
        },
    }
    browsers.AddCommand(spawn)

    return browsers
}

func profileCmd() *cobra.Command {
    profiles := &cobra.Command{
        Use:   "profiles",
        Short: "Manage browser profiles",
    }

    attach := &cobra.Command{
        Use:   "attach [vm] [context-id] [profile.json]",
        Short: "Attach profile to browser context",
        Args:  cobra.ExactArgs(3),
        Run: func(cmd *cobra.Command, args []string) {
            vm := args[0]
            ctxID := args[1]
            profileFile := args[2]

            data, err := ioutil.ReadFile(profileFile)
            if err != nil {
                log.Fatal(err)
            }

            var p Profile
            if err := json.Unmarshal(data, &p); err != nil {
                log.Fatal(err)
            }

            agentURL := fmt.Sprintf("http://%s:8080/profile/%s", vm, ctxID)
            body, _ := json.Marshal(p)
            resp, err := http.Post(agentURL, "application/json", bytes.NewReader(body))
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()
            fmt.Printf("Attached profile %s to context %s in VM %s\n", p.ID, ctxID, vm)
        },
    }
    profiles.AddCommand(attach)

    return profiles
}
```

**Notes:**
- **Nomad Integration:** Direct API for job register/list/deregister (fix HCL parsing with `github.com/hashicorp/nomad/helper/hcl2decode` for prod)
- **Agent Calls:** HTTP to VM IP:8080; assumes VM exposes network port
- **Enhancements:** Add flags for VMM/GPU/contexts in create; task status/cancel; profile list/export
- **Build:** `go mod init viper; go get github.com/hashicorp/nomad/api github.com/spf13/cobra; go build -o viper`

---

### **✅ Minimal MVP-Ready Path**

1. **RootFS:** Packer build → upload to store
2. **CLI:** `go build` → `./viper vms create my-vm` (registers job)
3. **Agent:** Deploy binary to rootfs; Nomad boots VM → agent :8080
4. **Workflow:** Spawn context → attach profile → submit task → fetch logs/screenshots
- **Robust:** Error handling, timeouts, async tasks
- **Organized:** Clear separation (CLI/Nomad/Agent); easy to extend/debug
- **Productive:** Drop-in replacement; consistent APIs; reproducible




***If you are beginning your work now, REMEMBER TO: (if just beginning,) CREATE A FILE TO KEEP DETAILED TRACK OF YOUR WORK AND REMEMBER TO UPDATE IT REGULARLY. IF YOU ARE CONTINUING, REMEMBER TO CHECK THE FILE TO CONTINUE WHERE YOU LEFT OFF.***

***The work tracking file should be detailed, methodological, organised and detailed enough for the remaining of the development to be a clear direction forward, and updated across all aspects immediately if changes were to be made. It should also align closely with git commits and updates, so that all work is tracked and versionable. Take initiative to ensure that we are always in the most productive state possible***

***Remember, you are treated as a co-founder in this startup and it comes with the responsibility of it, and the initiative and standards you are expected to uphold***

**And remember not to commit this CLAUDE.md file it is only for local use**
