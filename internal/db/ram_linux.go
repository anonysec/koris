//go:build linux

package db

import (
	"os"
	"strconv"
	"strings"
)

// detectSystemRAM reads /proc/meminfo to determine total system RAM in bytes.
// Falls back to 2 GiB if the file cannot be read or parsed.
func detectSystemRAM() int64 {
	const fallback = 2 << 30 // 2 GiB

	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return fallback
	}

	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		// Format: "MemTotal:       16384000 kB"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fallback
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return fallback
		}
		return kb * 1024 // kB → bytes
	}

	return fallback
}
