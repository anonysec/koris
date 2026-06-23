package tui

import (
	"bytes"
	"os"
	"testing"
)

func TestNew_EnvVarDisablesDashboard(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantDash bool
	}{
		{"false disables", "false", false},
		{"FALSE disables", "FALSE", false},
		{"False disables", "False", false},
		{"0 disables", "0", false},
		{"no disables", "no", false},
		{"NO disables", "NO", false},
		{"true keeps enabled", "true", true},
		{"1 keeps enabled", "1", true},
		{"yes keeps enabled", "yes", true},
		{"empty keeps enabled", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("PANEL_TUI_ENABLED", tt.envValue)
				defer os.Unsetenv("PANEL_TUI_ENABLED")
			} else {
				os.Unsetenv("PANEL_TUI_ENABLED")
			}

			l := New(WithOutput(&bytes.Buffer{}))
			if l.dashboardEnabled != tt.wantDash {
				t.Errorf("dashboardEnabled = %v, want %v", l.dashboardEnabled, tt.wantDash)
			}
		})
	}
}

func TestNew_WithDashboardOverridesEnvVar(t *testing.T) {
	// Set env var to disable dashboard.
	os.Setenv("PANEL_TUI_ENABLED", "false")
	defer os.Unsetenv("PANEL_TUI_ENABLED")

	// Explicitly enable dashboard via option — should override env var.
	l := New(WithOutput(&bytes.Buffer{}), WithDashboard(true))
	if !l.dashboardEnabled {
		t.Error("WithDashboard(true) should override PANEL_TUI_ENABLED=false")
	}
	if !l.dashboardOverridden {
		t.Error("dashboardOverridden should be true after WithDashboard call")
	}
}

func TestNew_WithDashboardFalseOverridesDefault(t *testing.T) {
	// Ensure env var is NOT set (so default would be true).
	os.Unsetenv("PANEL_TUI_ENABLED")

	// Explicitly disable dashboard via option.
	l := New(WithOutput(&bytes.Buffer{}), WithDashboard(false))
	if l.dashboardEnabled {
		t.Error("WithDashboard(false) should disable dashboard regardless of env")
	}
	if !l.dashboardOverridden {
		t.Error("dashboardOverridden should be true after WithDashboard call")
	}
}

func TestNew_DefaultDashboardEnabled(t *testing.T) {
	// No env var set, no WithDashboard option.
	os.Unsetenv("PANEL_TUI_ENABLED")

	l := New(WithOutput(&bytes.Buffer{}))
	if !l.dashboardEnabled {
		t.Error("dashboard should be enabled by default when env var is not set")
	}
	if l.dashboardOverridden {
		t.Error("dashboardOverridden should be false when WithDashboard is not called")
	}
}
