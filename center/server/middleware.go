package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

type contextKey string

const reqIDKey contextKey = "reqId"

// statusWriter wraps ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// withLogging logs every request with method, path, status, duration.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newID("req")
		ctx := context.WithValue(r.Context(), reqIDKey, reqID)
		r = r.WithContext(ctx)

		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)

		logInfo("request",
			func(e *LogEntry) {
				e.ReqID = reqID
				e.Method = r.Method
				e.Path = r.URL.Path
				e.Status = sw.status
				e.Dur = time.Since(start).String()
			},
		)
	})
}

// withMaxBody limits request body size.
func withMaxBody(limit int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next(w, r)
	}
}

// ====== rate limiter (token bucket per IP) ======

type rateBucket struct {
	tokens    float64
	lastTime  time.Time
	ratePerSec float64
	burst     float64
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	rate    float64
	burst   float64
}

func NewRateLimiter(ratePerSec, burst float64) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*rateBucket),
		rate:    ratePerSec,
		burst:   burst,
	}
	// Cleanup stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	now := time.Now()
	if !ok {
		rl.buckets[ip] = &rateBucket{
			tokens:    rl.burst - 1,
			lastTime:  now,
			ratePerSec: rl.rate,
			burst:     rl.burst,
		}
		return true
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * b.ratePerSec
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, b := range rl.buckets {
		if b.lastTime.Before(cutoff) {
			delete(rl.buckets, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func withRateLimit(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
