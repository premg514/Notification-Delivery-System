package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	limit  int64
	window time.Duration
	client *redis.Client
	script *redis.Script
}

func NewRateLimiter(limit int, redisURL string) (*RateLimiter, error) {
	if limit <= 0 {
		limit = 1
	}

	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &RateLimiter{
		limit:  int64(limit),
		window: time.Minute,
		client: client,
		script: redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`),
	}, nil
}

func (rl *RateLimiter) Close() error {
	if rl.client == nil {
		return nil
	}
	return rl.client.Close()
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, err := rl.allow(c.Request.Context(), rl.keyFor(c.ClientIP()))
		if err != nil {
			c.JSON(503, map[string]string{"error": "rate limiter unavailable"})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(429, map[string]string{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) allow(ctx context.Context, key string) (bool, error) {
	current, err := rl.script.Run(ctx, rl.client, []string{key}, int(rl.window.Milliseconds())).Int64()
	if err != nil {
		return false, err
	}

	return current <= rl.limit, nil
}

func (rl *RateLimiter) keyFor(client string) string {
	windowKey := time.Now().UTC().Unix() / int64(rl.window.Seconds())
	return fmt.Sprintf("ratelimit:%s:%d", client, windowKey)
}
