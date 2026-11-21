package discovery

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

// pendingTTL is how long a query nonce accepts answers.
const pendingTTL = 10 * time.Second

// Scanner sends DISCOVER queries and collects ANNOUNCE answers. Only answers
// whose nonce matches a recent query are accepted.
type Scanner struct {
	conn    *net.UDPConn
	targets []netip.AddrPort

	mu      sync.Mutex
	pending map[string]time.Time
}

// NewScanner opens an ephemeral IPv4 socket that sends to targets.
func NewScanner(targets []netip.AddrPort) (*Scanner, error) {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("open discovery socket: %w", err)
	}
	return &Scanner{conn: conn, targets: targets, pending: map[string]time.Time{}}, nil
}

// LocalAddr returns the address answers are sent back to.
func (s *Scanner) LocalAddr() netip.AddrPort {
	return s.conn.LocalAddr().(*net.UDPAddr).AddrPort()
}

// Close releases the socket.
func (s *Scanner) Close() error {
	return s.conn.Close()
}

// Query sends one DISCOVER to every target. It fails with ErrNoRoute when there
// are no targets or every send fails.
func (s *Scanner) Query(now time.Time) error {
	if len(s.targets) == 0 {
		return apperr.ErrNoRoute
	}
	q, err := NewDiscover()
	if err != nil {
		return err
	}
	data, err := Encode(q)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.prune(now)
	s.pending[q.Nonce] = now
	s.mu.Unlock()

	failed := 0
	var lastErr error
	for _, target := range s.targets {
		if _, err := s.conn.WriteToUDPAddrPort(data, target); err != nil {
			failed++
			lastErr = err
		}
	}
	if failed == len(s.targets) {
		return fmt.Errorf("%w: %w", apperr.ErrNoRoute, lastErr)
	}
	return nil
}

// Run reads answers and sends accepted ones to out until ctx ends.
func (s *Scanner) Run(ctx context.Context, out chan<- Announcement) error {
	go func() {
		<-ctx.Done()
		_ = s.conn.Close()
	}()
	buf := make([]byte, limits.MaxDiscoveryDatagram+1)
	for {
		n, from, err := s.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read discovery answers: %w", err)
		}
		answer, err := Decode(buf[:n])
		if err != nil || answer.Type != TypeAnnounce {
			continue
		}
		now := time.Now()
		if !s.accepts(answer.Nonce, now) {
			continue
		}
		a := Announcement{
			Name: answer.Name,
			Addr: netip.AddrPortFrom(from.Addr(), uint16(answer.Port)), //nolint:gosec // G115: port validated to 1..65535 by Decode
			Seen: now,
		}
		select {
		case out <- a:
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Scanner) accepts(nonce string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sent, ok := s.pending[nonce]
	return ok && now.Sub(sent) < pendingTTL
}

func (s *Scanner) pendingCount(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	return len(s.pending)
}

// prune removes expired nonces. The caller must hold s.mu.
func (s *Scanner) prune(now time.Time) {
	for nonce, sent := range s.pending {
		if now.Sub(sent) >= pendingTTL {
			delete(s.pending, nonce)
		}
	}
}

// BroadcastTargets returns the discovery destinations for the given interface
// addresses: the directed broadcast of each IPv4 subnet, plus the limited
// broadcast address. Loopback and IPv6 addresses are skipped.
func BroadcastTargets(port int, addrs []net.Addr) []netip.AddrPort {
	seen := map[netip.AddrPort]bool{}
	var targets []netip.AddrPort
	add := func(a netip.AddrPort) {
		if !seen[a] {
			seen[a] = true
			targets = append(targets, a)
		}
	}
	add(netip.AddrPortFrom(netip.AddrFrom4([4]byte{255, 255, 255, 255}), uint16(port))) //nolint:gosec // G115: port validated by the caller
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil || ip4.IsLoopback() || len(ipNet.Mask) != net.IPv4len {
			continue
		}
		bcast := make(net.IP, net.IPv4len)
		for i := range bcast {
			bcast[i] = ip4[i] | ^ipNet.Mask[i]
		}
		add(netip.AddrPortFrom(netip.AddrFrom4([4]byte(bcast)), uint16(port))) //nolint:gosec // G115: port validated by the caller
	}
	return targets
}
