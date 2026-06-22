package config

import "os"

type Config struct {
	Addr          string
	DBDSN         string
	SessionSecret string
	Version       string
	Domain        string
	AdminWebDir   string
	PortalWebDir  string
	MigrationsDir string
}

func Load() Config {
	return Config{
		Addr:          env("PANEL_ADDR", ":9080"),
		DBDSN:         env("PANEL_DB_DSN", "radius:radius@tcp(127.0.0.1:3306)/radius?parseTime=true&multiStatements=true"),
		SessionSecret: env("PANEL_SESSION_SECRET", "change-me-in-production-32chars!"),
		Version:       env("PANEL_VERSION", "0.1.0-lite"),
		Domain:        env("PANEL_DOMAIN", "localhost"),
		AdminWebDir:   env("PANEL_ADMIN_DIR", "web/admin"),
		PortalWebDir:  env("PANEL_PORTAL_DIR", "web/portal"),
		MigrationsDir: env("PANEL_MIGRATIONS", "migrations"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
