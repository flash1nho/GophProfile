package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type client struct {
	tokens int
	last   time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	rate    int
	burst   int
}

func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*client),
		rate:    rate,
		burst:   burst,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		rl.mu.Lock()
		c, exists := rl.clients[ip]
		if !exists {
			c = &client{tokens: rl.burst, last: time.Now()}
			rl.clients[ip] = c
		}

		now := time.Now()
		elapsed := now.Sub(c.last).Seconds()
		c.tokens += int(elapsed * float64(rl.rate))
		if c.tokens > rl.burst {
			c.tokens = rl.burst
		}
		c.last = now

		if c.tokens <= 0 {
			rl.mu.Unlock()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		c.tokens--
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
