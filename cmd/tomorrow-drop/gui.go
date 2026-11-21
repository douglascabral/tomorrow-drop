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
	// listening counts the incoming listener until it is closed.
	listening sync.WaitGroup
	avail     context.CancelFunc
	scanMu    sync.Mutex
	scanner   *discovery.Scanner

	devicesList *widget.List
	emptyHint   *fyne.Container
	rows        []ui.DeviceRow
	// homeList is what the home list shows: the paired devices, then the found ones
	// (home_list.go). homeParts holds the widgets of its lines.
	homeList  []homeItem
	homeParts map[fyne.CanvasObject]*homeItemParts
	// localAddrs are this device's addresses, used to drop its own answers.
	localAddrs []netip.Addr
	// addrs reads this device's addresses; tests replace it.
	addrs func() []netip.Addr
	// pickingFile is true while the system's storage picker is open to save a
	// received file: leaving the foreground for it must not cancel the reception.
	pickingFile atomic.Bool
	// share is what another app shared, waiting for a device (share.go).
	share     *pendingShare
	pendingMu sync.Mutex
	// bg counts the goroutines that change the screen after a tap, so a test can
	// wait for them before it ends.
	bg sync.WaitGroup
	// receiving is true while a reception is on screen (see endReception).
	receiving atomic.Bool
	// pairHint is the name the device being paired announces, offered as the
	// start of the name the person gives it.
	pairHint string
	// paired remembers which paired device answered at each address (paired.go).
	pairedMu sync.Mutex
	paired   map[netip.AddrPort]trust.Peer
	// screenOn keeps the screen on (true) or lets it turn off (false). Only
	// phones use it; main sets it from the platform driver.
	screenOn func(bool)
}

// background runs fn in a goroutine that is counted in g.bg.
func (g *gui) background(fn func()) {
	g.bg.Add(1)
	go func() {
		defer g.bg.Done()
		fn()
	}()
}

// keepScreenOn keeps the phone's screen on while the app is in the foreground.
// Leaving the foreground lets the screen turn off, which interrupts transfers.
func (g *gui) keepScreenOn(on bool) {
	if g.mobile && g.screenOn != nil {
		g.screenOn(on)
	}
}

// newGUI builds the application state. save picks where received files go; nil
// means the Downloads folder, which is used on desktop.
func newGUI(ctx context.Context, tr *i18n.Bundle, home, cfgPath string, save transfer.Destination) (*gui, error) {
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
		id:        id,
		store:     trust.Open(filepath.Join(home, "trust.json")),
		reg:       discovery.NewRegistry(90*time.Second, limits.MaxDiscoveredDevices),
		tlsPort:   defaultTLSPort,
		discPort:  discovery.DefaultPort,
		ops:       map[int]context.CancelFunc{},
		rowParts:  map[fyne.CanvasObject]*deviceRowParts{},
		homeParts: map[fyne.CanvasObject]*homeItemParts{},
		paired:    map[netip.AddrPort]trust.Peer{},
		addrs:     discovery.LocalAddresses,
	}
	recv, err := transfer.NewReceiver(transfer.ReceiverConfig{
		Decide:      g.askOffer,
		StagingDir:  filepath.Join(home, "staging"),
		DestDir:     downloads,
		Destination: save,
		OnProgress:  g.onReceiveProgress,
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
		// Wait until the listener is really closed: once this returns, no new
		// connection can be accepted (PRD §17.2).
		g.listening.Wait()
	}
	g.availMu.Unlock()
	// Only mobile interrupts transfers on the way to the background (task 057).
	// Desktop keeps the transfers already authorized (PRD §17.2).
	if !on && g.mobile && !g.pickingFile.Load() {
		g.cancelOperations()
	}
	if err := g.ctl.SetVisible(on); err != nil {
		g.logError(err)
	}
	g.keepScreenOn(on)
}

func (g *gui) startListeners(ctx context.Context) {
	addrs, _ := net.InterfaceAddrs()
	sc, err := discovery.NewScanner(discovery.BroadcastTargets(discovery.DefaultPort, addrs))
	if err == nil {
		g.scanMu.Lock()
		g.scanner = sc
		g.scanMu.Unlock()
		// A local channel: the goroutines must not read g.found, which the next
		// start replaces while they may still be running.
		found := make(chan discovery.Announcement, 32)
		g.found = found
		go func() { _ = sc.Run(ctx, found) }()
		go g.collect(ctx, found)
	}

	name := announcedName()
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
	g.listening.Add(1)
	go func() {
		defer g.listening.Done()
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
			g.observe(a)
		}
	}
}

// observe adds an answer to the device list. This device's own answers are
// dropped, so the list shows only other devices.
func (g *gui) observe(a discovery.Announcement) {
	if discovery.IsOwnAnnouncement(a, g.localAddrs, g.tlsPort) {
		return
	}
	_ = g.reg.Observe(a, time.Now())
	g.refreshDevices()
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

// endReception takes the progress ring off after a reception. A reception that
// was cancelled, by the person or by the phone leaving the foreground, says so
// and offers to wait for files again; otherwise the screen would keep showing a
// transfer that is no longer running.
func (g *gui) endReception(err error) {
	wasReceiving := g.receiving.Swap(false)
	g.onUI(func() { g.recvMeter = nil })
	if wasReceiving && err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, apperr.ErrCanceled)) {
		g.showOutcome(false, g.tr.T("gui.progress.canceled"), "", nil, g.receiveAgainAction())
	}
}

// receiveAgainAction is the completion action that opens the receive screen.
func (g *gui) receiveAgainAction() doneAction {
	return doneAction{label: g.tr.T("gui.done.again"), run: g.showReceive}
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
		g.pairAsked(ctx, s, true)
		return
	}
	res, err := g.recv.Serve(ctx, s)
	g.endReception(err)
	if err != nil {
		// A trusted device that connects without an offer (for example, when
		// the person only opens it) is not a failed transfer.
		if errors.Is(err, apperr.ErrPeerWantsToPair) {
			if peer, ok := g.pairedByKey(s.PeerKey()); ok {
				g.askUnpairOnRequest(peer)
			}
			return
		}
		if ctx.Err() != nil || errors.Is(err, apperr.ErrNoOffer) {
			return
		}
		g.showOutcome(false, g.tr.T("gui.receive.failed"), g.tr.T(apperr.MessageKey(err)), nil, g.receiveAgainAction())
		return
	}
	if res.Text != "" {
		g.showText(res.Text)
		return
	}
	g.showCompleted(g.tr.T("gui.receive.done"), g.savedText(len(res.Files)), res.Files, g.receiveAgainAction())
}

// pairAsked runs the pairing on s and reports whether the device was paired.
// announce says "Paired" in an alert; a pairing the person started goes on to
// the device screen instead.
func (g *gui) pairAsked(ctx context.Context, s *transport.Session, announce bool) bool {
	name, err := pairing.Run(ctx, s, g.id, g.store, pairing.Prompts{
		Confirm: g.askPairing,
		Name:    g.askName,
	})
	if err != nil {
		if !errors.Is(err, pairing.ErrCancelled) {
			g.notify(g.tr.T("gui.pair.failed"), g.tr.T(apperr.MessageKey(err)))
		}
		return false
	}
	if announce {
		g.notify(g.tr.T("gui.pair.done"), g.tr.T("gui.pair.paired", name))
	}
	g.refreshDevices()
	return true
}

// manualEndpoint parses what the person typed in the manual connection field.
func manualEndpoint(text string) (netip.AddrPort, error) {
	return discovery.ParseEndpoint(text)
}
