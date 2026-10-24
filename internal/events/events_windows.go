//go:build windows

package events

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modwevtapi     = syscall.NewLazyDLL("wevtapi.dll")
	procEvtQuery   = modwevtapi.NewProc("EvtQuery")
	procEvtNext    = modwevtapi.NewProc("EvtNext")
	procEvtRender  = modwevtapi.NewProc("EvtRender")
	procEvtClose   = modwevtapi.NewProc("EvtClose")
)

const (
	EvtQueryChannelPath      = 0x1
	EvtQueryFilePath         = 0x2
	EvtQueryReverseDirection = 0x200

	EvtRenderEventValues = 0
	EvtRenderEventXml    = 1
)

type WindowsEventReader struct {
	offline offlineEventReader
}

func NewEventReader() EventReader {
	return &WindowsEventReader{}
}

func (w *WindowsEventReader) ReadRecent(channel string, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	chanPtr, err := syscall.UTF16PtrFromString(channel)
	if err != nil {
		return nil, err
	}
	queryPtr, err := syscall.UTF16PtrFromString("*")
	if err != nil {
		return nil, err
	}

	flags := uintptr(EvtQueryChannelPath | EvtQueryReverseDirection)
	handle, _, callErr := procEvtQuery.Call(0, uintptr(unsafe.Pointer(chanPtr)), uintptr(unsafe.Pointer(queryPtr)), flags)
	if handle == 0 {
		return nil, fmt.Errorf("EvtQuery failed for channel %s: %v", channel, callErr)
	}
	defer procEvtClose.Call(handle)

	return w.readEvents(handle, limit)
}

func (w *WindowsEventReader) ReadFile(evtxPath string, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	pathPtr, err := syscall.UTF16PtrFromString(evtxPath)
	if err != nil {
		return nil, err
	}
	queryPtr, err := syscall.UTF16PtrFromString("*")
	if err != nil {
		return nil, err
	}

	flags := uintptr(EvtQueryFilePath | EvtQueryReverseDirection)
	handle, _, _ := procEvtQuery.Call(0, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(queryPtr)), flags)
	if handle == 0 {
		// Fallback to offline XML reader
		return w.offline.ReadFile(evtxPath, limit)
	}
	defer procEvtClose.Call(handle)

	return w.readEvents(handle, limit)
}

func (w *WindowsEventReader) readEvents(queryHandle uintptr, limit int) ([]EventRecord, error) {
	var records []EventRecord
	eventHandles := make([]uintptr, limit)
	var returned uint32

	ret, _, _ := procEvtNext.Call(
		queryHandle,
		uintptr(limit),
		uintptr(unsafe.Pointer(&eventHandles[0])),
		1000, // 1 sec timeout
		0,
		uintptr(unsafe.Pointer(&returned)),
	)

	if ret == 0 && returned == 0 {
		return records, nil
	}

	buf := make([]uint16, 16384)
	var bufUsed uint32
	var propCount uint32

	for i := 0; i < int(returned); i++ {
		evHandle := eventHandles[i]
		if evHandle == 0 {
			continue
		}

		rRet, _, _ := procEvtRender.Call(
			0,
			evHandle,
			EvtRenderEventXml,
			uintptr(len(buf)*2),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&bufUsed)),
			uintptr(unsafe.Pointer(&propCount)),
		)

		procEvtClose.Call(evHandle)

		if rRet != 0 {
			xmlStr := syscall.UTF16ToString(buf[:bufUsed/2])
			if rec, err := ParseEventXML([]byte(xmlStr)); err == nil {
				records = append(records, *rec)
			}
		}
	}

	return records, nil
}
