package middleware

import (
	"time"

	"notification-system/internal/observability"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		observability.ObserveHTTPRequest(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}
