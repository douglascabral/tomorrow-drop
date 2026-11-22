package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/pairing"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

const (
	tlsPort      = 47521
	discoveryUDP = 47520
	scanEvery    = 30 * time.Second
)

// searchNow asks for a discovery query at once: the Refresh button.
func (c *core) searchNow() {
	select {
	case c.scanNow <- struct{}{}:
	default:
	}
}

// pending is a question waiting for the person: a pairing step or an offer.
type pendingPair struct {
	local, remote identity.Fingerprint
	incoming      bool
	suggested     string
	stage         string // "confirm" or "name"
	confirm       chan bool
	name          chan string
}

type pendingOffer struct {
	offer  transfer.Offer
	from   string
	answer chan bool
}

type job struct {
	id       int
	title    string
	fraction float64
	state    string
	message  string
}

// core holds the application's state. Access is guarded by mu.
type core struct {
	mu      sync.Mutex
	tr      *i18n.Bundle
	home    string
	docsDir string
	name    string
	id      *identity.Identity
	store   *trust.Store
	reg     *discovery.Registry
	recv    *transfer.Receiver
	addrs   map[string]string // fingerprint hex -> last address that answered
	// presence is the last reachability check of each paired device; presenceBusy
	// keeps one check at a time (startPresenceCheck).
	presence     *discovery.Presence
	presenceBusy atomic.Bool
	pair         *pendingPair
	offer        *pendingOffer
	jobs         []*job // operations in progress only: nothing is kept after they end
	nextJob      int
	result       *resultJSON // the outcome of the last operation, until the person closes it
	received     *string     // a text received, shown until the person closes it
	notice       string
	listening    bool
	scanNow      chan struct{}
	stop         context.CancelFunc
	ctx          context.Context
}

// newCore opens the application's files in home and creates the identity on
// first use. docsDir is where received files are saved (visible in the Files app).
func newCore(home, docsDir, name, lang string) (*core, error) {
	tr, err := i18n.Load(i18n.Resolve(lang, i18n.DetectSystem()))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("create application folder: %w", err)
	}
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		return nil, fmt.Errorf("create documents folder: %w", err)
	}
	id, err := identity.Store{Dir: home}.LoadOrCreate()
	if err != nil {
		return nil, err
	}
	c := &core{
		tr: tr, home: home, docsDir: docsDir, name: name, id: id,
		store:    trust.Open(filepath.Join(home, "trust.json")),
		reg:      discovery.NewRegistry(90*time.Second, limits.MaxDiscoveredDevices),
		addrs:    loadAddrs(home),
		scanNow:  make(chan struct{}, 1),
		presence: discovery.NewPresence(presenceTTL),
	}
	recv, err := transfer.NewReceiver(transfer.ReceiverConfig{
		Decide:     c.askOffer,
		StagingDir: filepath.Join(home, "staging"),
		DestDir:    docsDir,
	})
	if err != nil {
		return nil, err
	}
	c.recv = recv
	c.ctx, c.stop = context.WithCancel(context.Background())
	return c, nil
}

// availability starts or stops listening and discovery. The app calls it when it
// comes to the foreground or goes to the background (PRD §17).
func (c *core) setAvailable(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !on {
		c.interruptLocked()
	}
	if on == c.listening {
		return
	}
	if !on {
		c.stop()
		c.ctx, c.stop = context.WithCancel(context.Background())
		c.listening = false
		c.notice = ""
		return
	}
	c.listening = true
	c.startListeners(c.ctx)
}

// interruptLocked ends the running operations when the app leaves the screen:
// they are cancelled, and the person is told when they come back. Caller holds mu.
func (c *core) interruptLocked() {
	if len(c.jobs) == 0 {
		return
	}
	c.jobs = nil
	c.result = &resultJSON{OK: false, Title: c.tr.T("ios.background.title"), Message: c.tr.T("ios.background.message")}
}

// startListeners runs the discovery answers, the periodic scan and the TLS
// listener until ctx ends. Caller holds mu.
func (c *core) startListeners(ctx context.Context) {
	local := discovery.LocalAddresses()
	if _, err := discovery.Listen(ctx, discovery.ListenerOptions{
		Addr: "0.0.0.0:" + strconv.Itoa(discoveryUDP), Name: c.name, TCPPort: tlsPort,
	}); err != nil {
		c.notice = c.tr.T(apperr.MessageKey(err))
	}
	// Paired devices are checked on their own timer, also when no scanner runs.
	go func() {
		t := time.NewTicker(scanEvery)
		defer t.Stop()
		c.startPresenceCheck()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.startPresenceCheck()
			}
		}
	}()
	targets := discovery.BroadcastTargets(discoveryUDP, addrsOf(local))
	if sc, err := discovery.NewScanner(targets); err == nil {
		found := make(chan discovery.Announcement, 32)
		go func() { _ = sc.Run(ctx, found) }()
		go func() {
			defer func() { _ = sc.Close() }()
			t := time.NewTicker(scanEvery)
			defer t.Stop()
			_ = sc.Query(time.Now())
			for {
				select {
				case <-ctx.Done():
					return
				case a := <-found:
					if !discovery.IsOwnAnnouncement(a, local, tlsPort) {
						c.mu.Lock()
						_ = c.reg.Observe(a, time.Now())
						c.mu.Unlock()
					}
				case <-c.scanNow:
					_ = sc.Query(time.Now())
					c.startPresenceCheck()
				case <-t.C:
					_ = sc.Query(time.Now())
				}
			}
		}()
	}
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(tlsPort))
	if err != nil {
		c.notice = c.tr.T(apperr.MessageKey(err))
		return
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go c.handleIncoming(ctx, conn)
		}
	}()
}

