package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// UserRateLimiter applies a per-user token-bucket rate limit. Unlike RateLimiter
// (keyed by IP), this middleware keys on the authenticated user ID pulled from
// the request context, so it MUST be mounted after Auth middleware.
// Intended for expensive endpoints like RAG where per-user cost control matters.
type UserRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
}

func NewUserRateLimiter(rps float64, burst int) *UserRateLimiter {
	rl := &UserRateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *UserRateLimiter) getVisitor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rl.rps, rl.burst)
		rl.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *UserRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for k, v := range rl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *UserRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			// No authenticated user — let the downstream Auth check reject.
			next.ServeHTTP(w, r)
			return
		}
		limiter := rl.getVisitor(userID.String())
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
