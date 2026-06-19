//go:build !linux

package main

import (
	"fmt"
	"strconv"
)

// Stubs for non-Linux platforms.
// The node agent only runs on Linux servers, but these allow
// cross-platform compilation for development and testing.

func diskPercent(mount string) float64 { return 0 }
func cpuPercent() float64              { return 0 }
func memPercent() float64              { return 0 }
func netBytes() (rx, tx int64)         { return 0, 0 }

var cpuPrevIdle, cpuPrevTotal uint64

func round2(v float64) float64 {
	n, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return n
}
