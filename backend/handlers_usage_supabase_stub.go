//go:build !windows

package main

import "fmt"

func supabaseFetchCount(jwt, userID, feature, today string) (int, error) {
	return 0, fmt.Errorf("supabase not available on this platform")
}

func supabaseIncrementRPC(jwt, userID, feature, today string) error {
	return fmt.Errorf("supabase not available on this platform")
}
