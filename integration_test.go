//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationSplitWithRealJJ drives the actual binary through `jj split`
// to prove the read-back contract holds against a real jj, not just the unit
// tests' assumptions about it.
func TestIntegrationSplitWithRealJJ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	jjPath, err := exec.LookPath("jj")
	if err != nil {
		t.Skip("jj not installed")
	}
	toolPath := buildTool(t)

	t.Run("deleted line moves to the new commit", func(t *testing.T) {
		editor := writeShellScript(t,
			"grep -v '^+delta$' \"$1/f.txt\" > \"$1/f.txt.tmp\"\n"+
				"mv \"$1/f.txt.tmp\" \"$1/f.txt\"\n",
		)
		f := newJJFixture(t, jjPath, toolPath, editor)
		f.write("f.txt", "alpha\nbeta\ngamma\n")
		f.run("describe", "-m", "base")
		f.run("new")
		f.write("f.txt", "alpha\nBETA\ngamma\ndelta\n")
		f.run("split")

		original := f.run("diff", "-r", "@-", "--git")
		newCommit := f.run("diff", "-r", "@", "--git")
		if !strings.Contains(newCommit, "+delta") {
			t.Fatalf("new commit should contain +delta:\n%s", newCommit)
		}
		if strings.Contains(original, "delta") {
			t.Fatalf("deleted line should not stay in original:\n%s", original)
		}
		if !strings.Contains(original, "+BETA") {
			t.Fatalf("original should keep the beta change:\n%s", original)
		}
	})

	t.Run("whole-file delete moves the diff to the new commit", func(t *testing.T) {
		editor := writeShellScript(t, "rm -f \"$1/mod.txt\"\n")
		f := newJJFixture(t, jjPath, toolPath, editor)
		f.write("mod.txt", "alpha\nbeta\n")
		f.run("describe", "-m", "base")
		f.run("new")
		f.write("mod.txt", "alpha\nBETA\ndelta\n")
		f.run("split")

		original := f.run("diff", "-r", "@-", "--git")
		newCommit := f.run("diff", "-r", "@", "--git")
		if strings.Contains(original, "BETA") {
			t.Fatalf("original should be unchanged for the reverted file:\n%s", original)
		}
		if !strings.Contains(newCommit, "+BETA") || !strings.Contains(newCommit, "+delta") {
			t.Fatalf("new commit should hold the whole diff:\n%s", newCommit)
		}
	})

	t.Run("deleted removal line defers the removal to the new commit", func(t *testing.T) {
		editor := writeShellScript(t,
			"grep -v '^-beta$' \"$1/f.txt\" > \"$1/f.txt.tmp\"\n"+
				"mv \"$1/f.txt.tmp\" \"$1/f.txt\"\n",
		)
		f := newJJFixture(t, jjPath, toolPath, editor)
		f.write("f.txt", "alpha\nbeta\ngamma\n")
		f.run("describe", "-m", "base")
		f.run("new")
		f.write("f.txt", "alpha\nBETA\ngamma\n")
		f.run("split")

		original := f.run("diff", "-r", "@-", "--git")
		newCommit := f.run("diff", "-r", "@", "--git")
		if !strings.Contains(original, "+BETA") {
			t.Fatalf("first commit should keep the addition:\n%s", original)
		}
		if !strings.Contains(newCommit, "-beta") {
			t.Fatalf("new commit should contain the removal:\n%s", newCommit)
		}
	})
}

func buildTool(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "jj-split-cli")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

type jjFixture struct {
	t       *testing.T
	jjPath  string
	repoDir string
	env     []string
}

func newJJFixture(t *testing.T, jjPath, toolPath, editor string) *jjFixture {
	t.Helper()
	// Keep the config outside the repo so it does not show up as a working-copy
	// change and leak into the split.
	configPath := filepath.Join(t.TempDir(), "config.toml")
	config := fmt.Sprintf(`[user]
name = "test"
email = "test@example.com"
[ui]
diff-editor = [%q, "$left", "$right"]
editor = "true"
`, toolPath)
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &jjFixture{
		t:       t,
		jjPath:  jjPath,
		repoDir: t.TempDir(),
		env: append(os.Environ(),
			"JJ_CONFIG="+configPath,
			"JJ_EDITOR=true",
			"JJ_SPLIT_EDITOR="+editor,
		),
	}
	f.run("git", "init")
	// Guard against any future config leaking jj into the surrounding repo.
	want, _ := filepath.EvalSymlinks(f.repoDir)
	got, _ := filepath.EvalSymlinks(strings.TrimSpace(f.run("root")))
	if got != want {
		t.Fatalf("jj is not isolated to the fixture repo: root=%q want=%q", got, want)
	}
	return f
}

func (f *jjFixture) run(args ...string) string {
	f.t.Helper()
	cmd := exec.Command(f.jjPath, append([]string{"--no-pager", "--color=never"}, args...)...)
	cmd.Dir = f.repoDir
	cmd.Env = f.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("jj %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func (f *jjFixture) write(rel, content string) {
	f.t.Helper()
	path := filepath.Join(f.repoDir, filepath.FromSlash(rel))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}
