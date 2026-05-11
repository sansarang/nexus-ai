//go:build !windows

package winfix

type RepairStep struct {
	Name  string `json:"name"`
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

func RepairWindowsUpdate() chan RepairStep {
	ch := make(chan RepairStep, 1)
	close(ch)
	return ch
}
