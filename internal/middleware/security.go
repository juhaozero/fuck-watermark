package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"fuck-watermark/internal/config"
	"fuck-watermark/internal/model"
)

func APIKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if token == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token = strings.TrimSpace(auth[7:])
			}
		}

		if token != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Fail(401, "未授权访问"))
			return
		}
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

func RateLimit(cfg config.RateLimitConfig) gin.HandlerFunc {
	limiter := newIPRateLimiter(cfg)
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.Fail(429, "请求过于频繁，请稍后再试"))
			return
		}
		c.Next()
	}
}
