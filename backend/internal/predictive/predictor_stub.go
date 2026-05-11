//go:build !windows

package predictive

type Prediction struct {
	Type        string  `json:"type"`
	Probability float64 `json:"probability"`
	TimeFrame   string  `json:"time_frame"`
	Advice      string  `json:"advice"`
	AutoAction  string  `json:"auto_action"`
}

func PredictTrend(values []float64) float64 { return 0 }

func GeneratePredictions(cpu, disk, ram []float64) []Prediction { return nil }
