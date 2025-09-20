package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVMImageExists verifies that the built VM image exists and is valid
// This test requires Docker build artifacts to exist first
func TestVMImageExists(t *testing.T) {
	projectRoot := getProjectRoot(t)
	imagePath := filepath.Join(projectRoot, "dist", "viper-headless.qcow2")

	// Check if image exists
	_, err := os.Stat(imagePath)
	require.NoError(t, err, "VM image should exist at %s", imagePath)

	// Check file size (should be reasonable for a headless Chrome image)
	fileInfo, err := os.Stat(imagePath)
	require.NoError(t, err)

	// Image should be at least 50MB (compressed) but less than 2GB
	assert.Greater(t, fileInfo.Size(), int64(50*1024*1024), "Image too small")
	assert.Less(t, fileInfo.Size(), int64(2*1024*1024*1024), "Image too large")

	t.Logf("✅ VM image exists: %s (size: %d bytes)", imagePath, fileInfo.Size())
}

// TestNomadJobTemplate verifies the generated Nomad job template
// This test requires the job template to be generated first
func TestNomadJobTemplate(t *testing.T) {
	projectRoot := getProjectRoot(t)
	jobPath := filepath.Join(projectRoot, "dist", "example-job.hcl")

	// Check if job template exists
	_, err := os.Stat(jobPath)
	require.NoError(t, err, "Nomad job template should exist at %s", jobPath)

	// Read and basic validate content
	content, err := os.ReadFile(jobPath)
	require.NoError(t, err)

	jobContent := string(content)
	assert.Contains(t, jobContent, `driver = "ch"`, "Should use Cloud Hypervisor driver")
	assert.Contains(t, jobContent, "vmlinuz", "Should reference kernel")
	assert.Contains(t, jobContent, "viper-initramfs.gz", "Should reference initramfs")
	assert.Contains(t, jobContent, "viper-headless.qcow2", "Should reference rootfs disk")
	assert.Contains(t, jobContent, "viper-agent", "Should reference our agent")

	t.Logf("✅ Nomad job template generated and contains expected content")
}