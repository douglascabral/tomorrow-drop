package discovery

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/trust"
)

const (
	defaultRateLimit  = 4
	defaultRateWindow = 10 * time.Second
	defaultMaxJitter  = 200 * time.Millisecond
	rateCapacity      = 1024
)

// ListenerOptions configure a discovery listener. Zero values use defaults.
type ListenerOptions struct {
	// Addr is the UDP address to bind, for example "0.0.0.0:47520".
	Addr string
	// Name is the friendly name announced to other devices.
	Name string
	// TCPPort is the port of the TLS listener announced to other devices.
	TCPPort int
	// RateLimit is the maximum answers per origin within RateWindow.
	RateLimit  int
	RateWindow time.Duration
	// MaxJitter bounds the random delay before answering.
	MaxJitter time.Duration
}

// Listener answers DISCOVER queries while it is running. Answers are unicast
// to the querier, rate-limited per origin and sent after a short random delay.
type Listener struct {
	conn      *net.UDPConn
	name      string
	tcpPort   int
	limiter   *RateLimiter
	maxJitter time.Duration
}

// Listen binds the UDP socket and answers queries until ctx ends.
func Listen(ctx context.Context, opts ListenerOptions) (*Listener, error) {
	addr, err := netip.ParseAddrPort(opts.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen address %q: %w: %w", opts.Addr, apperr.ErrMalformed, err)
	}
	if err := trust.ValidateName(opts.Name); err != nil {
		return nil, err
	}
	if opts.TCPPort < 1 || opts.TCPPort > 65535 {
		return nil, fmt.Errorf("TCP port %d out of range: %w", opts.TCPPort, apperr.ErrMalformed)
	}
	if opts.RateLimit == 0 {
		opts.RateLimit = defaultRateLimit
	}
	if opts.RateWindow == 0 {
		opts.RateWindow = defaultRateWindow
	}
	if opts.MaxJitter == 0 {
		opts.MaxJitter = defaultMaxJitter
	}

	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(addr))
	if err != nil {
		return nil, fmt.Errorf("bind discovery socket: %w", err)
	}
	ln := &Listener{
		conn:      conn,
		name:      opts.Name,
		tcpPort:   opts.TCPPort,
		limiter:   NewRateLimiter(opts.RateLimit, opts.RateWindow, rateCapacity),
		maxJitter: opts.MaxJitter,
	}
	go ln.serve(ctx)
	return ln, nil
}

// LocalAddr returns the bound UDP address.
func (l *Listener) LocalAddr() netip.AddrPort {
	return l.conn.LocalAddr().(*net.UDPAddr).AddrPort()
}

func (l *Listener) serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		_ = l.conn.Close()
	}()
	buf := make([]byte, limits.MaxDiscoveryDatagram+1)
	for {
		n, from, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		l.handle(ctx, buf[:n], from)
	}
}

func (l *Listener) handle(ctx context.Context, data []byte, from *net.UDPAddr) {
	query, err := Decode(data)
	if err != nil || query.Type != TypeDiscover {
		return
	}
	if !l.limiter.Allow(from.IP.String(), time.Now()) {
		return
	}
	answer, err := NewAnnounce(query, l.name, l.tcpPort)
	if err != nil {
		return
	}
	out, err := Encode(answer)
	if err != nil {
		return
	}
	delay := randomDelay(l.maxJitter)
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
			_, _ = l.conn.WriteToUDP(out, from)
		}
	}()
}

// randomDelay returns a uniform delay in [0, max) using crypto/rand.
func randomDelay(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	return time.Duration(binary.BigEndian.Uint64(b[:]) % uint64(max)) //nolint:gosec // G115: value reduced modulo max, which fits in int64
}
