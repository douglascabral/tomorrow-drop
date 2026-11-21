package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/pairing"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
)

// cmd carries one command invocation: its context, output streams and env.
type cmd struct {
	ctx    context.Context
	stdout io.Writer
	stderr io.Writer
	e      *env
}

func (c *cmd) say(format string, args ...any) {
	_, _ = fmt.Fprintf(c.stdout, format+"\n", args...)
}

func (c *cmd) warn(key string, args ...any) {
	_, _ = fmt.Fprintln(c.stderr, c.e.Tr.T(key, args...))
}

func (c *cmd) identity() (*identity.Identity, error) {
	return identity.Store{Dir: c.e.Home}.LoadOrCreate()
}

func (c *cmd) trustStore() *trust.Store {
	return trust.Open(filepath.Join(c.e.Home, "trust.json"))
}

func (c *cmd) dispatch(args []string) error {
	if len(args) == 0 {
		return usage("cli.uso")
	}
	rest := args[1:]
	switch args[0] {
	case "identity":
		return c.identityCmd(rest)
	case "trust":
		return c.trustCmd(rest)
	case "discover":
		return c.discoverCmd(rest)
	case "pair":
		return c.pairCmd(rest)
	case "send":
		return c.sendCmd(rest)
	case "receive":
		return c.receiveCmd(rest, false)
	case "daemon":
		return c.receiveCmd(rest, true)
	}
	return usage("cli.uso")
}

func (c *cmd) identityCmd(args []string) error {
	if len(args) != 1 || args[0] != "show" {
		return usage("cli.uso")
	}
	id, err := c.identity()
	if err != nil {
		return err
	}
	c.say("%s", c.e.Tr.T("cli.identity", id.Fingerprint().Grouped()))
	return nil
}

func (c *cmd) trustCmd(args []string) error {
	if len(args) == 1 && args[0] == "list" {
		return c.trustList()
	}
	if len(args) == 2 && args[0] == "remove" {
		return c.trustRemove(args[1])
	}
	return usage("cli.uso")
}

func (c *cmd) trustList() error {
	peers, err := c.trustStore().List()
	if err != nil {
		return err
	}
	if len(peers) == 0 {
		c.say("%s", c.e.Tr.T("cli.trust_empty"))
		return nil
	}
	for _, p := range peers {
		c.say("%s", c.e.Tr.T("cli.trust_item", shortID(p.Fingerprint), p.Name, p.Fingerprint.Grouped()))
	}
	return nil
}

func (c *cmd) trustRemove(id string) error {
	store := c.trustStore()
	peers, err := store.List()
	if err != nil {
		return err
	}
	var match []trust.Peer
	for _, p := range peers {
		if strings.HasPrefix(p.Fingerprint.Hex(), strings.ToLower(id)) {
			match = append(match, p)
		}
	}
	switch len(match) {
	case 0:
		return cliError{code: exitFailure, key: "cli.trust_not_found", args: []any{id}}
	case 1:
		if err := store.Remove(match[0].Fingerprint); err != nil {
			return err
		}
		c.say("%s", c.e.Tr.T("cli.trust_removed", match[0].Name))
		return nil
	default:
		return cliError{code: exitUsage, key: "cli.trust_ambiguous", args: []any{id}}
	}
}

// shortID is the identifier shown by trust list and accepted by trust remove.
func shortID(fp identity.Fingerprint) string {
	return fp.Hex()[:16]
}

func (c *cmd) pairCmd(args []string) error {
	fs := flag.NewFlagSet("pair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		return usage("cli.uso")
	}
	ep, err := c.resolve(fs.Arg(0))
	if err != nil {
		return err
	}
	id, err := c.identity()
	if err != nil {
		return err
	}
	s, err := transport.Dial(c.ctx, ep, id, c.trustStore())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	if s.State() == protocol.StateTrusted {
		c.say("%s", c.e.Tr.T("cli.already_trusted"))
		return nil
	}
	return c.pairWith(s, id, *name)
}

// pairWith runs the shared pairing flow with the terminal as the person's answer.
func (c *cmd) pairWith(s *transport.Session, id *identity.Identity, name string) error {
	chosen := name
	got, err := pairing.Run(c.ctx, s, id, c.trustStore(), pairing.Prompts{
		Confirm: func(local, remote identity.Fingerprint) bool {
			c.say("%s", c.e.Tr.T("cli.pair_compare", local.Grouped(), remote.Grouped()))
			return c.e.ask(c.stdout, c.e.Tr.T("cli.pair_confirm"))
		},
		Name: func(identity.Fingerprint) string {
			if chosen != "" {
				return chosen
			}
			return c.e.readLine(c.stdout, c.e.Tr.T("cli.pair_name"))
		},
	})
	if errors.Is(err, pairing.ErrCancelled) {
		return cliError{code: exitRefused, key: "cli.pair_cancelled"}
	}
	if err != nil {
		return err
	}
	c.say("%s", c.e.Tr.T("cli.pair_done", got))
	return nil
}

