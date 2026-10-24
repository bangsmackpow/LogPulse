//go:build windows

package discovery

import (
	"os"
	"path/filepath"
)

func getTier1Paths() []string {
	var paths []string

	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		paths = append(paths, localApp)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		paths = append(paths, appData)
	}
	if progData := os.Getenv("PROGRAMDATA"); progData != "" {
		paths = append(paths, progData)
	}

	winDir := os.Getenv("WINDIR")
	if winDir == "" {
		winDir = "C:\\Windows"
	}
	paths = append(paths, filepath.Join(winDir, "Logs"))
	paths = append(paths, filepath.Join(winDir, "System32", "winevt", "Logs"))

	return paths
}
