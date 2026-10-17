package main

import (
	"math/rand"
	"slices"
	"testing"
)

func fc(lines []string) fileContent {
	raws := make([]string, len(lines))
	for i, line := range lines {
		raws[i] = line + "\n"
	}
	return fileContent{texts: lines, raws: raws}
}

// lcsLength is a brute-force reference used to check the diff is minimal.
func lcsLength(a, b []string) int {
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	return dp[0][0]
}

func applyOps(ops []lineOp) (left, right []string) {
	for _, op := range ops {
		switch op.kind {
		case ' ':
			left = append(left, op.text)
			right = append(right, op.text)
		case '-':
			left = append(left, op.text)
		case '+':
			right = append(right, op.text)
		}
	}
	return left, right
}

func checkDiff(t *testing.T, a, b []string) {
	t.Helper()
	ops := lineDiff(fc(a), fc(b))
	left, right := applyOps(ops)
	if !slices.Equal(left, a) || !slices.Equal(right, b) {
		t.Fatalf("ops do not transform %v -> %v (got %v -> %v)", a, b, left, right)
	}
	edits := 0
	for _, op := range ops {
		if op.kind != ' ' {
			edits++
		}
	}
	if want := len(a) + len(b) - 2*lcsLength(a, b); edits != want {
		t.Fatalf("edits = %d, want minimal %d for %v -> %v", edits, want, a, b)
	}
}

func TestLineDiffCases(t *testing.T) {
	cases := [][2][]string{
		{{"a", "b", "c"}, {"a", "B", "c"}},
		{{"a", "b", "c"}, {"a", "c"}},
		{{"a", "b"}, {"a", "b", "c"}},
		{{}, {"x", "y"}},
		{{"x", "y"}, {}},
		{{"a", "b", "c"}, {"c", "b", "a"}},
		{{"same"}, {"same"}},
		{{}, {}},
		{{"a", "a", "a"}, {"a", "a"}},
	}
	for _, c := range cases {
		checkDiff(t, c[0], c[1])
	}
}

func TestLineDiffRandomized(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	alphabet := []string{"a", "b", "c", "d"}
	randomSeq := func(n int) []string {
		s := make([]string, n)
		for i := range s {
			s[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return s
	}
	for i := 0; i < 1000; i++ {
		checkDiff(t, randomSeq(rng.Intn(14)), randomSeq(rng.Intn(14)))
	}
}

func TestBlockDiff(t *testing.T) {
	cases := [][2][]string{
		{{"a", "b", "c"}, {"a", "x", "y", "c"}},
		{{}, {"x"}},
		{{"x"}, {}},
		{{"a", "b"}, {"a", "b"}},
		{{"a", "b", "c", "d"}, {"a", "c", "d"}},
	}
	for _, c := range cases {
		ops := blockDiff(fc(c[0]), fc(c[1]))
		left, right := applyOps(ops)
		if !slices.Equal(left, c[0]) || !slices.Equal(right, c[1]) {
			t.Fatalf("blockDiff does not transform %v -> %v", c[0], c[1])
		}
	}
}

func TestGroupHunks(t *testing.T) {
	// Changes at indices 1 and 3, one equal line between them.
	ops := []lineOp{
		{kind: ' '}, {kind: '-'}, {kind: ' '}, {kind: '+'}, {kind: ' '},
	}
	hunks := groupHunks(ops, 1)
	if len(hunks) != 1 || hunks[0].start != 0 || hunks[0].end != 5 {
		t.Fatalf("context 1: got %+v, want one hunk {0 5}", hunks)
	}
	hunks = groupHunks(ops, 0)
	if len(hunks) != 2 {
		t.Fatalf("context 0: got %d hunks, want 2: %+v", len(hunks), hunks)
	}
	if hunks[0].start != 1 || hunks[0].end != 2 || hunks[1].start != 3 || hunks[1].end != 4 {
		t.Fatalf("context 0: got %+v, want [{1 2} {3 4}]", hunks)
	}
}
