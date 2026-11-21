package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

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

// gui connects the window to the shared core. Protocol, trust and transfer
// rules are never implemented here; this type only starts the services, asks
// people questions and shows results.
type gui struct {
	rowParts  map[fyne.CanvasObject]*deviceRowParts
	recvMeter *meter
	recvStart time.Time
	ctx       context.Context
	tr        *i18n.Bundle
	app       fyne.App
	win       fyne.Window
	home      string
	cfgPath   string

	id    *identity.Identity
	store *trust.Store
	recv  *transfer.Receiver
	reg   *discovery.Registry
	ctl   *discovery.Controller

	tlsPort  int
	discPort int
	boundMu  sync.Mutex
	bound    int
	mobile   bool
	opMu     sync.Mutex
	ops      map[int]context.CancelFunc
	nextOp   int
	queries  atomic.Int64
	found    chan discovery.Announcement

	availMu sync.Mutex
	avail   context.CancelFunc
	scanMu  sync.Mutex
	scanner *discovery.Scanner

	devicesList *widget.List
	rows        []ui.DeviceRow
}

func newGUI(ctx context.Context, tr *i18n.Bundle, home, cfgPath string) (*gui, error) {
	id, err := identity.Store{Dir: home}.LoadOrCreate()
	if err != nil {
		return nil, err
	}
	downloads, err := transfer.DefaultDownloadsDir()
	if err != nil {
		return nil, err
	}
	g := &gui{
		ctx: ctx, tr: tr, home: home, cfgPath: cfgPath,
		id:       id,
		store:    trust.Open(filepath.Join(home, "trust.json")),
		reg:      discovery.NewRegistry(90*time.Second, limits.MaxDiscoveredDevices),
		tlsPort:  defaultTLSPort,
		discPort: discovery.DefaultPort,
		ops:      map[int]context.CancelFunc{},
		rowParts: map[fyne.CanvasObject]*deviceRowParts{},
	}
	recv, err := transfer.NewReceiver(transfer.ReceiverConfig{
		Decide:     g.askOffer,
		StagingDir: filepath.Join(home, "staging"),
		DestDir:    downloads,
		OnProgress: g.onReceiveProgress,
	})
	if err != nil {
		return nil, err
	}
	g.recv = recv
	g.ctl = discovery.NewController(g.queryAll, g.logError, discovery.DefaultInterval, discovery.DefaultCooldown)
	go g.ctl.Run(ctx)
	return g, nil
}

// defaultTLSPort is the port the GUI listens on for transfers. It matches the CLI.
const defaultTLSPort = 47521

// setAvailable starts or stops discovery and the incoming listener. The window
// calls it from its lifecycle events: foreground means available, background
// means not available (PRD §17). Transfers already running are not affected.
func (g *gui) setAvailable(on bool) {
	g.availMu.Lock()
	if on && g.avail == nil {
		ctx, cancel := context.WithCancel(g.ctx)
		g.avail = cancel
		g.startListeners(ctx)
	}
	if !on && g.avail != nil {
		g.avail()
		g.avail = nil
	}
	g.availMu.Unlock()
	// Only mobile interrupts transfers on the way to the background (task 057).
	// Desktop keeps the transfers already authorized (PRD §17.2).
	if !on && g.mobile {
		g.cancelOperations()
	}
	if err := g.ctl.SetVisible(on); err != nil {
		g.logError(err)
	}
}

func (g *gui) startListeners(ctx context.Context) {
	addrs, _ := net.InterfaceAddrs()
	sc, err := discovery.NewScanner(discovery.BroadcastTargets(discovery.DefaultPort, addrs))
	if err == nil {
		g.scanMu.Lock()
		g.scanner = sc
		g.scanMu.Unlock()
		g.found = make(chan discovery.Announcement, 32)
		go func() { _ = sc.Run(ctx, g.found) }()
		go g.collect(ctx, g.found)
	}

	name := deviceName()
	if _, err := discovery.Listen(ctx, discovery.ListenerOptions{
		Addr: "0.0.0.0:" + strconv.Itoa(g.discPort), Name: name, TCPPort: g.tlsPort,
	}); err != nil {
		g.logError(err)
	}

	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(g.tlsPort))
	if err != nil {
		g.logError(err)
		return
	}
	g.setBoundTLSPort(ln.Addr().(*net.TCPAddr).Port)
	go func() {
		<-ctx.Done()
		g.setBoundTLSPort(0)
		_ = ln.Close()
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go g.handleIncoming(conn)
		}
	}()
}

