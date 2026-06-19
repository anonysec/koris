//go:build !linux

package main

// diskPercent is a stub for non-Linux platforms.
// The node agent only runs on Linux servers, but this allows
// cross-platform compilation for development/testing.
func diskPercent(mount string) float64 {
	return 0
}
