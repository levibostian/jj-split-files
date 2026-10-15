# DEVELOPMENT

## Architecture

```text
jj                                      jj-split-cli                         editor
 │  jj split                              │                                   │
 │── materialize sparse left/ + right/ ──▶│                                   │
 │                                        │── read right/ (original) ───▶ in-memory snapshot
 │                                        │── launch JJ_SPLIT_EDITOR ────────▶│ user deletes lines
 │                                        │◀─ editor exits ──────────────────│
 │                                        │── read right/ (edited) ────▶ in-memory snapshot
 │                                        │── keepAsSubsequence(original, edited)
 │                                        │── write sanitized right/ back
 │◀─ snapshot right/ ──────────────────────│
 │  first commit = right, new commit = deleted lines
```

The external boundary is jj's diff-editor protocol, documented in
[jj-diff-editor-protocol.md](jj-diff-editor-protocol.md). Everything the tool
does is downstream of `right/`.

## Data flow

1. jj runs `jj-split-cli <left> <right>` → `run()` in `main.go`.
2. `readTree(right)` → snapshot original file contents into memory.
3. `openEditor(right)` → run `$JJ_SPLIT_EDITOR <right>`, block until exit.
4. `readTree(right)` → snapshot the edited contents.
5. `applyDeletionOnly(original, edited, left, right)` → compute kept lines per
   file, plan whole-file reverts, reject unsupported edits, write the sanitized
   tree back.
6. Tool exits `0` → jj snapshots `right/` and assigns the removed lines to the
   new commit.

## Key design decisions

**Line deletions only need `right/`; `left/` is read for whole-file deletes.**
For `jj split`, jj materializes `right = target tree`. Because the read-back tree
becomes the first commit and the *complement* becomes the new commit, a line
simply removed from `right/` lands in the new commit automatically — no
inversion. `left/` is needed only to restore a file's parent version when the
user deletes the whole file.

**Deletion-only enforced with a subsequence test, not a diff algorithm.**
`keepAsSubsequence` walks both line lists once with two pointers. If every
edited line can be matched in order against the original, the edit was pure
deletion; otherwise it's an addition or reorder and we reject the whole run.
This is O(n), dependency-free, and makes reorders fail loudly instead of being
silently interpreted as delete+insert.

**Output lines come from the original bytes, not the edited bytes.**
`parseLines` keeps each line's raw terminator alongside its comparison text, and
`keepAsSubsequence` emits the original raw line. This means CRLF, trailing
newlines, and EOL normalization performed by the editor never leak into the
result — only the deletion itself is honored.

**Unsupported edits abort the run instead of being ignored.**
Returning a non-zero exit makes jj discard the entire split. It's louder than
silently dropping a reordered line, which would corrupt the split in a way the
user wouldn't notice.

**Whole-file deletion reverts to the parent version.**
Deleting a file in the editor restores `left/<path>` into `right/`, so the file
looks unchanged to the first commit and its entire diff moves to the new commit.
Added files have no parent version, so they stay absent and the whole addition
moves instead. `left/` is read only for this case.

**Symlinks are skipped.**
They can't be line-edited, and `readTree` leaving them out keeps them untouched
in `right/`.

## Maintenance

- Behavior depends on jj's sparse checkout of only changed files into `right/`.
  If that changes, file discovery changes with it.
- `code --wait --new-window` blocks until the *window* closes, not a tab. Users
  who prefer a tab close should override `JJ_SPLIT_EDITOR` with
  `code --reuse-window`.
- `JJ-INSTRUCTIONS` is skipped by name; jj deletes it before snapshotting.
- If jj changes the diff-editor protocol, update
  `docs/jj-diff-editor-protocol.md` and `readTree`/`applyDeletionOnly` together.
