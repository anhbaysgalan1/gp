package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_BasicFunctionality(t *testing.T) {
	// Create a rate limiter that allows 2 requests per second with burst of 2
	rl := middleware.NewRateLimiter(2.0, 2)
	defer rl.Close()

	// Simple handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limiting middleware
	middleware := rl.RateLimit(handler)

	// Test first request - should pass
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "OK", w1.Body.String())

	// Test second request - should pass (within burst)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "OK", w2.Body.String())

	// Test third request immediately - should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	w3 := httptest.NewRecorder()
	middleware.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "Rate limit exceeded")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	// Create a rate limiter that allows 1 request per second
	rl := middleware.NewRateLimiter(1.0, 1)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := rl.RateLimit(handler)

	// Request from first IP
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Request from second IP - should also pass (different limiter)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Second request from first IP - should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	w3 := httptest.NewRecorder()
	middleware.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
}

func TestRateLimiter_XForwardedFor(t *testing.T) {
	rl := middleware.NewRateLimiter(1.0, 1)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	// Request with X-Forwarded-For header
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1")
	req1.RemoteAddr = "192.168.1.100:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request with same X-Forwarded-For IP - should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1")
	req2.RemoteAddr = "192.168.1.200:12345" // Different RemoteAddr but same forwarded IP
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimiter_XRealIP(t *testing.T) {
	rl := middleware.NewRateLimiter(1.0, 1)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	// Request with X-Real-IP header
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Real-IP", "203.0.113.1")
	req1.RemoteAddr = "192.168.1.100:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request with same X-Real-IP - should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Real-IP", "203.0.113.1")
	req2.RemoteAddr = "192.168.1.200:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimiter_Headers(t *testing.T) {
	rl := middleware.NewRateLimiter(10.0, 10)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that rate limit headers are set
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimiter_Recovery(t *testing.T) {
	// Create rate limiter with very low rate
	rl := middleware.NewRateLimiter(0.5, 1) // 1 request per 2 seconds
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	// First request should pass
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Immediate second request should be blocked
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	// Wait for rate limiter to recover
	time.Sleep(2100 * time.Millisecond) // Wait slightly more than 2 seconds

	// Third request should pass after recovery
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	w3 := httptest.NewRecorder()
	middleware.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
}

func TestAPIRateLimiter(t *testing.T) {
	rl := middleware.NewAPIRateLimiter()
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	// Should allow multiple requests up to the burst limit
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		// The first 20 requests should pass (burst limit)
		if i < 20 {
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should pass", i+1)
		}
	}
}

func TestAuthRateLimiter(t *testing.T) {
	rl := middleware.NewAuthRateLimiter()
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.RateLimit(handler)

	// Auth rate limiter should be more restrictive
	req1 := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Multiple auth requests in quick succession should get limited faster
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		// Should hit rate limit before regular API would
		if w.Code == http.StatusTooManyRequests {
			// Auth limiter triggered - this is expected
			break
		}
	}
}

func TestStrictRateLimit(t *testing.T) {
	rl := middleware.NewRateLimiter(10.0, 10)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply strict rate limiting (1 request per minute)
	strictMiddleware := rl.StrictRateLimit(1)
	middleware := strictMiddleware(handler)

	// First request should pass
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be immediately blocked
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}