// beginOperation starts a transfer or pairing that the application tracks. The
// returned function ends it and must be called exactly once.
func (g *gui) beginOperation() (context.Context, func()) {
	ctx, cancel := context.WithCancel(g.ctx)
	g.opMu.Lock()
	g.nextOp++
	id := g.nextOp
	g.ops[id] = cancel
	g.opMu.Unlock()
	return ctx, func() {
		g.opMu.Lock()
		delete(g.ops, id)
		g.opMu.Unlock()
		cancel()
	}
}

// cancelOperations interrupts every tracked operation.
func (g *gui) cancelOperations() {
	g.opMu.Lock()
	defer g.opMu.Unlock()
	for id, cancel := range g.ops {
		cancel()
		delete(g.ops, id)
	}
}

// boundTLSPort returns the port of the incoming listener, or 0 when it is closed.
func (g *gui) boundTLSPort() int {
	g.boundMu.Lock()
	defer g.boundMu.Unlock()
	return g.bound
}

func (g *gui) setBoundTLSPort(port int) {
	g.boundMu.Lock()
	defer g.boundMu.Unlock()
	g.bound = port
}

// collect moves discovery answers into the device list and refreshes it.
func (g *gui) collect(ctx context.Context, found <-chan discovery.Announcement) {
	for {
		select {
		case <-ctx.Done():
			return
		case a := <-found:
			_ = g.reg.Observe(a, time.Now())
			g.refreshDevices()
		}
	}
}

func (g *gui) queryAll(now time.Time) error {
	g.queries.Add(1)
	g.scanMu.Lock()
	sc := g.scanner
	g.scanMu.Unlock()
	if sc == nil {
		return nil
	}
	return sc.Query(now)
}

func (g *gui) logError(err error) {
	fmt.Fprintln(os.Stderr, g.tr.T(apperr.MessageKey(err)))
}

// handleIncoming runs one accepted connection: pairing for unknown devices,
// receiving for trusted ones.
func (g *gui) handleIncoming(conn net.Conn) {
	ctx, done := g.beginOperation()
	defer done()
	s, err := transport.Server(ctx, conn, g.id, g.store)
	if err != nil {
		return
	}
	defer func() { _ = s.Close() }()
	if s.State() != protocol.StateTrusted {
		g.pairAsked(ctx, s)
		return
	}
	res, err := g.recv.Serve(ctx, s)
	g.onUI(func() { g.recvMeter = nil })
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		g.showCompleted(g.tr.T("gui.receive.failed"), g.tr.T(apperr.MessageKey(err)), nil)
		return
	}
	if res.Text != "" {
		g.showText(res.Text)
		return
	}
	g.showCompleted(g.tr.T("gui.receive.done"), g.tr.T("gui.receive.saved", len(res.Files)), res.Files)
}

func (g *gui) pairAsked(ctx context.Context, s *transport.Session) {
	name, err := pairing.Run(ctx, s, g.id, g.store, pairing.Prompts{
		Confirm: g.askPairing,
		Name:    g.askName,
	})
	if err != nil {
		if !errors.Is(err, pairing.ErrCancelled) {
			g.notify(g.tr.T("gui.pair.failed"), g.tr.T(apperr.MessageKey(err)))
		}
		return
	}
	g.notify(g.tr.T("gui.pair.done"), g.tr.T("gui.pair.paired", name))
	g.refreshDevices()
}

func deviceName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "tomorrow-drop"
	}
	return host
}

// manualEndpoint parses what the person typed in the manual connection field.
func manualEndpoint(text string) (netip.AddrPort, error) {
	return discovery.ParseEndpoint(text)
}
