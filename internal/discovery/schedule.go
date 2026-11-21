package discovery

import (
	"context"
	"sync"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
)

// DefaultInterval is the periodic query interval while visible (PRD §5.2).
const DefaultInterval = protocol.DiscoveryInterval

// DefaultCooldown is the minimum time between two Refresh queries.
const DefaultCooldown = 2 * time.Second

// Schedule decides when a DISCOVER query is due (PRD §5.2). Queries run only
// while the application is visible and not suspended by a transfer. The
// counter restarts after every query, including Refresh.
type Schedule struct {
	interval  time.Duration
	cooldown  time.Duration
	visible   bool
	suspended bool
	last      time.Time
}

// NewSchedule creates a schedule with the given periodic interval and refresh cooldown.
func NewSchedule(interval, cooldown time.Duration) *Schedule {
	return &Schedule{interval: interval, cooldown: cooldown}
}

// SetVisible records the visibility of the application. It reports true when
// a query must be sent now, which happens when the application becomes visible.
func (s *Schedule) SetVisible(now time.Time, visible bool) bool {
	becoming := visible && !s.visible
	s.visible = visible
	if becoming && !s.suspended {
		s.last = now
		return true
	}
	return false
}

// SetSuspended pauses or resumes periodic queries, for example during a transfer.
func (s *Schedule) SetSuspended(suspended bool) {
	s.suspended = suspended
}

// Due reports whether a periodic query is due at now, and records it if so.
func (s *Schedule) Due(now time.Time) bool {
	if !s.active() {
		return false
	}
	if !s.last.IsZero() && now.Sub(s.last) < s.interval {
		return false
	}
	s.last = now
	return true
}

// Refresh validates a manual refresh and records it. The caller must send the
// query when Refresh returns nil.
func (s *Schedule) Refresh(now time.Time) error {
	if !s.active() {
		return apperr.ErrUnauthorized
	}
	if !s.last.IsZero() && now.Sub(s.last) < s.cooldown {
		return apperr.ErrRateLimited
	}
	s.last = now
	return nil
}

// NextWake returns when the next periodic query is due, or false if none is scheduled.
func (s *Schedule) NextWake(now time.Time) (time.Time, bool) {
	if !s.active() {
		return time.Time{}, false
	}
	if s.last.IsZero() {
		return now, true
	}
	return s.last.Add(s.interval), true
}

func (s *Schedule) active() bool {
	return s.visible && !s.suspended
}

// Controller applies a Schedule to a query function and runs the periodic timer.
type Controller struct {
	mu      sync.Mutex
	sched   *Schedule
	query   func(time.Time) error
	onError func(error)
	now     func() time.Time
	wakeCh  chan struct{}
}

// NewController creates a controller that calls query for each due query.
// Errors from periodic queries are reported to onError.
func NewController(query func(time.Time) error, onError func(error), interval, cooldown time.Duration) *Controller {
	return &Controller{
		sched:   NewSchedule(interval, cooldown),
		query:   query,
		onError: onError,
		now:     time.Now,
		wakeCh:  make(chan struct{}, 1),
	}
}

// SetVisible updates visibility and queries immediately when the app becomes visible.
func (c *Controller) SetVisible(visible bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	send := c.sched.SetVisible(now, visible)
	c.notify()
	if send {
		return c.query(now)
	}
	return nil
}

// SetSuspended pauses or resumes periodic queries.
func (c *Controller) SetSuspended(suspended bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sched.SetSuspended(suspended)
	c.notify()
}

// Refresh handles a user click on Refresh, subject to the cooldown.
func (c *Controller) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if err := c.sched.Refresh(now); err != nil {
		return err
	}
	c.notify()
	return c.query(now)
}

// Run sends periodic queries until ctx ends.
func (c *Controller) Run(ctx context.Context) {
	for {
		c.mu.Lock()
		wake, ok := c.sched.NextWake(c.now())
		c.mu.Unlock()

		var timerC <-chan time.Time
		var timer *time.Timer
		if ok {
			timer = time.NewTimer(time.Until(wake))
			timerC = timer.C
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-timerC:
			c.tick()
		case <-c.wakeCh:
			if timer != nil {
				timer.Stop()
			}
		}
	}
}

func (c *Controller) tick() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.sched.Due(now) {
		if err := c.query(now); err != nil {
			c.onError(err)
		}
	}
}

func (c *Controller) notify() {
	select {
	case c.wakeCh <- struct{}{}:
	default:
	}
}
