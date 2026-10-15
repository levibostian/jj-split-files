package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// applyDeletionOnly rewrites rightDir so it equals the original tree with the
// lines the user deleted removed. Anything beyond deleting lines (adding,
// modifying, or reordering) is rejected so the resulting split stays
// predictable.
//
// Two forms of deletion are supported: deleting lines, and deleting a whole
// file. A whole-file delete restores the file's parent version, which moves
// that file's entire diff into the new commit.
func applyDeletionOnly(original, edited map[string][]byte, leftDir, rightDir string) error {
	writes := make(map[string][]byte)
	var reverts []string
	var problems []string

	for _, rel := range sortedKeys(edited) {
		if _, ok := original[rel]; !ok {
			problems = append(problems, "added file: "+rel)
		}
	}

	for _, rel := range sortedKeys(original) {
		editedContent, stillPresent := edited[rel]
		if !stillPresent {
			// Deleting a whole file means "move this file's changes to the new
			// commit": restoring the parent version leaves the file unchanged in
			// the first commit. Added files have no parent version, so they stay
			// absent and the whole addition moves instead.
			if _, err := os.Stat(filepath.Join(leftDir, filepath.FromSlash(rel))); err == nil {
				reverts = append(reverts, rel)
			}
			continue
		}
		origTexts, origRaws := parseLines(original[rel])
		editedTexts, _ := parseLines(editedContent)
		kept, ok := keepAsSubsequence(origTexts, origRaws, editedTexts)
		if !ok {
			problems = append(problems, "added or reordered lines: "+rel)
			continue
		}
		if out := strings.Join(kept, ""); out != string(original[rel]) {
			writes[rel] = []byte(out)
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf(
			"only line deletions are supported, but found:\n  %s\n"+
				"re-run and delete lines without adding or reordering any",
			strings.Join(problems, "\n  "),
		)
	}
	for _, rel := range reverts {
		if err := revertToParent(leftDir, rightDir, rel); err != nil {
			return err
		}
	}
	for _, rel := range sortedKeys(writes) {
		if err := writeTreeFile(rightDir, rel, writes[rel]); err != nil {
			return err
		}
	}
	return nil
}

// revertToParent copies a file's parent (left) version back into the working
// diff (right), preserving its permission bits.
func revertToParent(leftDir, rightDir, rel string) error {
	src := filepath.Join(leftDir, filepath.FromSlash(rel))
	dst := filepath.Join(rightDir, filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rel, err)
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rel, err)
	}
	if err := os.WriteFile(dst, content, info.Mode().Perm()); err != nil {
		return fmt.Errorf("restoring %s: %w", rel, err)
	}
	return nil
}

// keepAsSubsequence returns the original lines that survive after deletions,
// using raw (terminator-preserving) bytes for output. ok is false when edited
// contains a line that cannot be matched in order against original, meaning the
// user added or reordered lines.
func keepAsSubsequence(origTexts, origRaws, editedTexts []string) (kept []string, ok bool) {
	kept = make([]string, 0, len(origRaws))
	next := 0
	for _, edited := range editedTexts {
		for next < len(origTexts) && origTexts[next] != edited {
			next++
		}
		if next == len(origTexts) {
			return nil, false
		}
		kept = append(kept, origRaws[next])
		next++
	}
	return kept, true
}

// parseLines splits content into text (no terminator, for comparison) and raw
// (terminator preserved, for output) lines.
func parseLines(content []byte) (texts, raws []string) {
	s := string(content)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		raw := s[start : i+1]
		raws = append(raws, raw)
		texts = append(texts, trimTerminator(raw))
		start = i + 1
	}
	if start < len(s) {
		raw := s[start:]
		raws = append(raws, raw)
		texts = append(texts, trimTerminator(raw))
	}
	return texts, raws
}

func trimTerminator(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}