// handleIncoming serves one connection: pairing with a device that is not paired
// here, and receiving from a paired one.
func (c *core) handleIncoming(ctx context.Context, conn net.Conn) {
	s, err := transport.Server(ctx, conn, c.id, c.store)
	if err != nil {
		return
	}
	defer func() { _ = s.Close() }()
	// The other device listens on the TLS port; its source port is not that port.
	from := netip.AddrPortFrom(remoteIP(conn), tlsPort)
	if s.State() != protocol.StateTrusted {
		_ = c.runPairing(ctx, s, from, true, "")
		return
	}
	// The receive is listed while it runs, so the person keeps the app open.
	j := c.addJob("running", c.tr.T("ios.receive.running"), "")
	res, err := c.recv.Serve(ctx, s)
	if err != nil {
		if ctx.Err() == nil && !errors.Is(err, apperr.ErrNoOffer) {
			c.finish(j, "failed", c.tr.T(apperr.MessageKey(err)))
		} else {
			c.dropJob(j)
		}
		return
	}
	if res.Text != "" {
		c.mu.Lock()
		text := res.Text
		c.received = &text
		c.removeLocked(j)
		c.mu.Unlock()
		return
	}
	c.finish(j, "done", c.tr.T("ios.receive.saved", len(res.Files)))
}

// runPairing is the pairing flow shared by both directions. The person answers
// through pendingPair; the fingerprints are compared on both devices.
func (c *core) runPairing(ctx context.Context, s *transport.Session, at netip.AddrPort, incoming bool, suggested string) error {
	c.mu.Lock()
	if c.pair != nil {
		c.mu.Unlock()
		return fmt.Errorf("another pairing is waiting: %w", apperr.ErrBusy)
	}
	p := &pendingPair{incoming: incoming, suggested: suggested, confirm: make(chan bool, 1), name: make(chan string, 1)}
	c.pair = p
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.pair == p {
			c.pair = nil
		}
		c.mu.Unlock()
	}()
	name, err := pairing.Run(ctx, s, c.id, c.store, pairing.Prompts{
		Confirm: func(local, remote identity.Fingerprint) bool {
			c.mu.Lock()
			p.local, p.remote, p.stage = local, remote, "confirm"
			c.mu.Unlock()
			select {
			case ok := <-p.confirm:
				return ok
			case <-ctx.Done():
				return false
			}
		},
		Name: func(remote identity.Fingerprint) string {
			c.mu.Lock()
			p.stage = "name"
			c.mu.Unlock()
			select {
			case n := <-p.name:
				return n
			case <-ctx.Done():
				return ""
			}
		},
	})
	if err != nil {
		if !errors.Is(err, pairing.ErrCancelled) {
			c.addJob("failed", c.tr.T("ios.pair.title"), c.tr.T(apperr.MessageKey(err)))
		}
		return err
	}
	c.mu.Lock()
	c.rememberLocked(s.PeerKey(), at)
	c.mu.Unlock()
	c.addJob("done", c.tr.T("ios.pair.title"), c.tr.T("ios.pair.paired", name))
	return nil
}

// rememberLocked records the address of a paired device. Caller holds mu.
func (c *core) rememberLocked(key []byte, at netip.AddrPort) {
	fp, err := identity.FingerprintOf(key)
	if err != nil {
		return
	}
	c.addrs[fp.Hex()] = at.String()
	saveAddrs(c.home, c.addrs)
}

// askOffer is the decision for each offer (PRD §13): it waits for the person.
func (c *core) askOffer(ctx context.Context, offer transfer.Offer) bool {
	peers, _ := c.store.List()
	from := peerLabel(peers, offer.Peer)
	p := &pendingOffer{offer: offer, from: from, answer: make(chan bool, 1)}
	c.mu.Lock()
	c.offer = p
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.offer == p {
			c.offer = nil
		}
		c.mu.Unlock()
	}()
	select {
	case ok := <-p.answer:
		return ok
	case <-ctx.Done():
		return false
	}
}

