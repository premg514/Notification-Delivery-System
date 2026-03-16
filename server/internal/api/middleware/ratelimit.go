package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var clients = make(map[string]time.Time)
var mu sync.Mutex

func RateLimiter() gin.HandlerFunc {

	return func(c *gin.Context) {

		ip := c.ClientIP()

		mu.Lock()

		lastRequest, exists := clients[ip]

		if exists && time.Since(lastRequest) < time.Second {

			mu.Unlock()

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})

			c.Abort()
			return
		}

		clients[ip] = time.Now()

		mu.Unlock()

		c.Next()
	}
}