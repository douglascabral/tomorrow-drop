package main

import (
	"flag"
	"io"
)

// parseFlags parses the options of a command wherever they appear: before,
// between or after the arguments ("send a.txt --to X" as well as
// "send --to X a.txt"). The standard library stops at the first argument, so the
// documented form with the options last did not work. "--" ends the options.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var tail []string
	for i, a := range args {
		if a == "--" {
			tail = args[i+1:]
			args = args[:i]
			break
		}
	}
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return append(positional, tail...), nil
}
