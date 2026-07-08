package gateway

import (
    "os"
    "strconv"
    "strings"
)

type Config struct {
    ListenAddr     string
    TLSEnabled     bool
    TLSCert        string
    TLSKey         string
    TLSMode        string
    TLSEmail       string
    TLSDomain      string
    KorisPanelURL  string
    WorkerGRPCAddr string
    RateLimitRPS   float64
    RateLimitBurst int
    TrustedProxies []string
    APIKeys        []string
    LogFormat      string
}

func Load() Config {
    apiKeysRaw := os.Getenv("GATEWAY_API_KEYS")
    var apiKeys []string
    if apiKeysRaw != "" {
        for _, k := range strings.Split(apiKeysRaw, ",") {
            k = strings.TrimSpace(k)
            if k != "" {
                apiKeys = append(apiKeys, k)
            }
        }
    }
    trustedRaw := os.Getenv("GATEWAY_TRUSTED_PROXIES")
    var trusted []string
    if trustedRaw != "" {
        for _, p := range strings.Split(trustedRaw, ",") {
            p = strings.TrimSpace(p)
            if p != "" {
                trusted = append(trusted, p)
            }
        }
    }
    burst := 20
    if b := os.Getenv("GATEWAY_RATE_LIMIT_BURST"); b != "" {
        if v, err := strconv.Atoi(b); err == nil && v > 0 {
            burst = v
        }
    }
    rate := 10.0
    if r := os.Getenv("GATEWAY_RATE_LIMIT_RPS"); r != "" {
        if v, err := strconv.ParseFloat(r, 64); err == nil && v > 0 {
            rate = v
        }
    }
    return Config{
        ListenAddr:     getEnv("GATEWAY_LISTEN_ADDR", ":8080"),
        TLSEnabled:     os.Getenv("GATEWAY_TLS_ENABLED") == "true",
        TLSCert:        getEnv("GATEWAY_TLS_CERT", "/etc/koris/gateway-cert.pem"),
        TLSKey:         getEnv("GATEWAY_TLS_KEY", "/etc/koris/gateway-key.pem"),
        TLSMode:        getEnv("GATEWAY_TLS_MODE", "selfsigned"),
        TLSEmail:       getEnv("GATEWAY_TLS_EMAIL", ""),
        TLSDomain:      getEnv("GATEWAY_TLS_DOMAIN", ""),
        KorisPanelURL:  getEnv("GATEWAY_KORIS_URL", "http://127.0.0.1:2026"),
        WorkerGRPCAddr: getEnv("GATEWAY_WORKER_GRPC", "127.0.0.1:2027"),
        RateLimitRPS:   rate,
        RateLimitBurst: burst,
        TrustedProxies: trusted,
        APIKeys:        apiKeys,
        LogFormat:      getEnv("GATEWAY_LOG_FORMAT", "text"),
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}