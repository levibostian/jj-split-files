package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"plain", "code --wait --new-window", []string{"code", "--wait", "--new-window"}, false},
		{"collapses whitespace", "code   --wait\t--new-window", []string{"code", "--wait", "--new-window"}, false},
		{"double quoted path with spaces", `"/Applications/Visual Studio Code.app/bin/code" --wait`, []string{"/Applications/Visual Studio Code.app/bin/code", "--wait"}, false},
		{"single quoted path with spaces", `'/Applications/Visual Studio Code.app/bin/code' --wait`, []string{"/Applications/Visual Studio Code.app/bin/code", "--wait"}, false},
		{"quotes inside a field", `code --arg="a b"`, []string{"code", "--arg=a b"}, false},
		{"empty", "", nil, false},
		{"unterminated double quote", `code "oops`, nil, true},
		{"unterminated single quote", `code 'oops`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCommand(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"vscode", "code --wait --new-window"},
		{"VSCODE", "code --wait --new-window"},
		{"  zed  ", "zed --wait --new"},
		{"subl", "subl -n -w"},
		{"neovim", "nvim"},
		{"myeditor --flag", "myeditor --flag"},
		{"/path/to/editor", "/path/to/editor"},
	}
	for _, tt := range tests {
		if got := resolveEditor(tt.in); got != tt.want {
			t.Fatalf("resolveEditor(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// writeCaptureEditor writes a script in editorDir that records its arguments to
// capture, and returns the script path.
func writeCaptureEditor(t *testing.T, editorDir, capture string) string {
	t.Helper()
	if err := os.MkdirAll(editorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + capture + "'\n"
	editor := filepath.Join(editorDir, "capture.sh")
	if err := os.WriteFile(editor, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return editor
}

func TestOpenEditorUsesSpec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	t.Setenv("JJ_SPLIT_EDITOR", "") // ensure the spec is what runs
	base := t.TempDir()
	capture := filepath.Join(base, "args.txt")
	// Put the editor in a directory with a space to prove quoted commands work.
	editor := writeCaptureEditor(t, filepath.Join(base, "editor bin"), capture)

	rightDir := filepath.Join(base, "right dir")
	if err := os.MkdirAll(rightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := openEditor(rightDir, fmt.Sprintf("%q --custom-flag", editor)); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	want := "--custom-flag\n" + rightDir + "\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestOpenEditorEnvOverridesSpec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	base := t.TempDir()
	capture := filepath.Join(base, "args.txt")
	editor := writeCaptureEditor(t, filepath.Join(base, "bin"), capture)
	t.Setenv("JJ_SPLIT_EDITOR", editor)

	rightDir := filepath.Join(base, "right")
	if err := os.MkdirAll(rightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The spec would fail to execute, so success proves the env var won.
	if err := openEditor(rightDir, "/nonexistent/spec-editor"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(capture); err != nil {
		t.Fatalf("env editor did not run: %v", err)
	}
}

func TestOpenEditorRejectsUnterminatedQuote(t *testing.T) {
	t.Setenv("JJ_SPLIT_EDITOR", "")
	if err := openEditor(t.TempDir(), `code "unterminated`); err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}
