<p align="center">
  <img src="https://github.com/ccheshirecat/viper/blob/main/viper-github.png" alt="Viper - The Modular Engine for Modern Browser Automation"/>
</p>

<p align="center">
    <a href="https://github.com/ccheshirecat/viper/releases"><img src="https://img.shields.io/github/v/release/ccheshirecat/viper.svg" alt="Latest Release"></a>
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-blue.svg" alt="Go Version"></a>
    <a href="https://github.com/ccheshirecat/viper/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"></a>
    <a href="https://github.com/ccheshirecat/viper/graphs/contributors"><img src="https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat" alt="Contributions Welcome"></a>
</p>

---

**Viper** is a microVM-based browser automation framework that provides unparalleled session persistence, kernel-level security, and massive scalability for stateful browser tasks where stealth, reliability, and performance are paramount.

It is a general-purpose, extensible platform for orchestrating browser-based workloads, from large-scale data intelligence to complex, stateful automation.

## Why Viper?

Existing browser automation tools like Puppeteer and Playwright run as a single process on a host OS, leaving detectable fingerprints and lacking true isolation. Viper solves this at an architectural level.

| Feature                  | Puppeteer / Playwright | Viper                                     |
| ------------------------ | :--------------------: | :---------------------------------------: |
| **Sandboxing Model**     |    Process-level       | **True Kernel-Level via microVMs**        |
| **Session Persistence**  |  Cookies & Storage     | **Full System Snapshots** (VM state)      |
| **Security & Stealth**   |        Limited         | **Unparalleled Isolation & Cloaking**     |
| **Orchestration**        |        Manual          | **Built-in, Scalable (Nomad)**            |
| **Extensibility**        |        Library         | **First-Class Plugin Ecosystem**          |
| **Target Workloads**     |   Simple Scripting     | **Complex, Stateful, Long-Lived Tasks**   |

Viper isn't just another browser library; it's the **Kubernetes of browsers**—an orchestration platform for disposable, persistent, and secure browser-based identities.

## Core Features

- **MicroVM Isolation**: Every browser session runs in its own lightweight, fully isolated Alpine Linux microVM, providing a pristine, undetectable environment.
- **Plugin-Driven Workflows**: Extend Viper's core capabilities with specialized plugins for different domains (e.g., e-commerce, social media, data scraping).
- **Stateful Session Persistence**: Utilize hypervisor-level snapshots to save and resume the entire state of a VM, including browser sessions, cookies, and `localStorage`.
- **Scalable Orchestration**: Leverages Nomad to manage the lifecycle of hundreds or thousands of microVMs across a cluster of machines.
- **Production-Ready**: Built from day one with a "production-first" philosophy, emphasizing security, reliability, and professional-grade code.
- **Developer-Friendly**: A clean CLI, fast Docker-based VM image builds, and a powerful API for building custom solutions.

## Architecture Overview

Viper's architecture is built on a clear separation of concerns, providing a robust and scalable foundation.

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Viper CLI     │───▶│  Nomad Cluster   │───▶│   libvirt Host      │
│                 │    │                  │    │                     │
│ Plugin System:  │    │ microVM Jobs:    │    │ ┌─────────────────┐ │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ │   Alpine VM 1   │ │
│ │Casino Plugin│ │    │ │libvirt driver│ │    │ │ ┌─────────────┐ │ │
│ │Social Plugin│ │    │ │GPU Passthru │ │    │ │ │ viper-agent │ │ │
│ │Custom Plugin│ │    │ │Health Checks │ │    │ │ │ + Chromium  │ │ │
│ └─────────────┘ │    │ └──────────────┘ │    │ │ │   :8080     │ │ │
└─────────────────┘    └──────────────────┘    │ │ └─────────────┘ │ │
                                               │ └─────────────────┘ │
         Plugin Actions                        │                     │
         (claim_bonus, check_balance, etc.)    │ ┌─────────────────┐ │
                      │                       │ │   Alpine VM N   │ │
                      ▼                       │ │ ┌─────────────┐ │ │
┌───────────────────────────────────────────────┤ │ │ viper-agent │ │ │
│          HTTP API Calls to Agent            │ │ │ │ + Chromium  │ │ │
│ • /spawn/:context-id                        │ │ │ │   :808N     │ │ │
│ • /task (automation)                        │ │ │ └─────────────┘ │ │
│ • /profile/:context (inject sessions)       │ │ └─────────────────┘ │
│ • /executeJS (custom browser actions)       │ └─────────────────────┘
└─────────────────────────────────────────────┘

Hypervisor Stack:
┌─────────────────┐  ┌─────────────────┐
│   Development   │  │   Production    │
│ macOS + QEMU    │  │ Linux + CHV*    │
│ + libvirt       │  │ + libvirt       │
└─────────────────┘  └─────────────────┘
*CHV = Cloud Hypervisor
```



- **The CLI & Plugins**: The user's entry point for managing workloads and executing specialized automation logic.
- **Nomad Cluster**: The orchestration brain that schedules, monitors, and manages the lifecycle of microVMs.
- **libvirt Host**: The hypervisor layer that creates and runs the actual QEMU (dev) or Cloud Hypervisor (prod) VMs.
- **Alpine VM**: A minimal, secure environment running the `viper-agent` and a real Chromium browser.

## Getting Started

**Prerequisites:**
- Go 1.21+
- Docker
- Nomad with [Cloud Hypervisor driver](https://github.com/ccheshirecat/nomad-driver-ch)
- `qemu-utils` and `e2fsprogs` for image conversion

**1. Build the Binaries & VM Images:**

Viper uses a **Docker-based build pipeline** for speed and reliability. We build on `chromedp/headless-shell` and extract initramfs for Cloud Hypervisor.

```bash
# Clone the repository
git clone https://github.com/ccheshirecat/viper.git
cd viper

# Build CLI, Agent, and VM images in one command
make ci-build

# Or build step by step:
make build        # Build CLI and Agent binaries
make build-images # Create VM images from Docker

### 2. Start Nomad

Run a Nomad agent in development mode, ensuring the libvirt driver is configured.

```bash
# (See documentation for full libvirt setup)
nomad agent -dev
```

### 3. Launch Your First MicroVM

Use the Viper CLI to create your first isolated browser environment.

```bash
# Create a new microVM named 'my-first-vm'
./bin/viper vms create my-first-vm --memory 2048 --cpus 2

# List running VMs
./bin/viper vms list
```

### Example: Plugin-Based Automation
The true power of Viper is unlocked through its plugin system, which enables complex, long-lived automation workloads.
Imagine a specialized plugin for stake.com.

```bash
# 1. Install the specialized casino plugin
viper plugins install github.com/viper-plugins/casino-stake

# 2. Create a "workload pool" of 10 persistent microVMs,
#    each with its own browser session and pre-loaded account profile.
viper workloads create my-stake-farm \
  --plugin casino-stake \
  --profile profiles/stake-accounts.json \
  --count 10

# 3. Execute plugin-specific actions across the entire farm.
#    The plugin handles the complex browser interactions inside each isolated VM.
viper workloads action my-stake-farm claim_daily_bonus
viper workloads action my-stake-farm check_balance

# 4. Monitor the status of your workload pool
viper workloads status my-stake-farm
```

## Contributing

Contributions are welcome! We are looking for developers who are passionate about building the future of robust, secure automation. Please see our (upcoming) CONTRIBUTING.md guide for more details on how to get involved.

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for details.
