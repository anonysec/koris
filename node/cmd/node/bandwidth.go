package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UserBandwidth represents per-user bandwidth metrics derived from tc class stats.
type UserBandwidth struct {
	IP       string `json:"ip"`
	ClassID  string `json:"class_id"`
	RxBytes  int64  `json:"rx_bytes"`
	TxBytes  int64  `json:"tx_bytes"`
	RxBps    int64  `json:"rx_bps"`
	TxBps    int64  `json:"tx_bps"`
}

// tcClassStat holds raw byte counters for a single tc class.
type tcClassStat struct {
	ClassID string
	Bytes   int64
}

// BandwidthCollector runs tc -s class show dev and computes per-class bandwidth rates.
type BandwidthCollector struct {
	prev        map[string]int64
	lastCollect time.Time
}

// NewBandwidthCollector creates a new BandwidthCollector instance.
func NewBandwidthCollector() *BandwidthCollector {
	return &BandwidthCollector{
		prev:        make(map[string]int64),
		lastCollect: time.Time{},
	}
}

// Collect runs tc -s class show dev on the given device and returns per-class bandwidth data.
func (bc *BandwidthCollector) Collect(dev string) []UserBandwidth {
	out, err := exec.Command("tc", "-s", "class", "show", "dev", dev).CombinedOutput()
	if err != nil {
		return nil
	}

	current := parseTcOutput(string(out))
	now := time.Now()

	var dt float64
	if !bc.lastCollect.IsZero() {
		dt = now.Sub(bc.lastCollect).Seconds()
	}

	result := DeltaRates(current, bc.prev, dt)

	// Update prev state
	bc.prev = make(map[string]int64, len(current))
	for _, cs := range current {
		bc.prev[cs.ClassID] = cs.Bytes
	}
	bc.lastCollect = now

	return result
}

// classLineRe matches a tc class line like "class htb 1:52 parent ..." or "class htb 1:10 root ..."
var classLineRe = regexp.MustCompile(`class\s+\S+\s+(\d+:\d+)\s+`)

// sentLineRe matches the "Sent NNN bytes ..." line
var sentLineRe = regexp.MustCompile(`Sent\s+(\d+)\s+bytes`)

// parseTcOutput parses the output of tc -s class show dev and extracts per-class byte counters.
// It skips root classes (e.g. 1:0, 1:1, 1:10) and only returns user classes where the minor
// number (after the colon) represents the last octet of a user IP.
func parseTcOutput(output string) []tcClassStat {
	var stats []tcClassStat
	lines := strings.Split(output, "\n")

	var currentClassID string
	for _, line := range lines {
		// Check for class line
		if m := classLineRe.FindStringSubmatch(line); m != nil {
			classID := m[1]
			// Skip root/infrastructure classes (minor number <= 10)
			parts := strings.SplitN(classID, ":", 2)
			if len(parts) == 2 {
				minor, err := strconv.Atoi(parts[1])
				if err != nil || minor <= 10 {
					currentClassID = ""
					continue
				}
			}
			currentClassID = classID
			continue
		}

		// Check for Sent line
		if currentClassID != "" {
			if m := sentLineRe.FindStringSubmatch(line); m != nil {
				bytes, err := strconv.ParseInt(m[1], 10, 64)
				if err == nil {
					stats = append(stats, tcClassStat{
						ClassID: currentClassID,
						Bytes:   bytes,
					})
				}
				currentClassID = ""
			}
		}
	}

	return stats
}

// DeltaRates computes per-second bandwidth rates from current class stats and previous byte counters.
// Counter-wrap clamping: if the current byte count is less than the previous (counter wrap or reset),
// the rate is clamped to 0.
//
// NOTE: tc class stats on the tun0 egress qdisc represent data flowing FROM the server TO the client.
// This is the client's download (Rx) direction. We assign the same value to both RxBps and TxBps
// because we only have a single "Sent" counter from the egress qdisc - there are no per-class
// ingress filter stats available to measure client upload (Tx) separately. When ingress policing
// stats become available in the future, TxBps should be populated from those counters instead.
func DeltaRates(current []tcClassStat, prev map[string]int64, dt float64) []UserBandwidth {
	var result []UserBandwidth

	for _, cs := range current {
		bw := UserBandwidth{
			ClassID: cs.ClassID,
			RxBytes: cs.Bytes,
			TxBytes: cs.Bytes, // Same value; only egress (server->client) stats available
		}

		// Derive IP hint from class ID (last octet)
		parts := strings.SplitN(cs.ClassID, ":", 2)
		if len(parts) == 2 {
			bw.IP = parts[1]
		}

		if dt > 0 {
			if prevBytes, ok := prev[cs.ClassID]; ok {
				delta := cs.Bytes - prevBytes
				if delta < 0 {
					// Counter wrap - clamp to 0
					delta = 0
				}
				rate := int64(float64(delta) / dt)
				// Both Rx and Tx get the same rate because tc class on tun0 egress only
				// provides a single counter representing server->client (download) traffic.
				bw.RxBps = rate
				bw.TxBps = rate
			}
		}

		result = append(result, bw)
	}

	return result
}
