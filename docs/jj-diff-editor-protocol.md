# jj external diff-editor protocol

Reference spec for building an interactive editor that plugs into `jj split`
(and the other commands that use jj's diff editor). Describes exactly what jj
puts on disk, what it passes to the editor, and what it reads back.

Sources verified against **jj 0.46.0** (`jj help split`, `jj help -k config`,
live probes) and jj source:
`cli/src/merge_tools/{mod.rs,external.rs,diff_working_copies.rs}` and
`cli/src/commands/split.rs`. Reference implementation read:
<https://github.com/mashiro-no-rabo/diffpicker>.

---

## 1. Purpose and layer boundary

An external diff editor owns one responsibility: **turn a diff into an edited
working tree**. jj materializes the before/after trees into temp directories and
runs your program. Your program mutates the writable side. When it exits, jj
snapshots that side and uses the resulting tree as the selection.

This spec covers the **protocol** (config, arguments, directory layout, read-back,
exit codes). It deliberately does not cover TUI design, hunk diffing, or rendering
— those are the editor's own business.

## 2. Non-goals

- Not a merge/conflict-resolution spec (`jj resolve` uses `merge-args` / `$base`
  and a different code path).
- Not the diff *viewing* spec (`diff-formatter` / `diff-args`; viewing tools are
  invoked read-only with both sides read-only).
- Not the built-in `:builtin` (scm-diff-editor, vendored) protocol.

---

## 3. When jj invokes a diff editor

Commands that call the diff editor: `jj split`, `jj commit -i`, `jj squash -i`,
`jj restore -i`, `jj absorb` (interactive paths), and `jj diffedit`. Behavior is
identical across them; only the meaning of the resulting tree differs (see §9).

The editor is invoked **once** per edit operation (once per file in file-by-file
mode).

---

## 4. Configuration resolution

`ui.diff-editor` selects the tool. It may be:

- unset → default `:builtin` (jj prints a hint to set it to `:builtin` to silence);
- a string tool name, e.g. `"meld"` → load `[merge-tools.meld]` config,
  `program` defaults to the tool name;
- a list, e.g. `["/path/to/bin", "--flag", "$left", "$right"]` → command + args,
  `program` defaults to `args[0]`;
- `":builtin"` / `":editor"` / `":ours"` / `":theirs"` are reserved special names.

`jj split --tool NAME` overrides for one run.

Per-tool keys (`[merge-tools.<name>]`):

| Key | Meaning |
|-----|---------|
| `program` | Binary. Defaults to `<name>`. |
| `edit-args` | Args for diff editing. **Default `["$left", "$right"]`.** |
| `edit-invocation-mode` | `"dir"` (default) or `"file-by-file"`. |
| `diff-invocation-mode` | Independent; viewing only. |

`edit-args = []` explicitly disables the tool for diff editing (jj errors with
`EditArgsNotConfigured`). This is how built-ins like `mergeraf` opt out.

**3-pane trigger:** if `$output` appears *anywhere* in `edit-args`, jj switches to
3-pane mode (writes an `output/` dir and makes `right/` read-only). Presence of the
substitution is the switch; there is no separate config flag.

---

## 5. Invocation modes

Paths are passed as **absolute** paths. jj does not set any special environment
variables — the editor infers everything from the path operands and the contents
of the temp tree.

### 5.1 Directory mode (default)

```
<program> ... "$left" "$right"        # 2-pane
<program> ... "$left" "$output" "$right"   # 3-pane, depends on edit-args order
```

- `$left` — directory with the **before** tree (parent revision).
- `$right` — directory with the **after** tree (the revision being edited; all
  changes applied). This is the writable side in 2-pane mode.
- `$output` — directory initially identical to `$right`. This is the writable
  side in 3-pane mode; `$right` is read-only there.

Only **changed paths** are checked out (sparse). Unchanged files do not appear.

### 5.2 File-by-file mode

`edit-invocation-mode = "file-by-file"` runs the program once per changed file:

```
<program> ... "$left/f.txt" "$right/f.txt"
```

- Operands are **file** paths, not directories.
- One invocation per changed file, in arbitrary order.
- In 3-pane mode, `$output/f.txt` is the writable operand.
- For an added file the `$left` path does **not** exist; for a deleted file the
  `$right` path does **not** exist.
- `JJ-INSTRUCTIONS` is not materialized in this mode.
- A non-zero exit aborts the whole operation (already-written files are still
  discarded because no snapshot happens).

Use `diff-invocation-mode` if viewing should stay directory-based while editing
is per-file, or vice versa.

---

## 6. Temp directory layout

```
<jj temp root>/                 e.g. /tmp/jj-diff-AbCdEf
├── left/                       before tree (read-only)
├── right/                      after tree  (read-only in 3-pane)
├── output/                     writable copy of right (3-pane only)
├── left_state/                 jj TreeState bookkeeping (do not touch)
├── right_state/
└── output_state/               (3-pane only)
```

- Root is created with `tempfile` prefix `jj-diff-`, removed when jj is done.
- `left/`, `right/`, `output/` are the operands; the `*_state/` dirs are jj
  internals. An editor must not walk out of the operand it was given.
- The editor's CWD is **not** the temp root; it is wherever jj was launched.
- Read-only: `left/` always; `right/` too in 3-pane mode. Attempting to write to
  a read-only side fails. `output/` is always writable.

---

## 7. `JJ-INSTRUCTIONS`

A synthetic file jj writes into the writable side (2-pane: `right/`; 3-pane:
both `right/` and `output/`) when `ui.diff-instructions = true` (default):

- It is **deleted before snapshotting**, and edits to it are ignored. Editors
  should skip it when enumerating files.
- Suppressed entirely by `ui.diff-instructions = false`.
- Contains human-readable guidance plus the commit summary. Example for split:

  ```
  You are splitting a commit into two: <change> <commit> (no description set)

  The diff initially shows the changes in the commit you're splitting.

  Adjust the right side until it shows the contents you want to split into the
  new commit.
  The changes that are not selected will replace the original commit.
  ```

- In 3-pane mode the `right/` copy adds `The content of this pane should NOT be
  edited.` and the `output/` copy adds `Please make your edits in this pane.`

---

## 8. Read-back semantics

After the editor exits successfully, jj snapshots the writable tree —
**`output/` if 3-pane, otherwise `right/`** — exactly like a working copy:

- file present → content (and mode) from disk; missing file → removed/absent;
- additions, deletions, renames all derived from a full tree scan;
- executable bit and symlinks preserved (`EolConversionMode::None`, auto exec);
- snapshot uses `force_tracking_matcher = EverythingMatcher` and
  `max_new_file_size = u64::MAX`, i.e. no size cap and nothing left untracked;
- the `JJ-INSTRUCTIONS` path is deleted first.

The resulting tree becomes the "after" tree for the operation. `left/` is never
read back except as the diff baseline.

**Consequence for implementers:** the initial `right/`/`output/` content already
equals "all changes selected". To *deselect* something you must actively revert
it in the writable tree (restore from `left/`, or delete for additions). The
common pattern (diffpicker, scm-diff-editor) is: start with no hunks selected and
rewrite each file to contain only the selected hunks, writing the result back to
the writable tree.

---

## 9. Split semantics (selection → commits)

For `jj split`, `Diff::new(parent_tree, target_tree)` is materialized as
`left=parent`, `right=target`.

- The read-back tree becomes the **first commit** = the *selected* changes. It
  keeps the original revision's change id (unless `-o`/`-A`/`-B`, which generate a
  new change id and move it).
