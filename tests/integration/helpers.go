package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// getProjectRoot walks up the directory tree to find the project root
func getProjectRoot(t *testing.T) string {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Walk up the directory tree to find the project root
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		wd = parent
	}
}