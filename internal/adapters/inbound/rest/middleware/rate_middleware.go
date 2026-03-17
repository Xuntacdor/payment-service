package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new map of rate limiters per IP.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// Optional: You could run a cleanup routine to drop stale IP keys
	// over time to avoid unbound memory growth.

	return limiter
}

// GetLimiter returns the rate limiter for a specific IP.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		limiter, exists = i.ips[ip]
		if !exists {
			limiter = rate.NewLimiter(i.r, i.b)
			i.ips[ip] = limiter
		}
		i.mu.Unlock()
	}
	return limiter
}

// RateLimitMiddleware enforces limits based on client IP.
func RateLimitMiddleware(requestsPerSec float64, burstSize int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Limit(requestsPerSec), burstSize)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		l := limiter.GetLimiter(clientIP)

		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}
		c.Next()
	}
}
