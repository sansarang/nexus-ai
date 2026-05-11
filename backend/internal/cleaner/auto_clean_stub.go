//go:build !windows

package cleaner

type CleanResult struct {
	Item       string `json:"item"`
	FreedBytes int64  `json:"freed_bytes"`
	Error      string `json:"error,omitempty"`
}

func AutoClean(items []string) []CleanResult { return nil }
