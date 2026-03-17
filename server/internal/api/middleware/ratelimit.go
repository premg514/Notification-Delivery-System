package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count     int
	windowEnd time.Time
}

type RateLimiter struct {
	limit int
	mu    sync.Mutex
	store map[string]*visitor
}

func NewRateLimiter(limit int) *RateLimiter {
	rl := &RateLimiter{
		limit: limit,
		store: make(map[string]*visitor),
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.store[key]
	if !exists || now.After(entry.windowEnd) {
		rl.store[key] = &visitor{
			count:     1,
			windowEnd: now.Add(time.Minute),
		}
		return true
	}

	if entry.count >= rl.limit {
		return false
	}

	entry.count++
	return true
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		rl.mu.Lock()
		for key, entry := range rl.store {
			if now.After(entry.windowEnd) {
				delete(rl.store, key)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
