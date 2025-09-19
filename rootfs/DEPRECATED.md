# DEPRECATED: Packer ISO-based Build System

**This directory contains the old Packer-based VM build system and is now deprecated.**

## Why Deprecated?

Viper is a **browser automation framework**, not a cloud infrastructure stack. We've moved to a more focused approach:

1. **Docker-based builds** using `chromedp/headless-shell` as base image
2. **Extract initramfs** from Docker containers for Cloud Hypervisor
3. **Faster iteration** - builds in seconds, not minutes
4. **Better focus** - leverage existing optimized browser images instead of building OS from scratch

## New Approach

Use the new Docker-based image system:

```bash
# Build VM images the new way
make build-images

# This creates:
# - dist/viper-headless.qcow2 (Cloud Hypervisor ready)
# - dist/example-job.hcl (Nomad job template)
```

## Migration Path

If you were using Packer builds:

**Old:**
```bash
make rootfs-build  # 5+ minutes, complex dependencies
```

**New:**
```bash
make build-images  # <1 minute, Docker-based
```

## Files in This Directory

- `alpine-macos.pkr.hcl` - macOS ARM64 Packer template (deprecated)
- `alpine-x86_64.pkr.hcl` - x86_64 Packer template (deprecated)
- `http/setup.sh` - Alpine installation script (deprecated)

These files are kept for reference but should not be used for new development.

## See Instead

- `images/headless/` - New Docker-based build system
- `Makefile` - Updated build targets
- Main README for current build instructions