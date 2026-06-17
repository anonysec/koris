package main

import (
	"testing"
)

func TestParseTcOutput(t *testing.T) {
	sampleOutput := `class htb 1:10 root prio 0 rate 10000Mbit ceil 10000Mbit burst 0b cburst 0b 
 Sent 1234567 bytes 1000 pkt (dropped 0, overlimits 0 requeues 0) 
 backlog 0b 0p requeues 0
 lended: 0 borrowed: 0 giants: 0
 tokens: -500 ctokens: -500

class htb 1:1 root prio 0 rate 1000Mbit ceil 1000Mbit burst 0b cburst 0b 
 Sent 999999 bytes 500 pkt (dropped 0, overlimits 0 requeues 0) 
 backlog 0b 0p requeues 0

class htb 1:52 parent 1:1 prio 52 rate 10Mbit ceil 10Mbit burst 1600b cburst 1600b 
 Sent 98765 bytes 50 pkt (dropped 0, overlimits 0 requeues 0) 
 backlog 0b 0p requeues 0

class htb 1:100 parent 1:1 prio 100 rate 5Mbit ceil 5Mbit burst 1600b cburst 1600b 
 Sent 54321 bytes 30 pkt (dropped 0, overlimits 0 requeues 0) 
 backlog 0b 0p requeues 0
`

	stats := parseTcOutput(sampleOutput)

	// Should skip 1:10 (minor <= 10) and 1:1 (minor <= 10), parse 1:52 and 1:100
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}

	if stats[0].ClassID != "1:52" {
		t.Errorf("expected class 1:52, got %s", stats[0].ClassID)
	}
	if stats[0].Bytes != 98765 {
		t.Errorf("expected 98765 bytes for 1:52, got %d", stats[0].Bytes)
	}

	if stats[1].ClassID != "1:100" {
		t.Errorf("expected class 1:100, got %s", stats[1].ClassID)
	}
	if stats[1].Bytes != 54321 {
		t.Errorf("expected 54321 bytes for 1:100, got %d", stats[1].Bytes)
	}
}

func TestParseTcOutputEmpty(t *testing.T) {
	stats := parseTcOutput("")
	if len(stats) != 0 {
		t.Fatalf("expected 0 stats for empty input, got %d", len(stats))
	}
}

func TestDeltaRatesNormal(t *testing.T) {
	current := []tcClassStat{
		{ClassID: "1:52", Bytes: 200000},
		{ClassID: "1:100", Bytes: 100000},
	}
	prev := map[string]int64{
		"1:52":  100000,
		"1:100": 50000,
	}
	dt := 10.0 // 10 seconds

	result := DeltaRates(current, prev, dt)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// 1:52: (200000-100000)/10 = 10000 bps
	if result[0].ClassID != "1:52" {
		t.Errorf("expected class 1:52, got %s", result[0].ClassID)
	}
	if result[0].RxBps != 10000 {
		t.Errorf("expected RxBps 10000 for 1:52, got %d", result[0].RxBps)
	}
	if result[0].TxBps != 10000 {
		t.Errorf("expected TxBps 10000 for 1:52, got %d", result[0].TxBps)
	}
	if result[0].IP != "52" {
		t.Errorf("expected IP '52' for 1:52, got %s", result[0].IP)
	}

	// 1:100: (100000-50000)/10 = 5000 bps
	if result[1].RxBps != 5000 {
		t.Errorf("expected RxBps 5000 for 1:100, got %d", result[1].RxBps)
	}
}

func TestDeltaRatesCounterWrap(t *testing.T) {
	current := []tcClassStat{
		{ClassID: "1:52", Bytes: 5000}, // Counter wrapped (less than previous)
	}
	prev := map[string]int64{
		"1:52": 100000,
	}
	dt := 10.0

	result := DeltaRates(current, prev, dt)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	// Counter wrap should clamp to 0
	if result[0].RxBps != 0 {
		t.Errorf("expected RxBps 0 on counter wrap, got %d", result[0].RxBps)
	}
	if result[0].TxBps != 0 {
		t.Errorf("expected TxBps 0 on counter wrap, got %d", result[0].TxBps)
	}
}

func TestDeltaRatesNoPrevious(t *testing.T) {
	current := []tcClassStat{
		{ClassID: "1:52", Bytes: 50000},
	}
	prev := map[string]int64{} // No previous data

	dt := 10.0

	result := DeltaRates(current, prev, dt)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	// No previous data means no rate calculation
	if result[0].RxBps != 0 {
		t.Errorf("expected RxBps 0 with no previous, got %d", result[0].RxBps)
	}
	if result[0].TxBps != 0 {
		t.Errorf("expected TxBps 0 with no previous, got %d", result[0].TxBps)
	}
}

func TestDeltaRatesZeroDt(t *testing.T) {
	current := []tcClassStat{
		{ClassID: "1:52", Bytes: 200000},
	}
	prev := map[string]int64{
		"1:52": 100000,
	}
	dt := 0.0 // First collection, no time delta

	result := DeltaRates(current, prev, dt)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	// Zero dt means no rate calculation
	if result[0].RxBps != 0 {
		t.Errorf("expected RxBps 0 with zero dt, got %d", result[0].RxBps)
	}
}
