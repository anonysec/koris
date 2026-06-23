package db

import (
	"database/sql"
	"testing"
	"time"
)

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name         string
		envMaxOpen   string
		envMaxIdle   string
		envLifetime  string
		baseCfg      PoolConfig
		wantMaxOpen  int
		wantMaxIdle  int
		wantLifetime time.Duration
	}{
		{
			name:         "no env vars set, config unchanged",
			envMaxOpen:   "",
			envMaxIdle:   "",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "override MaxOpen only",
			envMaxOpen:   "50",
			envMaxIdle:   "",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  50,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "override MaxIdle only",
			envMaxOpen:   "",
			envMaxIdle:   "20",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  20,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "override MaxLifetime with minutes",
			envMaxOpen:   "",
			envMaxIdle:   "",
			envLifetime:  "10m",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 10 * time.Minute,
		},
		{
			name:         "override MaxLifetime with seconds",
			envMaxOpen:   "",
			envMaxIdle:   "",
			envLifetime:  "300s",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 300 * time.Second,
		},
		{
			name:         "override all three",
			envMaxOpen:   "100",
			envMaxIdle:   "50",
			envLifetime:  "15m",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  100,
			wantMaxIdle:  50,
			wantLifetime: 15 * time.Minute,
		},
		{
			name:         "invalid MaxOpen ignored",
			envMaxOpen:   "abc",
			envMaxIdle:   "",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "invalid MaxIdle ignored",
			envMaxOpen:   "",
			envMaxIdle:   "xyz",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "invalid lifetime ignored",
			envMaxOpen:   "",
			envMaxIdle:   "",
			envLifetime:  "badvalue",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "zero MaxOpen ignored",
			envMaxOpen:   "0",
			envMaxIdle:   "",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
		{
			name:         "negative MaxIdle ignored",
			envMaxOpen:   "",
			envMaxIdle:   "-5",
			envLifetime:  "",
			baseCfg:      PoolConfig{MaxOpen: 25, MaxIdle: 10, MaxLifetime: 5 * time.Minute},
			wantMaxOpen:  25,
			wantMaxIdle:  10,
			wantLifetime: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars for this test case
			t.Setenv("PANEL_DB_MAX_OPEN", tt.envMaxOpen)
			t.Setenv("PANEL_DB_MAX_IDLE", tt.envMaxIdle)
			t.Setenv("PANEL_DB_MAX_LIFETIME", tt.envLifetime)

			// Use a real sql.DB (driver doesn't matter for pool config)
			db, err := sql.Open("mysql", "invalid:invalid@tcp(127.0.0.1:1)/test")
			if err != nil {
				t.Fatalf("sql.Open failed: %v", err)
			}
			defer db.Close()

			// Set the base config on the db before applying overrides
			db.SetMaxOpenConns(tt.baseCfg.MaxOpen)
			db.SetMaxIdleConns(tt.baseCfg.MaxIdle)
			db.SetConnMaxLifetime(tt.baseCfg.MaxLifetime)

			cfg := tt.baseCfg
			ApplyEnvOverrides(db, &cfg)

			if cfg.MaxOpen != tt.wantMaxOpen {
				t.Errorf("MaxOpen = %d, want %d", cfg.MaxOpen, tt.wantMaxOpen)
			}
			if cfg.MaxIdle != tt.wantMaxIdle {
				t.Errorf("MaxIdle = %d, want %d", cfg.MaxIdle, tt.wantMaxIdle)
			}
			if cfg.MaxLifetime != tt.wantLifetime {
				t.Errorf("MaxLifetime = %v, want %v", cfg.MaxLifetime, tt.wantLifetime)
			}
		})
	}
}
