//go:build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func diskPercent(mount string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mount, &stat); err != nil {
		return 0
	}
	total := float64(stat.Blocks)
	free := float64(stat.Bavail)
	if total <= 0 {
		return 0
	}
	return round2((total - free) / total * 100)
}

// cpuState stores the previous /proc/stat reading for non-blocking CPU percentage calculation.
var cpuPrevIdle, cpuPrevTotal uint64

func cpuPercent() float64 {
	idle, total := readCPU()
	if cpuPrevTotal == 0 {
		cpuPrevIdle, cpuPrevTotal = idle, total
		return 0
	}
	idleDelta := float64(idle - cpuPrevIdle)
	totalDelta := float64(total - cpuPrevTotal)
	cpuPrevIdle, cpuPrevTotal = idle, total
	if totalDelta <= 0 {
		return 0
	}
	return round2((1 - idleDelta/totalDelta) * 100)
}

func readCPU() (idle, total uint64) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	fields := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
	for i, field := range fields[1:] {
		v, _ := strconv.ParseUint(field, 10, 64)
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return idle, total
}

func memPercent() float64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	vals := map[string]float64{}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			vals[key], _ = strconv.ParseFloat(fields[1], 64)
		}
	}
	total := vals["MemTotal"]
	available := vals["MemAvailable"]
	if total <= 0 {
		return 0
	}
	return round2((total - available) / total * 100)
}

func netBytes() (rx, tx int64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		r, _ := strconv.ParseInt(fields[0], 10, 64)
		t, _ := strconv.ParseInt(fields[8], 10, 64)
		rx += r
		tx += t
	}
	return rx, tx
}

func round2(v float64) float64 {
	n, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return n
}
