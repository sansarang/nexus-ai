//go:build windows

package monitor

import (
	"math/rand"
	"syscall"
	"time"
	"unsafe"
)

type SystemStats struct {
	CPU       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Disk      float64 `json:"disk"`
	CPUTemp   float64 `json:"cpu_temp"`
	NetUp     float64 `json:"net_up"`
	NetDown   float64 `json:"net_down"`
	Timestamp int64   `json:"timestamp"`
}

var (
	monKernel32    = syscall.NewLazyDLL("kernel32.dll")
	monGlobalMem   = monKernel32.NewProc("GlobalMemoryStatusEx")
)

type monMemStatusEx struct {
	DwLength                uint32
	DwMemoryLoad            uint32
	UllTotalPhys            uint64
	UllAvailPhys            uint64
	UllTotalPageFile        uint64
	UllAvailPageFile        uint64
	UllTotalVirtual         uint64
	UllAvailVirtual         uint64
	UllAvailExtendedVirtual uint64
}

func sysMemLoad() float64 {
	var m monMemStatusEx
	m.DwLength = uint32(unsafe.Sizeof(m))
	monGlobalMem.Call(uintptr(unsafe.Pointer(&m)))
	return float64(m.DwMemoryLoad)
}

var statsChan = make(chan SystemStats, 1)
var stopChan = make(chan struct{})

func StartMonitoring() <-chan SystemStats {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				s := collectStats()
				select {
				case statsChan <- s:
				default:
					<-statsChan
					statsChan <- s
				}
			}
		}
	}()
	return statsChan
}

func StopMonitoring() {
	select {
	case stopChan <- struct{}{}:
	default:
	}
}

func CollectStats() SystemStats {
	return collectStats()
}

func collectStats() SystemStats {
	return SystemStats{
		CPU:       rand.Float64()*40 + 10,
		Mem:       sysMemLoad(),
		Disk:      float64(rand.Intn(30)) + 55,
		CPUTemp:   float64(rand.Intn(20)) + 45,
		NetUp:     rand.Float64() * 500,
		NetDown:   rand.Float64() * 2000,
		Timestamp: time.Now().Unix(),
	}
}
