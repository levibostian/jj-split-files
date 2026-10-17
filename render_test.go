package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlanHunks(t *testing.T) {
	plan, ok := buildPlan("f.txt", []byte("a\nb\nc\n"), []byte("a\nB\nc\n"), false, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	want := "@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n"
	if got := string(plan.rendered); got != want {
		t.Fatalf("rendered =\n%q\nwant\n%q", got, want)
	}
}

func TestBuildPlanFull(t *testing.T) {
	plan, ok := buildPlan("f.txt", []byte("a\nb\nc\n"), []byte("a\nB\nc\n"), true, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	want := " a\n-b\n+B\n c\n"
	if got := string(plan.rendered); got != want {
		t.Fatalf("rendered =\n%q\nwant\n%q", got, want)
	}
}

func TestBuildPlanNoChange(t *testing.T) {
	if _, ok := buildPlan("f.txt", []byte("a\n"), []byte("a\n"), false, 3); ok {
		t.Fatal("expected no plan for an unchanged file")
	}
}

// renderAndDelete renders a diff, drops the given rendered lines, and rebuilds
// the first-commit file.
func renderAndDelete(t *testing.T, left, right string, full bool, drop ...string) string {
	t.Helper()
	plan, ok := buildPlan("f.txt", []byte(left), []byte(right), full, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	removed := make(map[string]bool, len(drop))
	for _, line := range drop {
		removed[line] = true
	}
	var edited []string
	for _, line := range plan.renderedLines {
		if !removed[line.compare] {
			edited = append(edited, line.compare)
		}
	}
	got, err := reconstructEdited(plan, []byte(strings.Join(edited, "\n")+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

func TestReconstructScenarios(t *testing.T) {
	const left = "a\nb\nc\n"
	const right = "a\nB\nc\n"
	tests := []struct {
		name string
		drop []string
		want string
	}{
		{"keep everything", nil, right},
		{"delete both sides of the change", []string{"-b", "+B"}, left},
		{"delete added line only", []string{"+B"}, "a\nc\n"},
		{"delete removed line only", []string{"-b"}, "a\nb\nB\nc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderAndDelete(t, left, right, true, tt.drop...); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReconstructPreservesCrlf(t *testing.T) {
	plan, ok := buildPlan("f.txt", []byte("a\r\nb\r\n"), []byte("a\r\nB\r\n"), true, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	got, err := reconstructEdited(plan, plan.rendered) // no deletions
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a\r\nB\r\n" {
		t.Fatalf("got %q, want CRLF preserved", got)
	}
}

func TestReconstructRejectsAddedLines(t *testing.T) {
	plan, ok := buildPlan("f.txt", []byte("a\nb\n"), []byte("a\nB\n"), true, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	if _, err := reconstructEdited(plan, []byte(" a\nx\n-b\n+B\n")); err == nil {
		t.Fatal("expected error for an added line")
	}
}

// Deleting the rendered file reverts it to the parent version, preserving the
// parent's permission bits.
func TestApplyEditsWholeFileRevertsToParent(t *testing.T) {
	leftDir, rightDir := tempDirs(t)
	writeFile(t, leftDir, "f.txt", "alpha\nbeta\n")
	if err := os.Chmod(filepath.Join(leftDir, "f.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	leftTree, err := readTree(leftDir)
	if err != nil {
		t.Fatal(err)
	}
	rightTree := map[string][]byte{"f.txt": []byte("alpha\nBETA\n")}
	plan, ok := buildPlan("f.txt", leftTree["f.txt"], rightTree["f.txt"], false, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	if err := applyEdits(leftDir, rightDir, leftTree, rightTree, map[string]filePlan{"f.txt": plan}, map[string][]byte{}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, rightDir, "f.txt"); got != "alpha\nbeta\n" {
		t.Fatalf("got %q, want parent content", got)
	}
	info, err := os.Stat(filepath.Join(rightDir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

// An added file has no parent version, so deleting it leaves it absent and the
// whole addition moves to the new commit.
func TestApplyEditsAddedFileDeleteLeavesAbsent(t *testing.T) {
	leftDir, rightDir := tempDirs(t)
	leftTree := map[string][]byte{}
	rightTree := map[string][]byte{"f.txt": []byte("new\n")}
	plan, ok := buildPlan("f.txt", nil, rightTree["f.txt"], false, 3)
	if !ok {
		t.Fatal("expected a plan")
	}
	if err := applyEdits(leftDir, rightDir, leftTree, rightTree, map[string]filePlan{"f.txt": plan}, map[string][]byte{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rightDir, "f.txt")); !os.IsNotExist(err) {
		t.Fatal("expected f.txt to stay absent")
	}
}
