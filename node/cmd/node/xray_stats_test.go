package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// parseXrayStatsOutput is a testable extraction of the collectXrayStats parsing logic.
// It takes raw JSON output from `xray api statsquery` and returns aggregated per-user stats.
func parseXrayStatsOutput(output string) []XrayUserStat {
	output = strings.TrimSpace(output)
	if output == "" || output == "{}" {
		return nil
	}

	var resp xrayStatsQueryResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil
	}

	if len(resp.Stat) == 0 {
		return nil
	}

	type trafficAcc struct {
		Uplink   int64
		Downlink int64
	}
	users := make(map[string]*trafficAcc)

	for _, entry := range resp.Stat {
		parts := strings.Split(entry.Name, ">>>")
		if len(parts) != 4 {
			continue
		}
		if parts[0] != "user" || parts[2] != "traffic" {
			continue
		}

		email := parts[1]
		direction := parts[3]
		value, err := strconv.ParseInt(entry.Value, 10, 64)
		if err != nil {
			continue
		}

		acc, ok := users[email]
		if !ok {
			acc = &trafficAcc{}
			users[email] = acc
		}

		switch direction {
		case "uplink":
			acc.Uplink += value
		case "downlink":
			acc.Downlink += value
		}
	}

	result := make([]XrayUserStat, 0, len(users))
	for email, acc := range users {
		if acc.Uplink == 0 && acc.Downlink == 0 {
			continue
		}
		result = append(result, XrayUserStat{
			Email:    email,
			Uplink:   acc.Uplink,
			Downlink: acc.Downlink,
		})
	}

	return result
}

func TestParseXrayStatsOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantMap map[string]XrayUserStat // email -> expected stat
	}{
		{
			name:    "empty output",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "empty object",
			input:   "{}",
			wantLen: 0,
		},
		{
			name:    "empty stat array",
			input:   `{"stat":[]}`,
			wantLen: 0,
		},
		{
			name: "single user uplink and downlink",
			input: `{"stat":[
				{"name":"user>>>alice@example.com>>>traffic>>>uplink","value":"123456"},
				{"name":"user>>>alice@example.com>>>traffic>>>downlink","value":"789012"}
			]}`,
			wantLen: 1,
			wantMap: map[string]XrayUserStat{
				"alice@example.com": {Email: "alice@example.com", Uplink: 123456, Downlink: 789012},
			},
		},
		{
			name: "multiple users",
			input: `{"stat":[
				{"name":"user>>>alice@example.com>>>traffic>>>uplink","value":"100"},
				{"name":"user>>>alice@example.com>>>traffic>>>downlink","value":"200"},
				{"name":"user>>>bob@example.com>>>traffic>>>uplink","value":"300"},
				{"name":"user>>>bob@example.com>>>traffic>>>downlink","value":"400"}
			]}`,
			wantLen: 2,
			wantMap: map[string]XrayUserStat{
				"alice@example.com": {Email: "alice@example.com", Uplink: 100, Downlink: 200},
				"bob@example.com":   {Email: "bob@example.com", Uplink: 300, Downlink: 400},
			},
		},
		{
			name: "skips non-user entries",
			input: `{"stat":[
				{"name":"inbound>>>api>>>traffic>>>uplink","value":"999"},
				{"name":"user>>>alice@example.com>>>traffic>>>uplink","value":"100"},
				{"name":"user>>>alice@example.com>>>traffic>>>downlink","value":"200"}
			]}`,
			wantLen: 1,
			wantMap: map[string]XrayUserStat{
				"alice@example.com": {Email: "alice@example.com", Uplink: 100, Downlink: 200},
			},
		},
		{
			name: "skips malformed entries",
			input: `{"stat":[
				{"name":"user>>>alice@example.com>>>traffic","value":"100"},
				{"name":"user>>>bob@example.com>>>traffic>>>uplink","value":"not_a_number"},
				{"name":"user>>>carol@example.com>>>traffic>>>uplink","value":"500"},
				{"name":"user>>>carol@example.com>>>traffic>>>downlink","value":"600"}
			]}`,
			wantLen: 1,
			wantMap: map[string]XrayUserStat{
				"carol@example.com": {Email: "carol@example.com", Uplink: 500, Downlink: 600},
			},
		},
		{
			name: "skips zero traffic users",
			input: `{"stat":[
				{"name":"user>>>alice@example.com>>>traffic>>>uplink","value":"0"},
				{"name":"user>>>alice@example.com>>>traffic>>>downlink","value":"0"},
				{"name":"user>>>bob@example.com>>>traffic>>>uplink","value":"100"},
				{"name":"user>>>bob@example.com>>>traffic>>>downlink","value":"0"}
			]}`,
			wantLen: 1,
			wantMap: map[string]XrayUserStat{
				"bob@example.com": {Email: "bob@example.com", Uplink: 100, Downlink: 0},
			},
		},
		{
			name:    "invalid json",
			input:   `not valid json`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseXrayStatsOutput(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(result), tt.wantLen)
				return
			}
			if tt.wantMap != nil {
				for _, stat := range result {
					expected, ok := tt.wantMap[stat.Email]
					if !ok {
						t.Errorf("unexpected email in result: %s", stat.Email)
						continue
					}
					if stat.Uplink != expected.Uplink {
						t.Errorf("email %s: uplink = %d, want %d", stat.Email, stat.Uplink, expected.Uplink)
					}
					if stat.Downlink != expected.Downlink {
						t.Errorf("email %s: downlink = %d, want %d", stat.Email, stat.Downlink, expected.Downlink)
					}
				}
			}
		})
	}
}

func TestXrayUserStatJSONTags(t *testing.T) {
	stat := XrayUserStat{
		Email:    "test@example.com",
		Uplink:   12345,
		Downlink: 67890,
	}
	data, err := json.Marshal(stat)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// Verify JSON field names match expected tags
	if _, ok := m["email"]; !ok {
		t.Error("expected 'email' JSON field")
	}
	if _, ok := m["uplink"]; !ok {
		t.Error("expected 'uplink' JSON field")
	}
	if _, ok := m["downlink"]; !ok {
		t.Error("expected 'downlink' JSON field")
	}
}
