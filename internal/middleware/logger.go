package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"fuck-watermark/logs"
)

// RequestLogger 使用项目 logs 组件记录访问日志。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		if query != "" {
			path = path + "?" + query
		}
		logs.Infof("[访问] 状态码=%d 方法=%s 路径=%q 耗时=%s 客户端=%s",
			status, c.Request.Method, path, latency, c.ClientIP())
	}
}