func peerLabel(peers []trust.Peer, key []byte) string {
	for _, p := range peers {
		if bytes.Equal(p.Key, key) {
			return p.Name
		}
	}
	if fp, err := identity.FingerprintOf(key); err == nil {
		return fp.Hex()[:8]
	}
	return "?"
}

// addJob starts an operation. A running one is listed while it runs; any other
// state is the outcome, shown once and kept by nobody (the app keeps no history).
func (c *core) addJob(state, title, message string) *job {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state == "running" {
		c.nextJob++
		j := &job{id: c.nextJob, title: title, state: state}
		c.jobs = append(c.jobs, j)
		return j
	}
	c.result = &resultJSON{OK: state == "done", Title: title, Message: message}
	return nil
}

// finish ends a running operation: it leaves the list and its outcome is shown once.
func (c *core) finish(j *job, state, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// An operation already ended by leaving the screen keeps that message.
	if !c.removeLocked(j) {
		return
	}
	c.result = &resultJSON{OK: state == "done", Title: j.title, Message: message}
}

// dropJob ends a running operation without an outcome, for one that did not
// really happen (for example, a connection that never received an offer).
func (c *core) dropJob(j *job) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeLocked(j)
}

// removeLocked takes j out of the running list and reports whether it was
// there. The caller holds c.mu.
func (c *core) removeLocked(j *job) bool {
	for i, x := range c.jobs {
		if x == j {
			c.jobs = append(c.jobs[:i], c.jobs[i+1:]...)
			return true
		}
	}
	return false
}

// send sends a spec to a paired device and reports progress on j.
func (c *core) send(addr string, spec transfer.SendSpec, j *job) {
	ep, err := netip.ParseAddrPort(addr)
	finish := func(state, message string) { c.finish(j, state, message) }
	if err != nil {
		finish("failed", c.tr.T(apperr.MessageKey(apperr.ErrMalformed)))
		return
	}
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Minute)
	defer cancel()
	s, err := transport.Dial(ctx, ep, c.id, c.store)
	if err != nil {
		finish("failed", c.tr.T(apperr.MessageKey(err)))
		return
	}
	defer func() { _ = s.Close() }()
	if s.State() != protocol.StateTrusted {
		finish("failed", c.tr.T("ios.send.not_paired"))
		return
	}
	_, err = transfer.Send(ctx, s, spec, func(p transfer.Progress) {
		c.mu.Lock()
		if p.TotalSize > 0 {
			j.fraction = float64(p.TotalSent) / float64(p.TotalSize)
		}
		c.mu.Unlock()
	})
	if err != nil {
		finish("failed", c.tr.T(apperr.MessageKey(err)))
		return
	}
	c.mu.Lock()
	c.rememberLocked(s.PeerKey(), ep)
	c.mu.Unlock()
	finish("done", c.tr.T("ios.send.done"))
}

// state is the snapshot the interface shows.
func (c *core) state() stateJSON {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	// Lists are never null: the interface decodes them as arrays.
	st := stateJSON{
		Fingerprint: c.id.Fingerprint().Grouped(),
		Address:     formatAddrs(discovery.LocalAddresses()),
		Listening:   c.listening,
		Notice:      c.notice,
		Devices:     []deviceJSON{},
		Paired:      []pairedJSON{},
		Jobs:        []jobJSON{},
	}
	peers, _ := c.store.List()
	// The same device paired again is listed once, under its latest pairing.
	peers = ui.CurrentPeers(peers, c.addrs)
	for _, p := range peers {
		addr := c.addrs[p.Fingerprint.Hex()]
		st.Paired = append(st.Paired, pairedJSON{
			Name: p.Name, Fingerprint: p.Fingerprint.Grouped(), Hex: p.Fingerprint.Hex(), Addr: addr,
			Presence: c.presenceText(addr, now),
		})
	}
	// Online devices first, then the others in their stored order.
	sort.SliceStable(st.Paired, func(i, j int) bool {
		return st.Paired[i].Presence == "online" && st.Paired[j].Presence != "online"
	})
	pairedAt := map[string]trust.Peer{}
	for _, p := range peers {
		if a, ok := c.addrs[p.Fingerprint.Hex()]; ok {
			pairedAt[a] = p
		}
	}
	for _, d := range c.reg.List(now) {
		dj := deviceJSON{Name: d.Name, Addr: d.Addr.String()}
		if p, ok := pairedAt[d.Addr.String()]; ok {
			dj.Name, dj.Paired, dj.Fingerprint = p.Name, true, p.Fingerprint.Grouped()
		}
		st.Devices = append(st.Devices, dj)
	}
	if p := c.pair; p != nil {
		st.Pair = &pairJSON{
			Local: p.local.Grouped(), Remote: p.remote.Grouped(), Stage: p.stage,
			Suggested: p.suggested, Incoming: p.incoming,
		}
	}
	if o := c.offer; o != nil {
		oj := offerJSON{From: o.from, Count: len(o.offer.Items), Text: o.offer.Kind != transfer.KindFile}
		for _, it := range o.offer.Items {
			oj.Names = append(oj.Names, it.Name)
			oj.Bytes += it.Size
		}
		st.Offer = &oj
	}
	for _, j := range c.jobs {
		st.Jobs = append(st.Jobs, jobJSON{ID: j.id, Title: j.title, Fraction: j.fraction, State: j.state, Message: j.message})
	}
	if c.result != nil {
		r := *c.result
		st.Result = &r
	}
	st.Received = c.received
	return st
}

