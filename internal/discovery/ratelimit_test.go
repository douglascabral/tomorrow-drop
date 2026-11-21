package discovery

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToLimitPerWindow(t *testing.T) {
	rl := NewRateLimiter(3, 10*time.Second, 16)
	now := time.Unix(1000, 0)
	for i := 0; i < 3; i++ {
		if !rl.Allow("10.0.0.1", now) {
			t.Fatalf("request %d rejected within the limit", i+1)
		}
	}
	if rl.Allow("10.0.0.1", now) {
		t.Fatalf("request above the limit was allowed")
	}
}

func TestRateLimiterIsPerOrigin(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Second, 16)
	now := time.Unix(1000, 0)
	if !rl.Allow("10.0.0.1", now) || !rl.Allow("10.0.0.2", now) {
		t.Fatalf("a request from one origin blocked another origin")
	}
}

func TestRateLimiterRecoversAfterWindow(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Second, 16)
	now := time.Unix(1000, 0)
	rl.Allow("10.0.0.1", now)
	if rl.Allow("10.0.0.1", now.Add(5*time.Second)) {
		t.Fatalf("request inside the window was allowed")
	}
	if !rl.Allow("10.0.0.1", now.Add(11*time.Second)) {
		t.Fatalf("origin not allowed again after the window")
	}
}

func TestRateLimiterBoundsTrackedOrigins(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Second, 4)
	now := time.Unix(1000, 0)
	for i := 0; i < 100; i++ {
		rl.Allow(string(rune('a'+i%26))+string(rune('a'+i/26)), now.Add(time.Duration(i)*time.Millisecond))
	}
	if n := rl.tracked(); n > 4 {
		t.Fatalf("tracked origins = %d, want at most 4", n)
	}
}
