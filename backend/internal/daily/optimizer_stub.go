//go:build !windows

package daily

import "time"

type DailyReport struct {
	Date            string       `json:"date"`
	PCScore         int          `json:"pc_score"`
	CPUAvg          float64      `json:"cpu_avg"`
	MemAvg          float64      `json:"mem_avg"`
	DiskFreeGB      float64      `json:"disk_free_gb"`
	Recommendations []string     `json:"recommendations"`
	Predictions     []Prediction `json:"predictions"`
}

type Prediction struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Trend string  `json:"trend"`
}

func GenerateDailyReport() DailyReport {
	return DailyReport{
		Date:       time.Now().Format("2006-01-02"),
		PCScore:    78,
		CPUAvg:     22.5,
		MemAvg:     58.0,
		DiskFreeGB: 45.0,
		Recommendations: []string{"PC 상태가 양호합니다."},
		Predictions: []Prediction{
			{Label: "CPU 사용률", Value: 25, Trend: "stable"},
			{Label: "메모리 사용률", Value: 60, Trend: "up"},
			{Label: "디스크 여유", Value: 45, Trend: "down"},
		},
	}
}