func (c *core) toJSON() string {
	data, err := json.Marshal(c.state())
	if err != nil {
		return "{}"
	}
	return string(data)
}

// remoteIP is the address of the other device, without its source port.
func remoteIP(conn net.Conn) netip.Addr {
	if ua, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		if a, ok := netip.AddrFromSlice(ua.IP); ok {
			return a.Unmap()
		}
	}
	return netip.Addr{}
}

func addrsOf(list []netip.Addr) []net.Addr {
	out := make([]net.Addr, 0, len(list))
	for _, a := range list {
		if a.Is4() {
			out = append(out, &net.IPNet{IP: net.IP(a.AsSlice()), Mask: net.CIDRMask(24, 32)})
		}
	}
	return out
}

func formatAddrs(list []netip.Addr) string {
	s := ""
	for _, a := range list {
		if !a.Is4() || a.IsLoopback() || a.IsLinkLocalUnicast() {
			continue
		}
		if s != "" {
			s += ", "
		}
		s += a.String()
	}
	if s == "" {
		return "—"
	}
	return s
}

// nameAt is the name a device announces at addr, if discovery has seen it.
func (c *core) nameAt(ep netip.AddrPort) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.reg.List(time.Now()) {
		if d.Addr == ep {
			return d.Name
		}
	}
	return ""
}

// ctxOrBackground is the context of the app, or a background one before tdStart.
func (c *core) ctxOrBackground() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// presenceText is the presence of the device last seen at addr, as the interface
// names it: "online", "offline" or "unknown" (never checked, or too old).
func (c *core) presenceText(addr string, now time.Time) string {
	ep, err := netip.ParseAddrPort(addr)
	if err != nil {
		return "unknown"
	}
	switch c.presence.State(ep, now) {
	case discovery.PresenceOnline:
		return "online"
	case discovery.PresenceOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// startPresenceCheck checks the paired devices in the background. It does
// nothing while a transfer or a pairing is on screen, or while a check runs, and
// reports whether it started one. It never blocks the caller.
func (c *core) startPresenceCheck() bool {
	c.mu.Lock()
	busy := len(c.jobs) > 0 || c.pair != nil
	c.mu.Unlock()
	if busy || !c.presenceBusy.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer c.presenceBusy.Store(false)
		c.checkPresence()
	}()
	return true
}

// checkPresence probes the last known address of every paired device.
func (c *core) checkPresence() {
	c.mu.Lock()
	peers, _ := c.store.List()
	var eps []netip.AddrPort
	for _, p := range peers {
		if text, ok := c.addrs[p.Fingerprint.Hex()]; ok {
			if ep, err := netip.ParseAddrPort(text); err == nil {
				eps = append(eps, ep)
			}
		}
	}
	c.mu.Unlock()
	if len(eps) == 0 {
		return
	}
	c.presence.Check(c.ctxOrBackground(), eps, protocol.PresenceTimeout)
}

// presenceTTL is how long a reachability check counts as current: two discovery
// rounds and the probe timeout.
const presenceTTL = 2*protocol.DiscoveryInterval + protocol.PresenceTimeout

// unpair removes a device from the trust list, and forgets the address it was
// seen at and its reachability.
func (c *core) unpair(peer trust.Peer) error {
	if err := c.store.Remove(peer.Fingerprint); err != nil {
		return err
	}
	c.mu.Lock()
	addr, known := c.addrs[peer.Fingerprint.Hex()]
	delete(c.addrs, peer.Fingerprint.Hex())
	saveAddrs(c.home, c.addrs)
	c.mu.Unlock()
	if known {
		if ep, err := netip.ParseAddrPort(addr); err == nil {
			c.presence.Forget(ep)
		}
	}
	return nil
}

// pairedByHex returns the paired device with this fingerprint (hex).
func (c *core) pairedByHex(hex string) (trust.Peer, bool) {
	peers, err := c.store.List()
	if err != nil {
		return trust.Peer{}, false
	}
	for _, p := range peers {
		if p.Fingerprint.Hex() == hex {
			return p, true
		}
	}
	return trust.Peer{}, false
}
