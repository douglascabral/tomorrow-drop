package discovery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

type queryCounter struct {
	mu sync.Mutex
	n  int
}

func (q *queryCounter) query(time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.n++
	return nil
}

func (q *queryCounter) count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.n
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

func TestControllerQueriesImmediatelyThenPeriodically(t *testing.T) {
	var q queryCounter
	c := NewController(q.query, func(error) {}, 30*time.Millisecond, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	if err := c.SetVisible(true); err != nil {
		t.Fatalf("SetVisible: %v", err)
	}
	if q.count() != 1 {
		t.Fatalf("no immediate query on becoming visible")
	}
	waitUntil(t, "periodic queries", func() bool { return q.count() >= 3 })
}

func TestControllerSuspensionStopsPeriodicQueries(t *testing.T) {
	var q queryCounter
	c := NewController(q.query, func(error) {}, 30*time.Millisecond, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	_ = c.SetVisible(true)
	waitUntil(t, "first periodic query", func() bool { return q.count() >= 2 })

	c.SetSuspended(true)
	time.Sleep(40 * time.Millisecond)
	frozen := q.count()
	time.Sleep(150 * time.Millisecond)
	if q.count() != frozen {
		t.Fatalf("queries sent while suspended: %d -> %d", frozen, q.count())
	}

	c.SetSuspended(false)
	waitUntil(t, "queries after resume", func() bool { return q.count() > frozen })
}

func TestControllerHiddenStopsEverything(t *testing.T) {
	var q queryCounter
	c := NewController(q.query, func(error) {}, 30*time.Millisecond, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	_ = c.SetVisible(true)
	_ = c.SetVisible(false)

	time.Sleep(40 * time.Millisecond)
	frozen := q.count()
	time.Sleep(150 * time.Millisecond)
	if q.count() != frozen {
		t.Fatalf("queries sent while hidden: %d -> %d", frozen, q.count())
	}
}

func TestControllerRefreshIsRateLimited(t *testing.T) {
	var q queryCounter
	c := NewController(q.query, func(error) {}, time.Hour, time.Hour)
	_ = c.SetVisible(true)
	if err := c.Refresh(); !errors.Is(err, apperr.ErrRateLimited) {
		t.Fatalf("Refresh within cooldown = %v, want ErrRateLimited", err)
	}
	if q.count() != 1 {
		t.Fatalf("rate-limited Refresh still sent a query")
	}
}
