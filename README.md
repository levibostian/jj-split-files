# jj-split-cli

Split jj commits in your real code editor.

## Why

`jj split` normally opens a terminal TUI that only lets you pick whole hunks.
If you want to move a single line — or three scattered lines — to the new
commit, you're stuck counting context lines.

`jj-split-cli` opens the changed files in your normal editor instead. Delete
the lines you want the new commit to contain, close the window, done.

- **Before:** pick hunks, guess which lines are in each hunk, fight the TUI.
- **After:** delete exactly the lines you want in the new commit.

## Features

- **Your editor, your keybindings.** Opens the changed files in VS Code by
  default; works with any editor via `JJ_SPLIT_EDITOR`.
- **Delete lines to select them.** Deleted lines become the content of the new
  commit `jj split` creates.
- **Predictable by design.** Only deletions are honored. Added or reordered
  lines are rejected so a stray keystroke can't silently reshuffle the split.
- **Whole-file deletes move the file's diff.** Delete a modified file and its
  changes move to the new commit; delete an added file and the whole addition
  moves.
- **No dependencies.** Single Go binary.

## Prerequisites

- [jj](https://jj-vcs.github.io/jj/) 0.46 or newer.
- [Go](https://go.dev/dl/) 1.24+ to build.
- VS Code's `code` CLI on your `PATH` (or set `JJ_SPLIT_EDITOR`).

## Quick start

```sh
go build -o ~/.local/bin/jj-split-cli .
```

```toml
# ~/.config/jj/config.toml
[ui]
diff-editor = ["jj-split-cli", "$left", "$right"]
```

```sh
jj split
```

## Usage

1. Run `jj split` on the revision you want to split.
2. VS Code opens a new window with the changed files.
3. Delete the lines you want in the new commit. Leave everything else alone.
4. Close the window.
5. jj creates the second commit containing exactly the lines you deleted.

```text
$ jj split
# ... VS Code opens, you delete the `debug-log` lines, close the window ...
Selected changes : pqrstuvw 1a2b3c4d
Remaining changes: zyxwvuts 9f8e7d6c
Working copy  (@) now at: zyxwvuts 9f8e7d6c
```

The result, by design of `jj split`:

```text
original revision (reused)  →  everything you kept
new commit (new change id)  →  exactly the lines you deleted
```

If you touch anything besides deleting lines, the tool refuses to run the
split and tells you which file caused it.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JJ_SPLIT_EDITOR` | `code --wait --new-window` | Command used to open the files. The diff directory is appended as the last argument. |

Examples:

```sh
JJ_SPLIT_EDITOR="subl -n -w"       # Sublime Text
JJ_SPLIT_EDITOR="code --reuse-window"  # reuse an existing VS Code window
JJ_SPLIT_EDITOR="nvim"             # terminal editor
```

## Development

```sh
go test ./...
go build -o jj-split-cli .
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the design.
