// Created by cgo -godefs - DO NOT EDIT
// cgo.exe -godefs defs_diskio_windows.go

package diskio

type DiskPerformance = struct {
	BytesRead           [8]byte
	BytesWritten        [8]byte
	ReadTime            [8]byte
	WriteTime           [8]byte
	IdleTime            [8]byte
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           [8]byte
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
	Pad_cgo_0           [4]byte
}

type DiskManagementControlCode uint32

const (
	IoctlDiskPerformance    DiskManagementControlCode = 0x70020
	IoctlDiskPerformanceOff DiskManagementControlCode = 0x70060
)
