package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	limiters sync.Map // map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
	cleanup  *time.Ticker
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:    rate.Limit(requestsPerSecond),
		burst:   burst,
		cleanup: time.NewTicker(time.Minute),
	}

	// Start cleanup goroutine to remove old limiters
	go rl.cleanupOldLimiters()

	return rl
}

// getLimiter gets or creates a rate limiter for the given key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	limiter, exists := rl.limiters.Load(key)
	if !exists {
		newLimiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Store(key, newLimiter)
		return newLimiter
	}
	return limiter.(*rate.Limiter)
}

// cleanupOldLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) cleanupOldLimiters() {
	for {
		<-rl.cleanup.C
		// Remove limiters that don't have any remaining tokens and haven't been used
		rl.limiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rate.Limiter)
			// If limiter has been idle and has tokens available, it's safe to remove
			if limiter.Tokens() == float64(rl.burst) {
				rl.limiters.Delete(key)
			}
			return true
		})
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

// RateLimit returns a middleware that limits requests per IP
func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "10") // requests per second
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded. Please try again later."}`))
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", "10")
		w.Header().Set("X-RateLimit-Remaining", "9") // Approximate remaining

		next.ServeHTTP(w, r)
	})
}

// Close stops the cleanup ticker
func (rl *RateLimiter) Close() {
	rl.cleanup.Stop()
}

// StrictRateLimit returns a more restrictive rate limiter for sensitive endpoints
func (rl *RateLimiter) StrictRateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	strictLimiter := NewRateLimiter(float64(requestsPerMinute)/60.0, 1) // Convert per minute to per second

	return func(next http.Handler) http.Handler {
		return strictLimiter.RateLimit(next)
	}
}

// AuthRateLimit provides specific rate limiting for authentication endpoints
func NewAuthRateLimiter() *RateLimiter {
	// Allow 5 login attempts per minute per IP
	return NewRateLimiter(5.0/60.0, 5)
}

// APIRateLimit provides general API rate limiting
func NewAPIRateLimiter() *RateLimiter {
	// Allow 10 requests per second per IP with burst of 20
	return NewRateLimiter(10.0, 20)
}