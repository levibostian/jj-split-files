package main

import (
	"bytes"
	"fmt"
	"strings"
)

// renderedLine is one line written into the editor. compare is the exact text
// the user sees (marker plus content), used to detect which lines were deleted;
// opIndex is the op it came from, or -1 for hunk headers that carry no change.
type renderedLine struct {
	compare string
	opIndex int
}

// filePlan is a diff rendered into the editor plus enough context to rebuild the
// first-commit file from whatever the user leaves behind.
type filePlan struct {
	rel           string
	ops           []lineOp
	renderedLines []renderedLine
	rendered      []byte
}

func isBinary(content []byte) bool {
	return bytes.IndexByte(content, 0) >= 0
}

// buildPlan renders the diff between left and right for the editor. It returns
// ok=false when there is nothing to select (no changed lines).
func buildPlan(rel string, left, right []byte, full bool, context int) (filePlan, bool) {
	ops := lineDiff(parseFile(left), parseFile(right))
	if !hasChange(ops) {
		return filePlan{}, false
	}

	var lines []renderedLine
	if full {
		for i, op := range ops {
			lines = append(lines, renderedLine{compare: string(op.kind) + op.text, opIndex: i})
		}
	} else {
		oldPrefix, newPrefix := linePrefixCounts(ops)
		for _, h := range groupHunks(ops, context) {
			oldCount, newCount := 0, 0
			for _, op := range ops[h.start:h.end] {
				if op.kind == ' ' || op.kind == '-' {
					oldCount++
				}
				if op.kind == ' ' || op.kind == '+' {
					newCount++
				}
			}
			header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldPrefix[h.start]+1, oldCount, newPrefix[h.start]+1, newCount)
			lines = append(lines, renderedLine{compare: header, opIndex: -1})
			for i := h.start; i < h.end; i++ {
				lines = append(lines, renderedLine{compare: string(ops[i].kind) + ops[i].text, opIndex: i})
			}
		}
	}

	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(line.compare)
		sb.WriteByte('\n')
	}
	return filePlan{rel: rel, ops: ops, renderedLines: lines, rendered: []byte(sb.String())}, true
}

func hasChange(ops []lineOp) bool {
	for _, op := range ops {
		if op.kind != ' ' {
			return true
		}
	}
	return false
}

// linePrefixCounts returns, for each op index, how many left and right lines
// come before it. Used to number hunks.
func linePrefixCounts(ops []lineOp) (oldPrefix, newPrefix []int) {
	oldPrefix = make([]int, len(ops)+1)
	newPrefix = make([]int, len(ops)+1)
	for i, op := range ops {
		oldPrefix[i+1] = oldPrefix[i]
		newPrefix[i+1] = newPrefix[i]
		if op.kind == ' ' || op.kind == '-' {
			oldPrefix[i+1]++
		}
		if op.kind == ' ' || op.kind == '+' {
			newPrefix[i+1]++
		}
	}
	return oldPrefix, newPrefix
}

// reconstruct builds the first-commit file. A change line the user deleted moves
// to the new commit, so a deleted '+' line is dropped and a deleted '-' line is
// restored; common lines are always kept.
func reconstruct(plan filePlan, deleted map[int]bool) []byte {
	var sb strings.Builder
	for i, op := range plan.ops {
		switch op.kind {
		case ' ':
			sb.WriteString(op.raw)
		case '-':
			if deleted[i] {
				sb.WriteString(op.raw)
			}
		case '+':
			if !deleted[i] {
				sb.WriteString(op.raw)
			}
		}
	}
	return []byte(sb.String())
}
