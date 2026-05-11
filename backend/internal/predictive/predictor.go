//go:build windows

package predictive

import (
	"fmt"
	"math"
)

type Prediction struct {
	Type        string  `json:"type"`
	Probability float64 `json:"probability"`
	TimeFrame   string  `json:"time_frame"`
	Advice      string  `json:"advice"`
	AutoAction  string  `json:"auto_action"`
}

func PredictTrend(values []float64) float64 {
	n := float64(len(values))
	if n < 2 {
		return 0
	}
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func GeneratePredictions(cpuTemps, diskPercents, ramPercents []float64) []Prediction {
	var preds []Prediction

	if len(cpuTemps) >= 10 {
		slope := PredictTrend(cpuTemps)
		avg := average(cpuTemps)
		if slope > 0.05 && avg > 65 {
			prob := math.Min((avg-65)/35+slope*10, 0.95)
			preds = append(preds, Prediction{
				Type:        "cpu_overheat",
				Probability: prob,
				TimeFrame:   "3일 후",
				Advice:      "CPU 온도가 꾸준히 오르고 있어요. 냉각 팬 청소를 권장해요.",
				AutoAction:  "autoclean",
			})
		}
	}

	if len(diskPercents) >= 10 {
		slope := PredictTrend(diskPercents)
		cur := diskPercents[len(diskPercents)-1]
		if slope > 0 {
			days := (100 - cur) / (slope * 1440)
			if days < 14 && days > 0 {
				preds = append(preds, Prediction{
					Type:        "disk_full",
					Probability: 1 - days/14,
					TimeFrame:   fmt.Sprintf("%.0f일 후", days),
					Advice:      "저장공간이 빠르게 줄고 있어요. 미리 정리해두세요.",
					AutoAction:  "autoclean",
				})
			}
		}
	}

	if len(ramPercents) >= 10 {
		start := clamp(len(ramPercents)-50, 0, len(ramPercents))
		avg := average(ramPercents[start:])
		if avg > 80 {
			preds = append(preds, Prediction{
				Type:        "ram_pressure",
				Probability: math.Min((avg-80)/20, 0.95),
				TimeFrame:   "지금",
				Advice:      "메모리를 많이 사용 중이에요. 최적화를 권장해요.",
				AutoAction:  "monitor",
			})
		}
	}

	return preds
}
