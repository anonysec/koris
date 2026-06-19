//go:build linux

package main

import "syscall"

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
