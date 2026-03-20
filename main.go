package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	capacity   int
	tokens     float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimiter manages per-IP token buckets.
type RateLimiter struct {
	buckets    map[string]*TokenBucket
	mu         sync.Mutex
	capacity   int
	refillRate float64
}

func NewRateLimiter(capacity int, refillRate float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getBucket(ip string) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, exists := rl.buckets[ip]; !exists {
		rl.buckets[ip] = NewTokenBucket(rl.capacity, rl.refillRate)
	}
	return rl.buckets[ip]
}

// cleanup removes stale buckets every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		rl.buckets = make(map[string]*TokenBucket)
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		bucket := rl.getBucket(ip)

		if !bucket.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "rate limit exceeded",
				"message": "Too many requests. Please slow down.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Example Handlers ---

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello! You're within the rate limit.",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	// 10 requests per IP, refilling at 2 tokens/second
	limiter := NewRateLimiter(10, 2)

	mux := http.NewServeMux()
	mux.HandleFunc("/", helloHandler)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      limiter.Middleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("🚀 Server running at http://localhost:8080")
	fmt.Println("   Rate limit: 10 requests per IP, 2 tokens/sec refill")
	log.Fatal(server.ListenAndServe())
}
