package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// runEdit renders the diff into the writable tree, opens the editor, then
// rebuilds the first-commit tree from the changes the user deleted.
func runEdit(opts options) error {
	leftDir := filepath.Clean(opts.leftDir)
	rightDir := filepath.Clean(opts.rightDir)

	leftTree, err := readTree(leftDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", leftDir, err)
	}
	rightTree, err := readTree(rightDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rightDir, err)
	}

	plans := make(map[string]filePlan)
	for _, rel := range sortedKeys(rightTree) {
		if isBinary(leftTree[rel]) || isBinary(rightTree[rel]) {
			continue
		}
		plan, ok := buildPlan(rel, leftTree[rel], rightTree[rel], opts.full, opts.context)
		if !ok {
			continue
		}
		plans[rel] = plan
		if err := writeTreeFile(rightDir, rel, plan.rendered, 0); err != nil {
			return err
		}
	}

	if err := openEditor(rightDir); err != nil {
		return fmt.Errorf("editor: %w", err)
	}

	edited, err := readTree(rightDir)
	if err != nil {
		return fmt.Errorf("reading %s after editing: %w", rightDir, err)
	}

	return applyEdits(leftDir, rightDir, leftTree, rightTree, plans, edited)
}

type pendingWrite struct {
	rel     string
	content []byte
	mode    os.FileMode // 0 keeps the existing file's mode
}

// applyEdits turns the edited diff views back into the first-commit tree. It
// plans every write first so an unsupported edit aborts before anything changes.
func applyEdits(leftDir, rightDir string, leftTree, rightTree map[string][]byte, plans map[string]filePlan, edited map[string][]byte) error {
	for _, rel := range sortedKeys(edited) {
		if _, ok := rightTree[rel]; !ok {
			return fmt.Errorf("only deleting + and - lines is supported, but a file was added: %s", rel)
		}
	}

	var writes []pendingWrite
	for _, rel := range sortedKeys(rightTree) {
		plan, planned := plans[rel]
		if !planned {
			// Binary or unchanged file: restore the original so stray edits are ignored.
			writes = append(writes, pendingWrite{rel: rel, content: rightTree[rel]})
			continue
		}
		editedContent, present := edited[rel]
		if !present {
			// Whole-file delete reverts to the parent version, moving the file's
			// diff to the new commit. Added files have no parent, so they stay absent.
			if leftContent, ok := leftTree[rel]; ok {
				info, err := os.Stat(filepath.Join(leftDir, filepath.FromSlash(rel)))
				if err != nil {
					return fmt.Errorf("reading %s: %w", rel, err)
				}
				writes = append(writes, pendingWrite{rel: rel, content: leftContent, mode: info.Mode().Perm()})
			}
			continue
		}
		content, err := reconstructEdited(plan, editedContent)
		if err != nil {
			return err
		}
		writes = append(writes, pendingWrite{rel: rel, content: content})
	}

	for _, w := range writes {
		if err := writeTreeFile(rightDir, w.rel, w.content, w.mode); err != nil {
			return err
		}
	}
	return nil
}

func reconstructEdited(plan filePlan, editedContent []byte) ([]byte, error) {
	editedTexts, _ := parseLines(editedContent)
	original := make([]string, len(plan.renderedLines))
	for i, line := range plan.renderedLines {
		original[i] = line.compare
	}
	kept, ok := markKept(original, editedTexts)
	if !ok {
		return nil, fmt.Errorf("only deleting + and - lines is supported, but %s was added to or edited", plan.rel)
	}
	deleted := make(map[int]bool)
	for i, line := range plan.renderedLines {
		if line.opIndex >= 0 && !kept[i] {
			deleted[line.opIndex] = true
		}
	}
	return reconstruct(plan, deleted), nil
}
