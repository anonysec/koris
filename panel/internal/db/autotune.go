package db

import (
	"database/sql"
	"time"
)

// PoolConfig holds connection pool parameters applied to a *sql.DB.
type PoolConfig struct {
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
	MaxIdleTime time.Duration
}

// AutoTunePool detects available system RAM and applies appropriate
// connection pool settings to the given database handle. It returns
// the configuration that was applied.
func AutoTunePool(db *sql.DB) PoolConfig {
	ramBytes := detectSystemRAM()
	cfg := scalePool(ramBytes)

	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)

	return cfg
}

// scalePool returns pool settings proportional to available system RAM.
//
// Scaling tiers:
//
//	≤1 GB  → MaxOpen=10,  MaxIdle=5
//	≤2 GB  → MaxOpen=25,  MaxIdle=10
//	≥4 GB  → MaxOpen=50,  MaxIdle=25
//
// Values between 2–4 GB interpolate linearly.
func scalePool(ramBytes int64) PoolConfig {
	const (
		gb1 = 1 << 30 // 1 GiB
		gb2 = 2 << 30 // 2 GiB
		gb4 = 4 << 30 // 4 GiB
	)

	var maxOpen, maxIdle int

	switch {
	case ramBytes <= gb1:
		maxOpen = 10
		maxIdle = 5
	case ramBytes <= gb2:
		maxOpen = 25
		maxIdle = 10
	default:
		// 4GB+ gets the maximum settings
		if ramBytes >= gb4 {
			maxOpen = 50
			maxIdle = 25
		} else {
			// Between 2GB and 4GB: linear interpolation
			// 2GB → 25 open, 4GB → 50 open
			fraction := float64(ramBytes-gb2) / float64(gb4-gb2)
			maxOpen = 25 + int(fraction*25)
			maxIdle = 10 + int(fraction*15)
		}
	}

	return PoolConfig{
		MaxOpen:     maxOpen,
		MaxIdle:     maxIdle,
		MaxLifetime: 5 * time.Minute,
		MaxIdleTime: 2 * time.Minute,
	}
}
