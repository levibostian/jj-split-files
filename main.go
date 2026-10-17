// Command jj-split-cli is an external diff editor for jj.
//
// jj invokes it as `jj-split-cli [--editor NAME] [--full] [--context N]
// <left-dir> <right-dir>`
// (see docs/jj-diff-editor-protocol.md). It renders each changed file as a diff
// with ` `/`-`/`+` prefixes, opens your code editor, and treats every `-` or `+`
// line you delete as a change to move into the new commit that `jj split`
// creates.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jj-split-cli:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseArgs(args)
	if err != nil {
		return err
	}
	return runEdit(opts)
}
