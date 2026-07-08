package main

import (
    "crypto/tls"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/anonysec/koris/internal/gateway"
    "github.com/anonysec/koris/internal/tui"
    "golang.org/x/crypto/acme/autocert"
)

var logger *tui.Logger

func main() {
    cfg := gateway.Load()
    logger = tui.New(tui.WithLevel(tui.LevelInfo))
    logger.Info("gateway", "starting", map[string]any{"listen": cfg.ListenAddr, "tls": cfg.TLSMode})

    rateLimiter := gateway.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, cfg.TrustedProxies)
    defer rateLimiter.Stop()

    authMw := gateway.NewAuthMiddleware(cfg.APIKeys)

    panelProxy, err := gateway.KorisProxy(cfg.KorisPanelURL, authMw)
    if err != nil {
        logger.Error("gateway", "failed to create panel proxy", map[string]any{"error": err.Error()})
        os.Exit(1)
    }

    mux := http.NewServeMux()

    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","service":"gateway"}`))
    })
    mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ready","service":"gateway"}`))
    })

    apiMux := http.NewServeMux()
    apiMux.Handle("/", panelProxy)
    mux.Handle("/", authMw.RequireAPIKey(rateLimiter.Middleware(apiMux)))

    server := &http.Server{Addr: cfg.ListenAddr, Handler: mux}

    if cfg.TLSMode == "acme" && cfg.TLSDomain != "" {
        certManager := autocert.Manager{
            Prompt:      autocert.AcceptTOS,
            HostPolicy: autocert.HostWhitelist(cfg.TLSDomain),
            Email:      cfg.TLSEmail,
            Cache:     autocert.DirCache("/var/lib/gateway/acme"),
        }
        server.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
        go func() {
            redirect := http.Server{Addr: ":80", Handler: certManager.HTTPHandler(nil)}
            redirect.ListenAndServe()
        }()
        logger.Info("gateway", "ACME mode", map[string]any{"domain": cfg.TLSDomain})
        if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
            logger.Error("gateway", "server error", map[string]any{"error": err.Error()})
            os.Exit(1)
        }
    } else if cfg.TLSEnabled || cfg.TLSMode == "selfsigned" || cfg.TLSMode == "manual" {
        if cfg.TLSCert == "" || cfg.TLSKey == "" {
            logger.Error("gateway", "TLS enabled but cert/key not configured", nil)
            os.Exit(1)
        }
        logger.Info("gateway", "TLS mode", map[string]any{"cert": cfg.TLSCert})
        if err := server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil && err != http.ErrServerClosed {
            logger.Error("gateway", "server error", map[string]any{"error": err.Error()})
            os.Exit(1)
        }
    } else {
        logger.Info("gateway", "HTTP mode (internal only)", nil)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("gateway", "server error", map[string]any{"error": err.Error()})
            os.Exit(1)
        }
    }

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
}