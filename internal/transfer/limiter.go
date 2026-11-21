package transfer

// Limiter bounds the number of simultaneous transfers (task 025).
type Limiter struct {
	slots chan struct{}
}

// NewLimiter allows up to n concurrent transfers.
func NewLimiter(n int) *Limiter {
	return &Limiter{slots: make(chan struct{}, n)}
}

// TryAcquire takes a slot without blocking. It reports false when all are in use.
func (l *Limiter) TryAcquire() bool {
	select {
	case l.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release frees a slot taken by TryAcquire.
func (l *Limiter) Release() {
	<-l.slots
}

// InUse returns how many slots are taken.
func (l *Limiter) InUse() int {
	return len(l.slots)
}
