// Command jj-split-cli is an external diff editor for jj.
//
// jj invokes it as `jj-split-cli <left-dir> <right-dir>` (see
// docs/jj-diff-editor-protocol.md). It opens the changed files in your code
// editor, lets you delete lines, and treats the deleted lines as the content of
// the new commit that `jj split` creates.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jj-split-cli:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: %s <left-dir> <right-dir>", os.Args[0])
	}
	// left (args[0]) is the before tree, used when a whole-file delete should
	// revert the file to its parent version. right (args[1]) is what jj reads back.
	leftDir, rightDir := filepath.Clean(args[0]), filepath.Clean(args[1])

	original, err := readTree(rightDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rightDir, err)
	}

	if err := openEditor(rightDir); err != nil {
		return fmt.Errorf("editor: %w", err)
	}

	edited, err := readTree(rightDir)
	if err != nil {
		return fmt.Errorf("reading %s after editing: %w", rightDir, err)
	}

	return applyDeletionOnly(original, edited, leftDir, rightDir)
}
