package main

import (
	"io"
	"os"
	"testing"
)

func TestCurrentOperationIDWithoutJJ(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no jj binary available
	if got := currentOperationID(); got != "" {
		t.Fatalf("got %q, want empty when jj is unavailable", got)
	}
}

func TestPrintUndoHint(t *testing.T) {
	write := func(opID string) string {
		t.Helper()
		old := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		printUndoHint(opID)
		w.Close()
		os.Stdout = old
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}

	if got := write(""); got != "" {
		t.Fatalf("empty id should print nothing, got %q", got)
	}
	want := "jj-split-cli: to undo, run: jj op restore abc123\n"
	if got := write("abc123"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
