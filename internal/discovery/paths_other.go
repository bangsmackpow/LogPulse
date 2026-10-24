//go:build !windows

package discovery

import (
	"os"
	"path/filepath"
)

func getTier1Paths() []string {
	var paths []string

	// Common Linux / macOS log paths
	paths = append(paths, "/var/log")

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Library", "Logs"))
		paths = append(paths, filepath.Join(home, ".local", "state"))
	}

	return paths
}
