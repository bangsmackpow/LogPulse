//go:build !windows

package tailer

import "os"

func openSharedRead(filePath string) (*os.File, error) {
	return os.OpenFile(filePath, os.O_RDONLY, 0644)
}
