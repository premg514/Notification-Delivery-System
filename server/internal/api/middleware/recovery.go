package middleware

import (
	"log/slog"
	"net/http"

	"notification-system/internal/observability"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic recovered in HTTP request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"panic", recovered,
		)
		observability.IncHTTPPanics()

		if !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
