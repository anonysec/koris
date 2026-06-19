package backup

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateManifest_Structure(t *testing.T) {
	ts := time.Date(2024, 6, 15, 2, 0, 0, 0, time.UTC)
	nodes := []string{"node-1", "node-2"}
	skipped := []SkippedNode{{Name: "node-3", Reason: "timeout"}}
	files := map[string]FileInfo{"dump.sql": {Size: 1024}}

	m := GenerateManifest(ts, "2.1.0", "radius_next", nodes, skipped, files, 0, 0)

	if m.Version != "1.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0")
	}
	if m.ChecksumAlgorithm != "sha256" {
		t.Errorf("ChecksumAlgorithm = %q, want %q", m.ChecksumAlgorithm, "sha256")
	}
	// Verify timestamp is valid RFC3339
	if _, err := time.Parse(time.RFC3339, m.Timestamp); err != nil {
		t.Errorf("Timestamp %q is not valid RFC3339: %v", m.Timestamp, err)
	}
	if m.PanelVersion != "2.1.0" {
		t.Errorf("PanelVersion = %q, want %q", m.PanelVersion, "2.1.0")
	}
	if m.Database != "radius_next" {
		t.Errorf("Database = %q, want %q", m.Database, "radius_next")
	}
}

func TestGenerateManifest_NilLists(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		nodes   []string
		skipped []SkippedNode
	}{
		{
			name:    "nil node lists",
			nodes:   nil,
			skipped: nil,
		},
		{
			name:    "empty node lists",
			nodes:   []string{},
			skipped: []SkippedNode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := GenerateManifest(ts, "1.0.0", "db", tt.nodes, tt.skipped, nil, 0, 0)

			// Marshal to JSON and verify arrays are [] not null
			data, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			// Check nodes_included is [] not null
			if string(raw["nodes_included"]) == "null" {
				t.Error("nodes_included is null, want empty array []")
			}
			// Check nodes_skipped is [] not null
			if string(raw["nodes_skipped"]) == "null" {
				t.Error("nodes_skipped is null, want empty array []")
			}
		})
	}
}

func TestGenerateManifest_NodesPreserved(t *testing.T) {
	ts := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
	nodes := []string{"alpha", "beta", "gamma"}
	skipped := []SkippedNode{
		{Name: "delta", Reason: "unreachable"},
		{Name: "epsilon", Reason: "timeout"},
	}

	m := GenerateManifest(ts, "2.0.0", "db", nodes, skipped, nil, 0, 0)

	if len(m.NodesIncluded) != 3 {
		t.Fatalf("NodesIncluded len = %d, want 3", len(m.NodesIncluded))
	}
	for i, want := range nodes {
		if m.NodesIncluded[i] != want {
			t.Errorf("NodesIncluded[%d] = %q, want %q", i, m.NodesIncluded[i], want)
		}
	}

	if len(m.NodesSkipped) != 2 {
		t.Fatalf("NodesSkipped len = %d, want 2", len(m.NodesSkipped))
	}
	for i, want := range skipped {
		if m.NodesSkipped[i].Name != want.Name {
			t.Errorf("NodesSkipped[%d].Name = %q, want %q", i, m.NodesSkipped[i].Name, want.Name)
		}
		if m.NodesSkipped[i].Reason != want.Reason {
			t.Errorf("NodesSkipped[%d].Reason = %q, want %q", i, m.NodesSkipped[i].Reason, want.Reason)
		}
	}
}