func (c *cmd) sendCmd(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	to := fs.String("to", "", "")
	fromStdin := fs.Bool("stdin", false, "")
	if err := fs.Parse(args); err != nil {
		return usage("cli.uso")
	}
	if *to == "" {
		return usage("cli.flag_to_required")
	}
	spec, err := c.sendSpec(fs.Args(), *fromStdin)
	if err != nil {
		return err
	}
	ep, err := c.resolve(*to)
	if err != nil {
		return err
	}
	id, err := c.identity()
	if err != nil {
		return err
	}
	s, err := transport.Dial(c.ctx, ep, id, c.trustStore())
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	if s.State() != protocol.StateTrusted {
		return cliError{code: exitRefused, key: "cli.not_trusted", args: []any{*to}}
	}
	res, err := transfer.Send(c.ctx, s, spec, c.progress(spec))
	if err != nil {
		return err
	}
	if spec.Kind == transfer.KindFile {
		c.say("%s", c.e.Tr.T("cli.sent_files", len(spec.Paths)))
	} else {
		c.say("%s", c.e.Tr.T("cli.sent_text"))
	}
	_ = res
	return nil
}

func (c *cmd) sendSpec(paths []string, fromStdin bool) (transfer.SendSpec, error) {
	if fromStdin {
		if len(paths) != 0 {
			return transfer.SendSpec{}, usage("cli.uso")
		}
		data, err := io.ReadAll(io.LimitReader(c.e.In, limits.MaxTextBytes+1))
		if err != nil {
			return transfer.SendSpec{}, err
		}
		if len(data) > limits.MaxTextBytes {
			return transfer.SendSpec{}, cliError{code: exitUsage, key: "cli.stdin_too_large"}
		}
		return transfer.SendSpec{Kind: transfer.KindText, Text: string(data)}, nil
	}
	if len(paths) == 0 {
		return transfer.SendSpec{}, usage("cli.uso")
	}
	return transfer.SendSpec{Kind: transfer.KindFile, Paths: paths}, nil
}

// progress prints a line to stderr at most every mebibyte and at the end.
func (c *cmd) progress(spec transfer.SendSpec) func(transfer.Progress) {
	if spec.Kind != transfer.KindFile {
		return nil
	}
	var last int64 = -1 << 20
	return func(p transfer.Progress) {
		if p.TotalSent-last < 1<<20 && p.TotalSent != p.TotalSize {
			return
		}
		last = p.TotalSent
		c.warn("cli.progress", p.Name, p.TotalSent, p.TotalSize)
	}
}

// resolve turns a destination into an address: an IP:port is used directly, a
// device name is looked up with a short discovery scan.
func (c *cmd) resolve(dest string) (netip.AddrPort, error) {
	if ep, err := discovery.ParseEndpoint(dest); err == nil {
		return ep, nil
	}
	devices, err := c.scan(2 * time.Second)
	if err != nil {
		return netip.AddrPort{}, err
	}
	var match []discovery.Device
	for _, d := range devices {
		if strings.EqualFold(d.Name, dest) {
			match = append(match, d)
		}
	}
	switch len(match) {
	case 0:
		return netip.AddrPort{}, cliError{code: exitFailure, key: "cli.dest_not_found", args: []any{dest}}
	case 1:
		return match[0].Addr, nil
	default:
		return netip.AddrPort{}, cliError{code: exitUsage, key: "cli.dest_ambiguous", args: []any{dest}}
	}
}

// targetList is a repeatable --target flag holding explicit discovery addresses.
type targetList []netip.AddrPort

func (t *targetList) String() string { return fmt.Sprint(*t) }

func (t *targetList) Set(v string) error {
	ep, err := discovery.ParseEndpoint(v)
	if err != nil {
		return err
	}
	*t = append(*t, ep)
	return nil
}

func (c *cmd) discoverCmd(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var targets targetList
	fs.Var(&targets, "target", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return usage("cli.uso")
	}
	devices, err := c.scanWith(2*time.Second, targets)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		c.say("%s", c.e.Tr.T("cli.discover_none"))
		return nil
	}
	for _, d := range devices {
		c.say("%s", c.e.Tr.T("cli.discover_item", d.Name, d.Addr.String()))
	}
	return nil
}

// scan sends a discovery query to every local broadcast domain and collects
// answers for the given time.
func (c *cmd) scan(wait time.Duration) ([]discovery.Device, error) {
	return c.scanWith(wait, nil)
}

// scanWith is scan plus explicit targets, for networks where broadcast is blocked.
func (c *cmd) scanWith(wait time.Duration, extra []netip.AddrPort) ([]discovery.Device, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	targets := append(discovery.BroadcastTargets(discovery.DefaultPort, addrs), extra...)
	sc, err := discovery.NewScanner(targets)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sc.Close() }()

	ctx, cancel := context.WithTimeout(c.ctx, wait)
	defer cancel()
	found := make(chan discovery.Announcement, 32)
	go func() { _ = sc.Run(ctx, found) }()
	if err := sc.Query(time.Now()); err != nil {
		return nil, err
	}
	reg := discovery.NewRegistry(time.Minute, limits.MaxDiscoveredDevices)
	for {
		select {
		case a := <-found:
			_ = reg.Observe(a, time.Now())
		case <-ctx.Done():
			return reg.List(time.Now()), nil
		}
	}
}
