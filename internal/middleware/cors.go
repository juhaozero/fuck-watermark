package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowOrigins []string) gin.HandlerFunc {
	origins := normalizeOrigins(allowOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := pickOrigin(origin, origins)
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func normalizeOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{"*"}
	}
	out := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func pickOrigin(requestOrigin string, allowOrigins []string) string {
	for _, allowed := range allowOrigins {
		if allowed == "*" {
			if requestOrigin != "" {
				return requestOrigin
			}
			return "*"
		}
		if strings.EqualFold(requestOrigin, allowed) {
			return allowed
		}
	}
	return ""
}
