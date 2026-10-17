# DEVELOPMENT

## Architecture

```text
jj                                       jj-split-cli                                   editor
 │  jj split                               │                                              │
 │── materialize sparse left/ + right/ ───▶│                                              │
 │                                         │── read left/ + right/ ──────────────▶ in-memory trees
 │                                         │── lineDiff(left, right) → render hunks/full
 │                                         │── overwrite right/ with the diff view
 │                                         │── launch JJ_SPLIT_EDITOR(right) ─────────────▶│ user deletes -/+ lines
 │                                         │◀─ editor exits ──────────────────────────────│
 │                                         │── read right/ ──────────────────────▶ edited diff view
 │                                         │── markKept + reconstruct → first-commit tree
 │                                         │── overwrite right/ with plain content
 │◀─ snapshot right/ ───────────────────────│
 │  first commit = reconstructed tree, new commit = the changes deleted
```

The external boundary is jj's diff-editor protocol, documented in
[jj-diff-editor-protocol.md](jj-diff-editor-protocol.md). Everything downstream
of `right/` is ours to shape; jj only cares about the tree we leave behind.

## Data flow

1. `run()` → `parseArgs()` → `runEdit()`.
2. `currentOperationID()` → remember the pre-edit jj operation for the undo hint.
3. `readTree(leftDir)` + `readTree(rightDir)` → parent and target trees in memory.
4. `buildPlan()` per changed text file → `lineDiff()` then render hunks or full
   into `right/`, keeping the ops and rendered lines for reconstruction.
5. `openEditor(rightDir)` → run `$JJ_SPLIT_EDITOR <right>`, block until exit.
6. `readTree(rightDir)` → the edited diff views.
7. `applyEdits()` → `markKept()` the deleted rendered lines, `reconstruct()` the
   first-commit file, and write plain content back into `right/`.
8. `printUndoHint()` → print the pre-edit operation id.
9. Tool exits `0` → jj snapshots `right/`.

## Key design decisions

**The editor edits a rendered patch; we reconstruct the tree from it.**
The read-back tree becomes the first commit, so the tool must leave real file
content in `right/`. It overwrites `right/` with a diff view, then after the
editor exits, rebuilds plain content from the original left/right lines plus the
changes the user deleted.

**Deleting a `+` moves the addition; deleting a `-` moves the removal.**
Default (nothing deleted) reproduces the target tree, so all changes stay in the
first commit. A deleted `+` line is dropped; a deleted `-` line is restored. This
is the mirror of "delete a change to move it to the new commit" and keeps a
single rule across both markers.

**Common lines and hunk headers are not selectable.**
`reconstruct` always emits common (` `) lines and ignores deletions of `@@`
headers. Only `-`/`+` lines carry a change. Deleting a context line is therefore
a no-op rather than an error, which keeps the common case forgiving.

**Only deletions are honored; everything else aborts.**
`markKept` matches the edited lines as a subsequence of the rendered lines. A
line that can't be matched (added, modified, or reordered) fails the whole run
before any write, so a stray edit can't silently corrupt the split.

**Output bytes come from the original lines, never the editor.**
`parseLines` keeps each line's raw terminator alongside its comparison text, and
`reconstruct` emits the original raw line. CRLF, trailing newlines, and any EOL
normalization the editor does cannot leak into the result.

**`lineDiff` is Myers with a bounded fallback.**
Myers is O(ND), so the common "big file, small change" case uses a tiny trace.
`traceMemoryCap` guards the rare huge rewrite and falls back to
`blockDiff` (common prefix/suffix + replace block), which is correct but not
minimal.

**Whole-file deletion reverts to the parent version.**
Deleting the rendered file restores `left/<path>` into `right/`, so the file
looks unchanged to the first commit and its entire diff moves to the new commit.
Added files have no parent version, so they stay absent.

**Editor selection is a keyword table plus a raw-command fallback.**
`builtinEditors` hard-codes the wait flags for common editors (`code --wait
--new-window`, `zed --wait --new`, ...). Anything that isn't a keyword is passed
through as a command so users are never boxed in. `JJ_SPLIT_EDITOR` overrides
`--editor`, mirroring jj's own `$JJ_EDITOR` > `ui.editor` precedence so a shell
env var can override a config-baked `--editor`.

**Binary files are skipped.**
A file containing a NUL byte is left untouched and restored on exit, because
rendering it as text would corrupt it.

**The undo hint is a best-effort read-only jj query.**
`currentOperationID` runs `jj --ignore-working-copy --no-pager op log -n 1`
before the editor opens. Restoring that id undoes the edit because jj has not yet
committed the edit's own operation. `--ignore-working-copy` keeps the query from
snapshotting or locking, and any failure (no `jj` on `PATH`, cwd not in a repo)
yields an empty id and no hint rather than an error.

## Maintenance

- Hunk headers are cosmetic; `reconstruct` never depends on their line numbers.
- `--full` and hunks share the same `ops`, so reconstruction is identical either
  way; only rendering differs.
- `code --wait --new-window` blocks until the *window* closes, not a tab. Users
  who prefer a tab close should override `JJ_SPLIT_EDITOR` with
  `code --reuse-window`.
- `JJ-INSTRUCTIONS` is skipped by name; jj deletes it before snapshotting.
- If jj changes the diff-editor protocol, update `docs/jj-diff-editor-protocol.md`
  and `readTree`/`runEdit` together.
