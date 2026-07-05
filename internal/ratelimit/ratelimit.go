package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

type Limiter struct {
	visitors       map[string]*visitor
	mu             sync.Mutex
	rate           float64 // tokens per second
	burst          int
	trustedProxies map[string]bool
	trustedCIDRs   []*net.IPNet
	done           chan struct{}
}

func New(rate float64, burst int, trustedProxies []string) *Limiter {
	proxies := make(map[string]bool)
	var cidrs []*net.IPNet
	for _, p := range trustedProxies {
		if _, cidr, err := net.ParseCIDR(p); err == nil {
			cidrs = append(cidrs, cidr)
		} else {
			proxies[p] = true
		}
	}
	l := &Limiter{
		visitors:       make(map[string]*visitor),
		rate:           rate,
		burst:          burst,
		trustedProxies: proxies,
		trustedCIDRs:   cidrs,
		done:           make(chan struct{}),
	}
	go l.cleanup()
	return l
}

// Stop signals the cleanup goroutine to exit.
func (l *Limiter) Stop() {
	close(l.done)
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			for ip, v := range l.visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(l.visitors, ip)
				}
			}
			l.mu.Unlock()
		case <-l.done:
			return
		}
	}
}

func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, exists := l.visitors[ip]
	if !exists {
		l.visitors[ip] = &visitor{tokens: float64(l.burst) - 1, lastSeen: time.Now()}
		return true
	}
	elapsed := time.Since(v.lastSeen).Seconds()
	v.tokens += elapsed * l.rate
	if v.tokens > float64(l.burst) {
		v.tokens = float64(l.burst)
	}
	v.lastSeen = time.Now()
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

func (l *Limiter) isTrustedProxy(ip string) bool {
	if l.trustedProxies[ip] {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range l.trustedCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func (l *Limiter) clientIP(r *http.Request) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	if l.isTrustedProxy(remoteIP) {
		if fwd := r.Header.Get("X-Real-IP"); fwd != "" {
			if ip := net.ParseIP(fwd); ip != nil {
				return fwd
			}
		}
	}
	return remoteIP
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := l.clientIP(r)
		if !l.Allow(ip) {
			http.Error(w, `{"ok":false,"error":"rate_limit_exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
