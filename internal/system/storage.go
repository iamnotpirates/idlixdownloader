package system

import (
	"math"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

func GetDiskSpace(dirPath string) (*models.StorageInfo, error) {
	if dirPath == "" {
		dirPath = "."
	}
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	vol := filepath.VolumeName(absPath)
	if vol != "" {
		vol += `\`
	} else {
		vol = absPath
	}

	var freeBytes, totalBytes uint64

	if runtime.GOOS == "windows" {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
		var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
		pathPtr, err := syscall.UTF16PtrFromString(vol)
		if err == nil {
			r1, _, _ := getDiskFreeSpaceEx.Call(
				uintptr(unsafe.Pointer(pathPtr)),
				uintptr(unsafe.Pointer(&freeBytesAvailable)),
				uintptr(unsafe.Pointer(&totalNumberOfBytes)),
				uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
			)
			if r1 != 0 {
				freeBytes = freeBytesAvailable
				totalBytes = totalNumberOfBytes
			}
		}
	}

	if totalBytes == 0 {
		return &models.StorageInfo{
			Path: vol,
		}, nil
	}

	usedBytes := totalBytes - freeBytes
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	freeGB := float64(freeBytes) / (1024 * 1024 * 1024)
	usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
	percentUsed := (float64(usedBytes) / float64(totalBytes)) * 100.0

	return &models.StorageInfo{
		Path:        vol,
		TotalBytes:  totalBytes,
		FreeBytes:   freeBytes,
		UsedBytes:   usedBytes,
		TotalGB:     math.Round(totalGB*100) / 100,
		FreeGB:      math.Round(freeGB*100) / 100,
		UsedGB:      math.Round(usedGB*100) / 100,
		PercentUsed: math.Round(percentUsed*10) / 10,
	}, nil
}
