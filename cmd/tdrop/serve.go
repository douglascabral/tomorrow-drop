package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
)

// serveOptions configure receive and daemon.
type serveOptions struct {
	listen      string
	out         string
	name        string
	once        bool
	autoTrusted bool
}

func (c *cmd) receiveCmd(args []string, daemon bool) error {
	name := "receive"
	if daemon {
		name = "daemon"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listen := fs.String("listen", "0.0.0.0:"+strconv.Itoa(defaultTCPPort), "")
	out := fs.String("out", "", "")
	deviceName := fs.String("name", "", "")
	auto := fs.Bool("auto-accept-trusted", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return usage("cli.uso")
	}
	if *auto && !daemon {
		return usage("cli.uso")
	}
	if *out == "" {
		dir, err := transfer.DefaultDownloadsDir()
		if err != nil {
			return err
		}
		*out = dir
	}
	opts := serveOptions{listen: *listen, out: *out, name: *deviceName, once: !daemon, autoTrusted: *auto}
	return c.serve(opts)
}

// defaultTCPPort is the TLS port announced by receive and daemon.
const defaultTCPPort = 47521

// serve accepts connections, runs pairing for unknown devices and offers for
// trusted ones. receive handles one connection; daemon keeps going until the
// context ends.
func (c *cmd) serve(opts serveOptions) error {
	id, err := c.identity()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", opts.listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", opts.listen, err)
	}
	defer func() { _ = ln.Close() }()
	stop := context.AfterFunc(c.ctx, func() { _ = ln.Close() })
	defer stop()

	bound, err := netAddrPort(ln.Addr())
	if err != nil {
		return err
	}
	if c.e.Ready != nil {
		c.e.Ready(bound)
	}
	c.say("%s", c.e.Tr.T("cli.listening", bound.String()))

	name := opts.name
	if name == "" {
		name = deviceName()
	}
	c.startDiscovery(name, int(bound.Port())) //nolint:gosec // G115: a uint16 port always fits in int

	receiver, err := transfer.NewReceiver(transfer.ReceiverConfig{
		Decide:     c.decide(opts.autoTrusted),
		StagingDir: filepath.Join(c.e.Home, "staging"),
		DestDir:    opts.out,
	})
	if err != nil {
		return err
	}
	store := c.trustStore()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if c.ctx.Err() != nil {
				return apperr.FromContext(c.ctx.Err())
			}
			return fmt.Errorf("accept: %w", err)
		}
		failed, err := c.handleConn(conn, id, store, receiver, opts.out)
		if opts.once {
			return err
		}
		if err != nil && !failed {
			c.warn("cli.error", c.e.Tr.T(apperr.MessageKey(err)))
		}
	}
}

// handleConn runs one accepted connection. It reports whether the connection
// ended in an error, so a daemon can log it and keep serving.
func (c *cmd) handleConn(conn net.Conn, id *identity.Identity, store transportTrust, receiver *transfer.Receiver, out string) (bool, error) {
	s, err := transport.Server(c.ctx, conn, id, store)
	if err != nil {
		return true, err
	}
	defer func() { _ = s.Close() }()
	if s.State() != protocol.StateTrusted {
		return false, c.pairWith(s, id, "")
	}
	res, err := receiver.Serve(c.ctx, s)
	if err != nil {
		return true, err
	}
	if res.Text != "" {
		c.say("%s", res.Text)
		return false, nil
	}
	c.say("%s", c.e.Tr.T("cli.received_files", len(res.Files), out))
	return false, nil
}

// decide is the recipient's decision for each offer. Offers from trusted
// devices are accepted without asking only when the daemon was started with
// --auto-accept-trusted. Every other offer is shown and must be confirmed.
func (c *cmd) decide(autoTrusted bool) transfer.Decision {
	return func(_ context.Context, offer transfer.Offer) bool {
		size := int64(0)
		for _, it := range offer.Items {
			size += it.Size
		}
		from := identity.Fingerprint{}
		if fp, err := identity.FingerprintOf(offer.Peer); err == nil {
			from = fp
		}
		if autoTrusted {
			c.say("%s", c.e.Tr.T("cli.offer_auto", len(offer.Items), size, from.Grouped()))
			return true
		}
		c.say("%s", c.e.Tr.T("cli.offer", kindLabel(c.e.Tr, offer.Kind), len(offer.Items), size, from.Grouped()))
		return c.e.ask(c.stdout, c.e.Tr.T("cli.offer_confirm"))
	}
}

func kindLabel(tr *i18n.Bundle, k transfer.Kind) string {
	switch k {
	case transfer.KindText:
		return tr.T("cli.kind.text")
	case transfer.KindLink:
		return tr.T("cli.kind.link")
	default:
		return tr.T("cli.kind.file")
	}
}

// startDiscovery answers discovery queries while the command runs. A failure
// to bind is reported but does not stop the transfer service.
func (c *cmd) startDiscovery(name string, tcpPort int) {
	if c.e.DiscoveryAddr == "" {
		return
	}
	ln, err := discovery.Listen(c.ctx, discovery.ListenerOptions{Addr: c.e.DiscoveryAddr, Name: name, TCPPort: tcpPort})
	if err != nil {
		c.warn("cli.discovery_unavailable")
		return
	}
	if c.e.ReadyDiscovery != nil {
		c.e.ReadyDiscovery(ln.LocalAddr())
	}
}

func deviceName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "tomorrow-drop"
	}
	return host
}

func netAddrPort(a net.Addr) (netAddr, error) {
	tcp, ok := a.(*net.TCPAddr)
	if !ok {
		return netAddr{}, errors.New("listener is not TCP")
	}
	return toNetAddr(tcp), nil
}
