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

func TestOpenEditorPassesDirAndFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor is POSIX only")
	}
	base := t.TempDir()
	capture := filepath.Join(base, "args.txt")
	t.Setenv("CAPTURE", capture)

	// Put the editor in a directory with a space to prove quoted commands work.
	editorDir := filepath.Join(base, "editor bin")
	if err := os.MkdirAll(editorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	editor := filepath.Join(editorDir, "capture.sh")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CAPTURE\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JJ_SPLIT_EDITOR", fmt.Sprintf("%q --custom-flag", editor))

	rightDir := filepath.Join(base, "right dir")
	if err := os.MkdirAll(rightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := openEditor(rightDir); err != nil {
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

func TestOpenEditorRejectsUnterminatedQuote(t *testing.T) {
	t.Setenv("JJ_SPLIT_EDITOR", `code "unterminated`)
	if err := openEditor(t.TempDir()); err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}
