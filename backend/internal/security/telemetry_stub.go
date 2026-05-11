//go:build !windows

package security

func BlockTelemetry() error   { return nil }
func UnblockTelemetry() error { return nil }
