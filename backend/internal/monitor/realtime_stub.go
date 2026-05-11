//go:build !windows

package monitor

import "time"

type SystemStats struct {
	CPU       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Disk      float64 `json:"disk"`
	CPUTemp   float64 `json:"cpu_temp"`
	NetUp     float64 `json:"net_up"`
	NetDown   float64 `json:"net_down"`
	Timestamp int64   `json:"timestamp"`
}

func StartMonitoring() <-chan SystemStats {
	ch := make(chan SystemStats, 1)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ch <- SystemStats{CPU: 25, Mem: 60, Disk: 70, CPUTemp: 55, Timestamp: time.Now().Unix()}
		}
	}()
	return ch
}

func StopMonitoring() {}

func CollectStats() SystemStats {
	return SystemStats{CPU: 25, Mem: 60, Disk: 70, CPUTemp: 55, Timestamp: time.Now().Unix()}
}
