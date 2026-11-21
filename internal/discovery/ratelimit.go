package discovery

import (
	"sync"
	"time"
)

// RateLimiter allows at most max requests per origin within window. It
// tracks at most capacity origins, evicting the least recently active one.
type RateLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	capacity int
	hits     map[string][]time.Time
}

// NewRateLimiter creates a limiter.
func NewRateLimiter(max int, window time.Duration, capacity int) *RateLimiter {
	return &RateLimiter{max: max, window: window, capacity: capacity, hits: map[string][]time.Time{}}
}

// Allow records a request from origin at now and reports whether it is within the limit.
func (r *RateLimiter) Allow(origin string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	recent := r.hits[origin][:0]
	for _, h := range r.hits[origin] {
		if now.Sub(h) < r.window {
			recent = append(recent, h)
		}
	}
	if len(recent) >= r.max {
		r.hits[origin] = recent
		return false
	}
	if _, known := r.hits[origin]; !known && len(r.hits) >= r.capacity {
		r.evictLeastRecent()
	}
	r.hits[origin] = append(recent, now)
	return true
}

func (r *RateLimiter) tracked() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.hits)
}

func (r *RateLimiter) evictLeastRecent() {
	var oldestKey string
	var oldest time.Time
	first := true
	for key, times := range r.hits {
		if len(times) == 0 {
			delete(r.hits, key)
			return
		}
		last := times[len(times)-1]
		if first || last.Before(oldest) {
			oldestKey, oldest, first = key, last, false
		}
	}
	delete(r.hits, oldestKey)
}
