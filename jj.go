package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// currentOperationID returns the id of the latest jj operation. The tool runs
// before jj commits the diff edit's own operation, so restoring to this id
// undoes the edit. The query is read-only (--ignore-working-copy) and so never
// disturbs the in-flight operation. It returns "" when jj is unavailable or the
// working directory is not in a jj repo.
func currentOperationID() string {
	out, err := exec.Command(
		"jj", "--ignore-working-copy", "--no-pager",
		"op", "log", "-n", "1", "--no-graph", "-T", "id.short()",
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// printUndoHint writes a one-line hint telling the user how to undo the edit.
func printUndoHint(opID string) {
	if opID == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "jj-split-cli: to undo, run: jj op restore %s\n", opID)
}
