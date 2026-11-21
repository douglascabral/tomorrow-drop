// Command tdrop is the Tomorrow Drop command-line interface.
package main

import (
	"fmt"
	"io"
	"os"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/settings"
)

var version = "dev"

// env groups the external dependencies of the CLI, so run can be tested.
type env struct {
	Tr         *i18n.Bundle
	ConfigPath string
}

func main() {
	path, err := settings.DefaultPath()
	if err != nil {
		path = ""
	}
	cfg, err := settings.Load(path)
	if err != nil {
		cfg = settings.Default
	}
	tr, err := i18n.Load(i18n.Resolve(cfg.Language, i18n.DetectSystem()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, env{Tr: tr, ConfigPath: path}))
}

func run(args []string, stdout, stderr io.Writer, e env) int {
	switch {
	case len(args) == 1 && args[0] == "version":
		_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.versao", version, protocol.CurrentVersion))
		return 0
	case len(args) == 2 && args[0] == "config" && args[1] == "language":
		return showLanguage(stdout, stderr, e)
	case len(args) == 3 && args[0] == "config" && args[1] == "language":
		return saveLanguage(args[2], stdout, stderr, e)
	}
	_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.uso"))
	return 2
}

func showLanguage(stdout, stderr io.Writer, e env) int {
	cfg, err := settings.Load(e.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.erro_ler", e.Tr.T(apperr.MessageKey(err))))
		return 1
	}
	_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.idioma_atual", cfg.Language))
	return 0
}

func saveLanguage(value string, stdout, stderr io.Writer, e env) int {
	if !i18n.ValidPreference(value) {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.idioma_invalido", value))
		return 2
	}
	if err := settings.Save(e.ConfigPath, settings.Settings{Language: value}); err != nil {
		_, _ = fmt.Fprintln(stderr, e.Tr.T("cli.erro_salvar", e.Tr.T(apperr.MessageKey(err))))
		return 1
	}
	_, _ = fmt.Fprintln(stdout, e.Tr.T("cli.idioma_salvo", value))
	return 0
}
