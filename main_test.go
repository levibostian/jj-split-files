package main

import (
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
	// The file is rendered as an all-additions diff; delete the +beta line.
	t.Setenv("JJ_SPLIT_EDITOR", writeShellScript(t,
		"grep -v '^+beta$' \"$1/f.txt\" > \"$1/f.txt.tmp\"\n"+
			"mv \"$1/f.txt.tmp\" \"$1/f.txt\"\n",
	))

	if err := run([]string{leftDir, rightDir}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, rightDir, "f.txt"); got != "alpha\ngamma\n" {
		t.Fatalf("got %q, want %q", got, "alpha\ngamma\n")
	}
}

func TestRunDeletingMinusLineDefersRemoval(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	leftDir, rightDir := tempDirs(t)
	writeFile(t, leftDir, "f.txt", "alpha\nbeta\ngamma\n")
	writeFile(t, rightDir, "f.txt", "alpha\nBETA\ngamma\n")
	// Deleting the -beta line moves the removal to the new commit, so the first
	// commit keeps both beta and BETA.
	t.Setenv("JJ_SPLIT_EDITOR", writeShellScript(t,
		"grep -v '^-beta$' \"$1/f.txt\" > \"$1/f.txt.tmp\"\n"+
			"mv \"$1/f.txt.tmp\" \"$1/f.txt\"\n",
	))

	if err := run([]string{leftDir, rightDir}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, rightDir, "f.txt"); got != "alpha\nbeta\nBETA\ngamma\n" {
		t.Fatalf("got %q, want %q", got, "alpha\nbeta\nBETA\ngamma\n")
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
