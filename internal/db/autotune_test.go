package db

import (
	"testing"
	"time"
)

func TestScalePool(t *testing.T) {
	tests := []struct {
		name        string
		ramBytes    int64
		wantMaxOpen int
		wantMaxIdle int
	}{
		{
			name:        "512MB RAM (low memory)",
			ramBytes:    512 << 20,
			wantMaxOpen: 10,
			wantMaxIdle: 5,
		},
		{
			name:        "1GB RAM (minimum tier)",
			ramBytes:    1 << 30,
			wantMaxOpen: 10,
			wantMaxIdle: 5,
		},
		{
			name:        "1.5GB RAM (between 1GB and 2GB)",
			ramBytes:    (1 << 30) + (512 << 20),
			wantMaxOpen: 25,
			wantMaxIdle: 10,
		},
		{
			name:        "2GB RAM (standard tier)",
			ramBytes:    2 << 30,
			wantMaxOpen: 25,
			wantMaxIdle: 10,
		},
		{
			name:        "3GB RAM (between 2GB and 4GB)",
			ramBytes:    3 << 30,
			wantMaxOpen: 37,
			wantMaxIdle: 17,
		},
		{
			name:        "4GB RAM (high tier)",
			ramBytes:    4 << 30,
			wantMaxOpen: 50,
			wantMaxIdle: 25,
		},
		{
			name:        "8GB RAM (above 4GB)",
			ramBytes:    8 << 30,
			wantMaxOpen: 50,
			wantMaxIdle: 25,
		},
		{
			name:        "16GB RAM (well above 4GB)",
			ramBytes:    16 << 30,
			wantMaxOpen: 50,
			wantMaxIdle: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := scalePool(tt.ramBytes)

			if cfg.MaxOpen != tt.wantMaxOpen {
				t.Errorf("MaxOpen = %d, want %d", cfg.MaxOpen, tt.wantMaxOpen)
			}
			if cfg.MaxIdle != tt.wantMaxIdle {
				t.Errorf("MaxIdle = %d, want %d", cfg.MaxIdle, tt.wantMaxIdle)
			}
			if cfg.MaxLifetime != 5*time.Minute {
				t.Errorf("MaxLifetime = %v, want 5m", cfg.MaxLifetime)
			}
			if cfg.MaxIdleTime != 2*time.Minute {
				t.Errorf("MaxIdleTime = %v, want 2m", cfg.MaxIdleTime)
			}
		})
	}
}

func TestScalePoolInvariants(t *testing.T) {
	// MaxIdle should always be less than or equal to MaxOpen
	ramValues := []int64{
		256 << 20, // 256 MB
		512 << 20, // 512 MB
		1 << 30,   // 1 GB
		2 << 30,   // 2 GB
		3 << 30,   // 3 GB
		4 << 30,   // 4 GB
		8 << 30,   // 8 GB
		32 << 30,  // 32 GB
	}

	for _, ram := range ramValues {
		cfg := scalePool(ram)

		if cfg.MaxIdle > cfg.MaxOpen {
			t.Errorf("RAM=%d: MaxIdle(%d) > MaxOpen(%d)", ram, cfg.MaxIdle, cfg.MaxOpen)
		}
		if cfg.MaxOpen < 1 {
			t.Errorf("RAM=%d: MaxOpen(%d) < 1", ram, cfg.MaxOpen)
		}
		if cfg.MaxIdle < 1 {
			t.Errorf("RAM=%d: MaxIdle(%d) < 1", ram, cfg.MaxIdle)
		}
	}
}

func TestDetectSystemRAM(t *testing.T) {
	ram := detectSystemRAM()
	// On any platform, we should get a positive value
	if ram <= 0 {
		t.Errorf("detectSystemRAM() = %d, want > 0", ram)
	}
	// On non-Linux (where tests are likely run), expect 2GB fallback
	// On Linux, expect a real value (still positive)
}
