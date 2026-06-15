package config

import "os"

type Config struct {
	Addr          string
	DBDSN         string
	SetupKey      string
	SessionSecret string
	Version       string
	PublicBase    string
	AdminWebDir   string
	PortalWebDir  string
}

func Load() Config {
	setupKey := getenv("PANEL_SETUP_KEY", "")
	sessionSecret := getenv("PANEL_SESSION_SECRET", "")
	if sessionSecret == "" {
		sessionSecret = setupKey
	}
	if sessionSecret == "" {
		sessionSecret = "koris-next-dev-session-secret"
	}

	return Config{
		Addr:          getenv("PANEL_ADDR", ":8080"),
		DBDSN:         getenv("PANEL_DB_DSN", "radius:RadiusDb2026@tcp(127.0.0.1:3306)/radius?parseTime=true&multiStatements=true&charset=utf8mb4,utf8"),
		SetupKey:      setupKey,
		SessionSecret: sessionSecret,
		Version:       getenv("PANEL_VERSION", "next-dev"),
		PublicBase:    getenv("PANEL_PUBLIC_BASE", "/dashboard"),
		AdminWebDir:   getenv("PANEL_ADMIN_WEB_DIR", "/opt/koris-next/panel/web/admin/www"),
		PortalWebDir:  getenv("PANEL_PORTAL_WEB_DIR", "/opt/koris-next/panel/web/portal/www"),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
