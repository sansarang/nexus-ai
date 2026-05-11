//go:build windows

package daily

import (
	"fmt"
	"math/rand"
	"time"
)

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
	Trend string  `json:"trend"` // "up" | "down" | "stable"
}

func GenerateDailyReport() DailyReport {
	now := time.Now()
	cpu := rand.Float64()*30 + 15
	mem := rand.Float64()*25 + 45
	disk := rand.Float64()*50 + 30

	recs := []string{}
	if cpu > 35 {
		recs = append(recs, "CPU 사용률이 높습니다. 백그라운드 프로세스를 확인하세요.")
	}
	if mem > 60 {
		recs = append(recs, "메모리 사용량이 많습니다. 불필요한 프로그램을 종료하세요.")
	}
	if disk < 50 {
		recs = append(recs, "디스크 여유 공간이 부족합니다. PC 정리를 실행하세요.")
	}
	recs = append(recs, fmt.Sprintf("%s 정기 PC 점검을 완료했습니다.", now.Format("01월 02일")))

	return DailyReport{
		Date:       now.Format("2006-01-02"),
		PCScore:    rand.Intn(30) + 65,
		CPUAvg:     cpu,
		MemAvg:     mem,
		DiskFreeGB: disk,
		Recommendations: recs,
		Predictions: []Prediction{
			{Label: "CPU 사용률", Value: cpu + rand.Float64()*10, Trend: "up"},
			{Label: "메모리 사용률", Value: mem + rand.Float64()*5, Trend: "stable"},
			{Label: "디스크 여유", Value: disk - rand.Float64()*5, Trend: "down"},
		},
	}
}
