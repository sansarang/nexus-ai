//go:build !windows

package privacy

func DisableFeature(feature string, disable bool) error { return nil }