- The **second commit** = remaining changes, parented on the first (default), or a
  sibling with `--parallel`, or relocated with `-o`/`-A`/`-B`.
- Warnings jj prints:
  - all changes selected → "All changes have been selected, so the original
    revision will become empty."
  - nothing selected → "No changes have been selected, so the new revision will
    be empty."
- Splitting an empty commit is rejected (use `jj new`).

Other commands map the same read-back tree differently (e.g. `jj commit -i`
commits the selection, `jj restore -i` restores it), but the directory protocol
is unchanged.

---

## 10. Exit-code protocol

- Exit `0` → success; jj snapshots the writable side.
- Any non-zero exit → operation aborts with `Failed to edit diff / Tool exited
  with exit status: N`; **no changes are committed**. Draft edits to the temp tree
  are discarded with the temp dir.
- In directory mode the writable side is the only channel; there is no stdout
  protocol.

This is the whole contract: mutate the writable directory, exit `0`.

---

## 11. Edge cases

| Case | Behavior |
|------|----------|
| Added file | Absent in `left/`; present in `right/`. Deselect = delete from writable side. |
| Deleted file | Present in `left/`; absent in `right/`. Deselect = copy back from `left/`. |
| Empty file | Distinguish empty content from "missing"; missing means delete. |
| Executable bit | Preserved from the writable file's mode. |
| Symlink | Materialized as a symlink; keep it a symlink. |
| Binary / non-UTF-8 | Materialized as raw bytes; do not lossy-decode when writing back. |
| Unchanged file | Not checked out (not in sparse set); leave no stray copy. |
| `JJ-INSTRUCTIONS` | Skip; deleted before snapshot regardless. |
| No `left` dir entries | Pure-addition diff — `left/` exists but is empty. |

---

## 12. Minimal editor checklist

1. Read two (or three) path operands from argv; error if missing.
2. Decide writable side: `output` if provided, else `right`. Never write `left`.
3. Walk `left/` and the writable side; skip `JJ-INSTRUCTIONS` and `*_state/`.
4. Classify each path Added / Deleted / Modified / Unchanged.
5. Build hunks and let the user select.
6. Rewrite each file in the writable tree so it contains exactly the selected
   hunks (revert unselected hunks toward `left`).
7. Exit `0` on confirm, non-zero on abort.

---

## 13. Worked example

```toml
[ui]
diff-editor = ["diffpicker", "$left", "$right"]
```

```sh
jj split          # opens diffpicker on the materialized diff
```

Result: the hunks left in `right/` when diffpicker exits become the first
(selected) commit; everything reverted becomes the second (remaining) commit.

3-pane variant (right shown read-only, edits go to `output/`):

```toml
[ui]
diff-editor = ["mytool", "$left", "$output", "$right"]
```

---

## Maintenance note

If jj changes the temp layout, the `$`-substitution names, or the default
`edit-args`, update §5–§8. The read-back rule in §8 is the load-bearing part:
everything else can be re-derived from `jj help split`, `jj help -k config`
("Editing diffs" and "Editor"), and the two jj source files named at the top.
