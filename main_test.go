package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunValidatesArgCount(t *testing.T) {
	for _, args := range [][]string{nil, {"only-left"}, {"a", "b", "c"}} {
		if err := run(args); err == nil {
			t.Fatalf("run(%v): expected error", args)
		}
	}
}

func TestRunAppliesEditorDeletions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	leftDir, rightDir := tempDirs(t)
	writeFile(t, rightDir, "f.txt", "alpha\nbeta\ngamma\n")
	t.Setenv("JJ_SPLIT_EDITOR", writeShellScript(t,
		"grep -v '^beta$' \"$1/f.txt\" > \"$1/f.txt.tmp\"\n"+
			"mv \"$1/f.txt.tmp\" \"$1/f.txt\"\n",
	))

	if err := run([]string{leftDir, rightDir}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, rightDir, "f.txt"); got != "alpha\ngamma\n" {
		t.Fatalf("got %q, want %q", got, "alpha\ngamma\n")
	}
}

func TestRunRejectsUnsupportedEdits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	leftDir, rightDir := tempDirs(t)
	writeFile(t, rightDir, "f.txt", "alpha\nbeta\n")
	t.Setenv("JJ_SPLIT_EDITOR", writeShellScript(t,
		"printf 'sneaky\\n' >> \"$1/f.txt\"\n",
	))

	if err := run([]string{leftDir, rightDir}); err == nil {
		t.Fatal("expected error for an added line")
	}
}

func tempDirs(t *testing.T) (leftDir, rightDir string) {
	t.Helper()
	base := t.TempDir()
	leftDir = filepath.Join(base, "left")
	rightDir = filepath.Join(base, "right")
	for _, dir := range []string{leftDir, rightDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return leftDir, rightDir
}

// writeShellScript writes an executable POSIX shell script and returns its path.
func writeShellScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -e\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
