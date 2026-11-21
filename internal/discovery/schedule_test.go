package discovery

import (
	"errors"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
)

func TestScheduleQueriesImmediatelyWhenBecomingVisible(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	if !s.SetVisible(now, true) {
		t.Fatalf("no immediate query when the app became visible")
	}
	if s.SetVisible(now, true) {
		t.Fatalf("visible state change should not query again")
	}
}

func TestSchedulePeriodicQueryEvery30Seconds(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	if s.Due(now.Add(29 * time.Second)) {
		t.Fatalf("periodic query before 30 seconds")
	}
	if !s.Due(now.Add(30 * time.Second)) {
		t.Fatalf("no periodic query at 30 seconds")
	}
}

func TestScheduleNoQueriesWhileHidden(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	s.SetVisible(now, false)
	if s.Due(now.Add(time.Hour)) {
		t.Fatalf("periodic query while hidden")
	}
	if _, ok := s.NextWake(now); ok {
		t.Fatalf("a wake-up was scheduled while hidden")
	}
}

func TestScheduleSuspendedDuringTransfer(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	s.SetSuspended(true)
	if s.Due(now.Add(time.Minute)) {
		t.Fatalf("periodic query during a transfer")
	}
	s.SetSuspended(false)
	if !s.Due(now.Add(time.Minute)) {
		t.Fatalf("periodic queries did not resume after the transfer")
	}
}

func TestRefreshRestartsCounter(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	clicked := now.Add(20 * time.Second)
	if err := s.Refresh(clicked); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if s.Due(now.Add(30 * time.Second)) {
		t.Fatalf("counter was not restarted by Refresh")
	}
	if !s.Due(clicked.Add(30 * time.Second)) {
		t.Fatalf("no periodic query 30 seconds after Refresh")
	}
}

func TestRefreshIsRateLimited(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	if err := s.Refresh(now.Add(3 * time.Second)); err != nil {
		t.Fatalf("first Refresh: %v", err)
	}
	if err := s.Refresh(now.Add(4 * time.Second)); !errors.Is(err, apperr.ErrRateLimited) {
		t.Fatalf("repeated Refresh = %v, want ErrRateLimited", err)
	}
}

func TestRefreshRequiresVisibility(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	if err := s.Refresh(time.Unix(1000, 0)); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Refresh while hidden = %v, want ErrUnauthorized", err)
	}
}

func TestNextWakeReportsNextPeriodicQuery(t *testing.T) {
	s := NewSchedule(30*time.Second, 2*time.Second)
	now := time.Unix(1000, 0)
	s.SetVisible(now, true)
	wake, ok := s.NextWake(now.Add(5 * time.Second))
	if !ok || !wake.Equal(now.Add(30*time.Second)) {
		t.Fatalf("NextWake = %v (%v), want %v", wake, ok, now.Add(30*time.Second))
	}
}
