//go:build !linux

package db

// detectSystemRAM returns a fallback value of 2 GiB on non-Linux platforms
// where /proc/meminfo is unavailable.
func detectSystemRAM() int64 {
	return 2 << 30 // 2 GiB
}
