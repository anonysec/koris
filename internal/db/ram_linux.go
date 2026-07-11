//go:build linux

package db

import (
	"os"
	"strconv"
	"strings"
)

// detectSystemRAM returns the memory available to the process, preferring the
// container's cgroup limit over the host's total RAM. In Docker/Kubernetes the
// host RAM reported by /proc/meminfo can be far larger than the container's
// allocation, which would otherwise over-provision the connection pool.
// Falls back to 2 GiB if nothing can be determined.
func detectSystemRAM() int64 {
	const fallback = 2 << 30 // 2 GiB

	if n := detectCgroupMemoryLimit(); n > 0 {
		return n
	}

	// Fall back to host total RAM from /proc/meminfo.
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(line, "MemTotal:") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return fallback
			}
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				return kb * 1024
			}
		}
	}

	return fallback
}

// detectCgroupMemoryLimit reads the cgroup memory limit and returns it in
// bytes, or 0 if no effective limit is set (unlimited / not in a cgroup).
func detectCgroupMemoryLimit() int64 {
	// cgroup v2: /sys/fs/cgroup/memory.max ("max" means unlimited).
	if b, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		s := strings.TrimSpace(string(b))
		if s != "" && s != "max" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}

	// cgroup v1: /sys/fs/cgroup/memory/memory.limit_in_bytes reports a huge
	// sentinel when unlimited, so only trust it when it is below host RAM.
	if b, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		s := strings.TrimSpace(string(b))
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 && n < 1<<62 {
			if host := hostMemTotal(); n < host {
				return n
			}
		}
	}

	return 0
}

// hostMemTotal returns the host's total RAM from /proc/meminfo, used to detect
// the cgroup v1 "unlimited" sentinel.
func hostMemTotal() int64 {
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						return kb * 1024
					}
				}
			}
		}
	}
	return 1 << 62
}
