package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	// Database
	DBDSN string

	// gRPC
	GRPCAddr string

	// Security
	APIKey string

	// Concurrency
	WorkerConcurrency int

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPEnabled  bool

	// Telegram
	TelegramBotToken string
	TelegramChatID   string
	TelegramEnabled  bool
}

// LoadWorker loads worker configuration from environment
func LoadWorker() WorkerConfig {
	concurrency := 4
	if c := os.Getenv("WORKER_CONCURRENCY"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 0 {
			concurrency = v
		}
	}

	return WorkerConfig{
		DBDSN:             getEnv("WORKER_PG_DSN", "postgres://koris:@localhost:5432/koris?sslmode=disable"),
		GRPCAddr:          getEnv("WORKER_GRPC_ADDR", "0.0.0.0:2026"),
		APIKey:            os.Getenv("WORKER_API_KEY"),
		WorkerConcurrency: concurrency,
		SMTPHost:         os.Getenv("PANEL_SMTP_HOST"),
		SMTPPort:         os.Getenv("PANEL_SMTP_PORT"),
		SMTPUser:         os.Getenv("PANEL_SMTP_USER"),
		SMTPPass:         os.Getenv("PANEL_SMTP_PASS"),
		SMTPFrom:         os.Getenv("PANEL_SMTP_FROM"),
		SMTPEnabled:      strings.ToLower(os.Getenv("PANEL_SMTP_ENABLED")) == "true",
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		TelegramEnabled:  strings.ToLower(os.Getenv("TELEGRAM_ENABLED")) == "true",
	}
}

// GatewayConfig holds gateway-specific configuration
type GatewayConfig struct {
	ListenAddr      string
	GRPCAddr        string
	APIKeys         []string
	TLSMode         string
	TLSCert         string
	TLSKey          string
	TLSEmail        string
	TLSDomain       string
	PanelURL        string
	WorkerGRPCAddr  string
	RateLimitRPS    float64
	RateLimitBurst  int
	TrustedProxies  []string
	LogFormat       string
}

// LoadGateway loads gateway configuration from environment
func LoadGateway() GatewayConfig {
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

	burst := 200
	if b := os.Getenv("GATEWAY_RATE_LIMIT_BURST"); b != "" {
		if v, err := strconv.Atoi(b); err == nil && v > 0 {
			burst = v
		}
	}

	rate := 100.0
	if r := os.Getenv("GATEWAY_RATE_LIMIT_RPS"); r != "" {
		if v, err := strconv.ParseFloat(r, 64); err == nil && v > 0 {
			rate = v
		}
	}

	return GatewayConfig{
		ListenAddr:     getEnv("GATEWAY_LISTEN_ADDR", ":443"),
		GRPCAddr:       getEnv("GATEWAY_GRPC_ADDR", ":2025"),
		APIKeys:        apiKeys,
		TLSMode:        getEnv("GATEWAY_TLS_MODE", "selfsigned"),
		TLSCert:        getEnv("GATEWAY_TLS_CERT", ""),
		TLSKey:         getEnv("GATEWAY_TLS_KEY", ""),
		TLSEmail:       getEnv("GATEWAY_TLS_EMAIL", ""),
		TLSDomain:      getEnv("GATEWAY_TLS_DOMAIN", ""),
		PanelURL:       getEnv("GATEWAY_KORIS_PANEL_URL", "http://panel:8080"),
		WorkerGRPCAddr: getEnv("GATEWAY_WORKER_GRPC_ADDR", "worker:2026"),
		RateLimitRPS:   rate,
		RateLimitBurst: burst,
		TrustedProxies: trusted,
		LogFormat:      getEnv("GATEWAY_LOG_FORMAT", "json"),
	}
}

// gRPC configuration
func (c *Config) GetGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		ConnectTimeout:    5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
		MetricsInterval:   60 * time.Second,
	}
}

// GRPCConfig holds gRPC client configuration
type GRPCConfig struct {
	ConnectTimeout    time.Duration
	KeepaliveInterval time.Duration
	MetricsInterval   time.Duration
}
