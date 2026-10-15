package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// defaultEditor opens one window per run and blocks until it is closed.
const defaultEditor = "code --wait --new-window"

// openEditor launches the user's code editor on the temp diff directory and
// waits for it to exit. Override the command (flags included) with
// JJ_SPLIT_EDITOR; the directory is appended as the final argument.
func openEditor(dir string) error {
	command := defaultEditor
	if custom := os.Getenv("JJ_SPLIT_EDITOR"); custom != "" {
		command = custom
	}
	fields, err := splitCommand(command)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("JJ_SPLIT_EDITOR is empty")
	}
	args := append(append([]string{}, fields[1:]...), dir)
	cmd := exec.Command(fields[0], args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %q: %w", command, err)
	}
	return nil
}

// splitCommand splits a command line into arguments, honoring single and
// double quotes so editor paths containing spaces work. It does not expand
// variables or interpret backslash escapes.
func splitCommand(command string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inField := false
	var quote rune
	for _, r := range command {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inField = true
		case r == ' ' || r == '\t' || r == '\n':
			if inField {
				fields = append(fields, current.String())
				current.Reset()
				inField = false
			}
		default:
			current.WriteRune(r)
			inField = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in %q", command)
	}
	if inField {
		fields = append(fields, current.String())
	}
	return fields, nil
}
