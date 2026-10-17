# jj-split-cli

Split jj commits by deleting lines in your real code editor.

## Why

`jj split` normally opens a terminal TUI that only lets you pick whole hunks.
Picking a single line — or three scattered lines — means fighting the context.

`jj-split-cli` opens each changed file in your normal editor, rendered as a diff:

```diff
@@ -1,3 +1,3 @@
 func main() {
-    fmt.Println("old")
+    fmt.Println("new")
     return
 }
```

Delete a `-` or `+` line to move that change into the new commit. Close the
window, done.

## Features

- **Your editor, your keybindings.** VS Code by default; any editor via
  `JJ_SPLIT_EDITOR`.
- **A diff, not a raw file.** Every changed file opens as a hunked diff with
  ` `/`-`/`+` prefixes, so changes jump out immediately.
- **Delete a change to move it.** Deleting a `+` line moves that addition to the
  new commit; deleting a `-` line moves that removal to the new commit (the line
  comes back in the first commit).
- **Whole-file deletes move the file's diff.** Delete the rendered file and its
  entire diff moves to the new commit.
- **Predictable by design.** Only `-`/`+` line deletions are honored. Added,
  edited, or reordered lines are rejected so a stray keystroke can't reshuffle
  the split.
- **Single Go binary.**

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
2. VS Code opens a new window with each changed file rendered as a diff.
3. Delete the `-`/`+` lines for the changes you want in the new commit.
4. Close the window.
5. jj creates the new commit containing exactly the changes you deleted.

```text
$ jj split
# ... VS Code opens on app.go ...
@@ -1,10 +1,13 @@
 package main

+import "fmt"
+
 func main() {
 	greet("world")
+	fmt.Println("done")
 	return
 }

 func greet(name string) {
-	println("hello " + name)
+	println("hi " + name)
 }
```

Result, by design of `jj split`:

```text
original revision (reused)  →  the changes you kept
new commit (new change id)  →  the changes you deleted
```

If you touch anything besides deleting `-`/`+` lines, the tool refuses to run
the split and tells you which file caused it.

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `JJ_SPLIT_EDITOR` | `code --wait --new-window` | Command used to open the files. The diff directory is appended as the last argument. |

Flags go in the `diff-editor` list after the binary:

| Flag | Default | Description |
|------|---------|-------------|
| `--full` | off | Render the whole file instead of only the changed hunks. |
| `--context N` | `3` | Lines of context around each hunk (only affects the default hunk mode). |

```toml
# Show full files so you can delete anywhere:
[ui]
diff-editor = ["jj-split-cli", "--full", "$left", "$right"]

# Wider hunks, 8 lines of context:
[ui]
diff-editor = ["jj-split-cli", "--context", "8", "$left", "$right"]
```

Editor examples:

```sh
JJ_SPLIT_EDITOR="subl -n -w"           # Sublime Text
JJ_SPLIT_EDITOR="code --reuse-window"  # reuse an existing VS Code window
JJ_SPLIT_EDITOR="nvim"                 # terminal editor
```

## Development

```sh
go test ./...            # unit + real-jj integration (skips if jj is absent)
go test -short ./...     # unit tests only
go build -o jj-split-cli .
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the design.
