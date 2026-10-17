package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type options struct {
	leftDir  string
	rightDir string
	full     bool
	context  int
}

// parseArgs reads optional flags followed by the two directory operands jj
// passes. Flags let the user pick full-file rendering or a custom context size
// from their jj config, e.g. ["jj-split-cli", "--full", "$left", "$right"].
func parseArgs(args []string) (options, error) {
	opts := options{context: 3}
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--full":
			opts.full = true
		case arg == "--hunks":
			opts.full = false
		case arg == "--context":
			if i+1 >= len(args) {
				return options{}, fmt.Errorf("--context needs a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				return options{}, fmt.Errorf("invalid --context value %q", args[i])
			}
			opts.context = n
		case strings.HasPrefix(arg, "--context="):
			value := strings.TrimPrefix(arg, "--context=")
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 {
				return options{}, fmt.Errorf("invalid --context value %q", value)
			}
			opts.context = n
		case strings.HasPrefix(arg, "-"):
			return options{}, fmt.Errorf("unknown flag %q", arg)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		return options{}, fmt.Errorf("usage: %s [--full] [--context N] <left-dir> <right-dir>", os.Args[0])
	}
	opts.leftDir = positional[0]
	opts.rightDir = positional[1]
	return opts, nil
}
