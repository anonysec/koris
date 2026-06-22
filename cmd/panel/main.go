package main

import (
	"log"
	"net/http"
	"os"

	"KorisLite/internal/api"
	"KorisLite/internal/config"
	"KorisLite/internal/db"
)

func main() {
	cfg := config.Load()

	log.Printf("[main] KorisPanel Lite %s starting...", cfg.Version)

	// Open database
	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("[main] database connection failed: %v", err)
	}
	log.Printf("[main] database connected")

	// Run migrations
	if err := db.Migrate(database, cfg.MigrationsDir); err != nil {
		log.Fatalf("[main] migrations failed: %v", err)
	}
	log.Printf("[main] migrations complete")

	// Create server
	srv := api.New(database, cfg)
	handler := srv.Routes()

	// Check if initial setup needed
	var adminCount int
	database.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&adminCount)
	if adminCount == 0 {
		log.Printf("[main] ⚠ No admin accounts found. Create one with:")
		log.Printf("[main]   POST /api/auth/setup {\"username\": \"admin\", \"password\": \"yourpass\"}")
		// Register setup endpoint
		mux := http.NewServeMux()
		mux.Handle("/", handler)
		mux.HandleFunc("/api/auth/setup", srv.SetupHandler())
		handler = mux
	}

	log.Printf("[main] listening on %s", cfg.Addr)
	log.Printf("[main] admin: http://localhost%s/dashboard/", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("[main] server error: %v", err)
	}
}

func init() {
	// Ensure proper timezone for MariaDB
	os.Setenv("TZ", "UTC")
}
