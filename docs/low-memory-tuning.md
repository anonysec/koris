# 🧠 Low-Memory Tuning Guide

Profiles for running Koris on small VPS instances (≈1 GB RAM).

## Panel (Go binary)

The panel auto-optimizes for low memory when no explicit `GOMAXPROCS`/`GOGC`/`GOMEMLIMIT` are set:

- `GOMAXPROCS=1` (single thread)
- `GOGC=50` (more frequent GC, lower peak memory)
- `GOMEMLIMIT=100MB` (soft memory cap)

Override with env vars if needed:

- `GOMAXPROCS=2` for multi-core
- `GOGC=100` for default GC behavior
- `GOMEMLIMIT=200000000` for a 200 MB limit

Database pool sizing (see [Configuration](configuration.md)):

- `PANEL_DB_MAX_OPEN=10`
- `PANEL_DB_MAX_IDLE=2`
- `PANEL_DB_MAX_LIFETIME=5m`

## PostgreSQL / TimescaleDB (recommended for 1 GB RAM)

Add to `postgresql.conf` (or a drop-in in `conf.d/`):

```ini
shared_buffers = 128MB
effective_cache_size = 384MB
work_mem = 4MB
maintenance_work_mem = 32MB
max_connections = 30
wal_buffers = 4MB
# TimescaleDB
timescaledb.max_background_workers = 2
max_worker_processes = 4
```

For the bundled Docker stack, pass these via the TimescaleDB container command or a mounted config file.

## Node Agent (knode)

Already lightweight (~5 MB RSS). No tuning needed.

## Expected Memory Usage (1 GB server)

| Component | Approx. RSS |
|-----------|-------------|
| PostgreSQL/TimescaleDB | ~250 MB |
| Panel binary | ~30–50 MB |
| FreeRADIUS | ~30 MB |
| OS + buffers | ~200 MB |
| **Headroom** | **~450 MB** |
