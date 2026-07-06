package middleware

import (
	"sync"
	"time"

	"golang.org/x/time/rate"

	"fuck-watermark/internal/config"
)

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	cfg      config.RateLimitConfig
}

func newIPRateLimiter(cfg config.RateLimitConfig) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		cfg:      cfg,
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	limiter := l.getLimiter(ip)
	return limiter.Allow()
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, ok := l.limiters[ip]; ok {
		return limiter
	}

	rpm := l.cfg.RequestsPerMinute
	if rpm <= 0 {
		rpm = 60
	}
	burst := l.cfg.Burst
	if burst <= 0 {
		burst = 10
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rpm)), burst)
	l.limiters[ip] = limiter
	return limiter
}
