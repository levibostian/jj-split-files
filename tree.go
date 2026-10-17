package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// instructionsFile is jj's synthetic instruction file. jj deletes it before
// snapshotting, and edits to it are ignored, so the tool skips it entirely.
const instructionsFile = "JJ-INSTRUCTIONS"

// readTree reads every regular file under root into memory, keyed by its
// slash-separated path relative to root. Symlinks are skipped because they
// can't be line-edited.
func readTree(root string) (map[string][]byte, error) {
	tree := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == instructionsFile {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tree[rel] = content
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func sortedKeys(tree map[string][]byte) []string {
	keys := make([]string, 0, len(tree))
	for key := range tree {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// writeTreeFile overwrites a file inside dir. A zero mode keeps the existing
// file's permissions; a non-zero mode is applied after writing (used when
// restoring a file that was deleted in the editor).
func writeTreeFile(dir, rel string, content []byte, mode os.FileMode) error {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	perm := mode
	if perm == 0 {
		perm = 0o644
	}
	if err := os.WriteFile(path, content, perm); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	if mode != 0 {
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("setting mode on %s: %w", rel, err)
		}
	}
	return nil
}
