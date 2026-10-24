//go:build windows

package tailer

import (
	"os"

	"golang.org/x/sys/windows"
)

func openSharedRead(filePath string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		p,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		// Fallback to standard os.OpenFile
		return os.OpenFile(filePath, os.O_RDONLY, 0644)
	}

	return os.NewFile(uintptr(handle), filePath), nil
}
