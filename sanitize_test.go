package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeepAsSubsequence(t *testing.T) {
	tests := []struct {
		name     string
		original string
		edited   string
		want     string
		ok       bool
	}{
		{"no change", "a\nb\nc\n", "a\nb\nc\n", "a\nb\nc\n", true},
		{"delete middle", "a\nb\nc\n", "a\nc\n", "a\nc\n", true},
		{"delete first", "a\nb\nc\n", "b\nc\n", "b\nc\n", true},
		{"delete last", "a\nb\nc\n", "a\nb\n", "a\nb\n", true},
		{"delete all", "a\nb\n", "", "", true},
		{"delete duplicates", "a\na\nb\n", "a\nb\n", "a\nb\n", true},
		{"no trailing newline kept", "a\nb", "a\n", "a\n", true},
		{"no trailing newline dropped", "a\nb", "a", "a\n", true},
		{"crlf preserved", "a\r\nb\r\n", "a\r\n", "a\r\n", true},
		{"add line", "a\nb\n", "a\nx\nb\n", "", false},
		{"modify line", "a\nb\n", "a\nB\n", "", false},
		{"reorder", "a\nb\nc\n", "b\na\nc\n", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTexts, origRaws := parseLines([]byte(tt.original))
			editedTexts, _ := parseLines([]byte(tt.edited))
			kept, ok := keepAsSubsequence(origTexts, origRaws, editedTexts)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok {
				if got := strings.Join(kept, ""); got != tt.want {
					t.Fatalf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestApplyDeletionOnlyWritesKeptLines(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "f.txt", "a\nc\n")
	original := map[string][]byte{"f.txt": []byte("a\nb\nc\n")}
	edited, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDeletionOnly(original, edited, dir, dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "f.txt"); got != "a\nc\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyDeletionOnlyDropsUserAdditions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "f.txt", "a\nc\nextra\n")
	original := map[string][]byte{"f.txt": []byte("a\nb\nc\n")}
	edited, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDeletionOnly(original, edited, dir, dir); err == nil {
		t.Fatal("expected error for added line")
	}
}

func TestApplyDeletionOnlyRejectsAddedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "new.txt", "hello\n")
	original := map[string][]byte{}
	edited, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDeletionOnly(original, edited, dir, dir); err == nil {
		t.Fatal("expected error for added file")
	}
}

// A file with no parent version (an addition) stays deleted when removed in
// the editor, so the whole addition moves to the new commit.
func TestApplyDeletionOnlyAddedFileStaysDeleted(t *testing.T) {
	dir := t.TempDir()
	original := map[string][]byte{"f.txt": []byte("a\nb\n")}
	edited, err := readTree(dir) // f.txt absent
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDeletionOnly(original, edited, dir, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "f.txt")); !os.IsNotExist(err) {
		t.Fatal("expected f.txt to stay deleted")
	}
}

// Deleting a modified file in the editor restores its parent version so the
// file's diff moves to the new commit.
func TestApplyDeletionOnlyWholeFileRevertsToParent(t *testing.T) {
	leftDir := t.TempDir()
	rightDir := t.TempDir()
	writeFile(t, leftDir, "f.txt", "alpha\nbeta\n")
	if err := os.Chmod(filepath.Join(leftDir, "f.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := map[string][]byte{"f.txt": []byte("alpha\nBETA\n")}
	edited, err := readTree(rightDir) // f.txt deleted in the editor
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDeletionOnly(original, edited, leftDir, rightDir); err != nil {
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

func TestReadTreeSkipsInstructions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, instructionsFile, "ignore me\n")
	writeFile(t, dir, "keep.txt", "x\n")
	tree, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tree[instructionsFile]; ok {
		t.Fatal("JJ-INSTRUCTIONS should be skipped")
	}
	if _, ok := tree["keep.txt"]; !ok {
		t.Fatal("keep.txt should be read")
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
