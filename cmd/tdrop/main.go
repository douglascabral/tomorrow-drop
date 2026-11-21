// Command tdrop is the Tomorrow Drop command-line interface.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/settings"
)

var version = "dev"

func main() {
	ctx, stop := signalContext()
	defer stop()

	path, _ := settings.DefaultPath()
	cfg, err := settings.Load(path)
	if err != nil {
		cfg = settings.Default
	}
	tr, err := i18n.Load(i18n.Resolve(cfg.Language, i18n.DetectSystem()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
	e := env{
		Tr:            tr,
		ConfigPath:    path,
		Home:          defaultHome(),
		DiscoveryAddr: "0.0.0.0:" + strconv.Itoa(discovery.DefaultPort),
		In:            os.Stdin,
	}
	code := runCtx(ctx, os.Args[1:], os.Stdout, os.Stderr, e)
	stop()
	os.Exit(code)
}

// defaultHome is the per-user directory for identity, trust and staging files.
func defaultHome() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tomorrow-drop")
	}
	return filepath.Join(dir, "tomorrow-drop")
}

func run(args []string, stdout, stderr io.Writer, e env) int {
	return runCtx(context.Background(), args, stdout, stderr, e)
}

func runCtx(ctx context.Context, args []string, stdout, stderr io.Writer, e env) int {
	if e.In == nil {
		e.In = strings.NewReader("")
	}
	e.reader = bufio.NewReader(e.In)

	switch {
	case len(args) >= 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h"):
		_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.help"))
		return exitOK
	case len(args) == 1 && args[0] == "version":
		_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.versao", version, protocol.CurrentVersion))
		return exitOK
	case len(args) == 2 && args[0] == "config" && args[1] == "language":
		return showLanguage(stdout, stderr, &e)
	case len(args) == 3 && args[0] == "config" && args[1] == "language":
		return saveLanguage(args[2], stdout, stderr, &e)
	}

	c := &cmd{ctx: ctx, stdout: stdout, stderr: stderr, e: &e}
	code, msg := report(e.Tr, c.dispatch(args))
	if msg != "" {
		_, _ = fmt.Fprintln(stderr, msg)
	}
	return code
}

func showLanguage(stdout, stderr io.Writer, e *env) int {
	cfg, err := settings.Load(e.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.erro_ler", e.Tr.T(errKey(err))))
		return exitFailure
	}
	_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.idioma_atual", cfg.Language))
	return exitOK
}

func saveLanguage(value string, stdout, stderr io.Writer, e *env) int {
	if !i18n.ValidPreference(value) {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.idioma_invalido", value))
		return exitUsage
	}
	if err := settings.Save(e.ConfigPath, settings.Settings{Language: value}); err != nil {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.erro_salvar", e.Tr.T(errKey(err))))
		return exitFailure
	}
	_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.idioma_salvo", value))
	return exitOK
}

// signalContext cancels the command on Ctrl-C, closing sessions and listeners.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}
