package main

import "strings"

// builtinEditors maps convenience keywords to an editor command. The diff
// directory is appended as the final argument, so GUI editors must wait for the
// window to close and terminal editors block naturally.
var builtinEditors = map[string]string{
	"vscode":   "code --wait --new-window",
	"code":     "code --wait --new-window",
	"sublime":  "subl -n -w",
	"subl":     "subl -n -w",
	"zed":      "zed --wait --new",
	"textmate": "mate -w",
	"mate":     "mate -w",
	"vim":      "vim",
	"nvim":     "nvim",
	"neovim":   "nvim",
	"helix":    "hx",
	"hx":       "hx",
	"emacs":    "emacs -nw",
	"micro":    "micro",
	"nano":     "nano",
}

// resolveEditor turns an editor spec into a command line. A spec matching a
// built-in keyword expands to that editor's command; anything else is treated as
// a command line supplied by the user.
func resolveEditor(spec string) string {
	if command, ok := builtinEditors[strings.ToLower(strings.TrimSpace(spec))]; ok {
		return command
	}
	return spec
}
