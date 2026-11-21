package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/i18n"
)

// env groups the external dependencies of the CLI, so commands can be tested
// without touching the real home directory, network or terminal.
type env struct {
	Tr *i18n.Bundle
	// ConfigPath is the settings file (language).
	ConfigPath string
	// Home holds identity.key, trust.json and the staging directory.
	Home string
	// DiscoveryAddr is the UDP address for discovery answers. Empty disables it.
	DiscoveryAddr string
	// In is the source of answers to prompts; Stdin in production.
	In io.Reader
	// NoBroadcast stops discovery from querying the local network by broadcast,
	// so only explicit targets are asked. Tests set it, so that they never reach
	// the devices that happen to be on the real network.
	NoBroadcast bool
	// Ready, when set, receives the address the TLS listener bound to.
	Ready func(netip.AddrPort)
	// ReadyDiscovery, when set, receives the address the discovery listener bound to.
	ReadyDiscovery func(netip.AddrPort)

	reader *bufio.Reader
}

// cliError carries an exit code and the translation key of the message to show.
type cliError struct {
	code int
	key  string
	args []any
}

func (e cliError) Error() string { return e.key }

const (
	exitOK       = 0
	exitFailure  = 1
	exitUsage    = 2
	exitRefused  = 3
	exitCanceled = 130
)

func usage(key string) error {
	return cliError{code: exitUsage, key: key}
}

// report maps an error to an exit code and the text to show, translated.
func report(tr *i18n.Bundle, err error) (int, string) {
	if err == nil {
		return exitOK, ""
	}
	var ce cliError
	if errors.As(err, &ce) {
		return ce.code, tr.T(ce.key, ce.args...)
	}
	switch {
	case errors.Is(err, apperr.ErrCanceled):
		return exitCanceled, tr.T("cli.canceled")
	case errors.Is(err, apperr.ErrDeclined), errors.Is(err, apperr.ErrBusy), errors.Is(err, apperr.ErrUnauthorized):
		return exitRefused, tr.T(apperr.MessageKey(err))
	default:
		return exitFailure, tr.T("cli.error", tr.T(apperr.MessageKey(err)))
	}
}

// ask shows a question and reports whether the answer is yes. Anything other
// than "y" or "yes" is no, including end of input.
func (e *env) ask(out io.Writer, question string) bool {
	answer := e.readLine(out, question)
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes" || a == "s" || a == "sim"
}

func (e *env) readLine(out io.Writer, question string) string {
	_, _ = fmt.Fprint(out, question)
	line, err := e.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}

// errKey returns the translation key of a configuration error.
func errKey(err error) string {
	return apperr.MessageKey(err)
}
