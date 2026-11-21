// Command tdrop é a interface de linha de comando do Tomorrow Drop.
package main

import (
	"fmt"
	"io"
	"os"

	"tomorrow-drop/internal/protocol"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		_, _ = fmt.Fprintf(stdout, "tdrop %s (protocolo %d)\n", version, protocol.CurrentVersion)
		return 0
	}
	_, _ = fmt.Fprintln(stderr, "uso: tdrop version")
	return 2
}
