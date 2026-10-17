package main

// fileContent is a file split into comparison text and raw (terminator
// preserving) lines.
type fileContent struct {
	texts []string
	raws  []string
}

func parseFile(content []byte) fileContent {
	texts, raws := parseLines(content)
	return fileContent{texts: texts, raws: raws}
}

// lineOp is one line of a diff between two versions of a file.
type lineOp struct {
	kind byte   // ' ' common, '-' left-only, '+' right-only
	text string // comparison text
	raw  string // original bytes including terminator
}

// traceMemoryCap bounds Myers' trace allocations. Past this, lineDiff falls
// back to a prefix/suffix block diff so a pathologically large rewrite cannot
// exhaust memory. Myers handles the common "big file, small change" case with a
// tiny trace, so the cap is only hit by huge rewrites.
const traceMemoryCap = 8_000_000

// lineDiff returns the per-line operations that transform left into right using
// Myers' O(ND) algorithm with deletion preferred over insertion on ties.
func lineDiff(left, right fileContent) []lineOp {
	n, m := len(left.texts), len(right.texts)
	furthest := n + m
	if furthest == 0 {
		return nil
	}
	rowWidth := 2*furthest + 1
	v := make([]int, rowWidth)
	var trace [][]int
	end := -1
	for d := 0; d <= furthest; d++ {
		if (d+1)*rowWidth > traceMemoryCap {
			return blockDiff(left, right)
		}
		trace = append(trace, append([]int(nil), v...))
		for k := -d; k <= d; k += 2 {
			idx := k + furthest
			var x int
			if k == -d || (k != d && v[idx-1] < v[idx+1]) {
				x = v[idx+1]
			} else {
				x = v[idx-1] + 1
			}
			y := x - k
			for x < n && y < m && left.texts[x] == right.texts[y] {
				x++
				y++
			}
			v[idx] = x
			if x >= n && y >= m {
				end = d
				break
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return blockDiff(left, right)
	}

	var ops []lineOp
	x, y := n, m
	for d := end; d > 0; d-- {
		tv := trace[d]
		k := x - y
		idx := k + furthest
		var prevK int
		if k == -d || (k != d && tv[idx-1] < tv[idx+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := tv[prevK+furthest]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
			ops = append(ops, lineOp{kind: ' ', text: left.texts[x], raw: left.raws[x]})
		}
		if prevK == k+1 {
			y--
			ops = append(ops, lineOp{kind: '+', text: right.texts[y], raw: right.raws[y]})
		} else {
			x--
			ops = append(ops, lineOp{kind: '-', text: left.texts[x], raw: left.raws[x]})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		ops = append(ops, lineOp{kind: ' ', text: left.texts[x], raw: left.raws[x]})
	}
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
	return ops
}

// blockDiff is the bounded-memory fallback: a common prefix and suffix are kept
// as context and everything between is a replace block.
func blockDiff(left, right fileContent) []lineOp {
	prefix := 0
	for prefix < len(left.texts) && prefix < len(right.texts) && left.texts[prefix] == right.texts[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(left.texts)-prefix && suffix < len(right.texts)-prefix &&
		left.texts[len(left.texts)-1-suffix] == right.texts[len(right.texts)-1-suffix] {
		suffix++
	}
	var ops []lineOp
	for i := 0; i < prefix; i++ {
		ops = append(ops, lineOp{kind: ' ', text: left.texts[i], raw: left.raws[i]})
	}
	for i := prefix; i < len(left.texts)-suffix; i++ {
		ops = append(ops, lineOp{kind: '-', text: left.texts[i], raw: left.raws[i]})
	}
	for i := prefix; i < len(right.texts)-suffix; i++ {
		ops = append(ops, lineOp{kind: '+', text: right.texts[i], raw: right.raws[i]})
	}
	for i := len(left.texts) - suffix; i < len(left.texts); i++ {
		ops = append(ops, lineOp{kind: ' ', text: left.texts[i], raw: left.raws[i]})
	}
	return ops
}

// hunk is a half-open range of ops to render.
type hunk struct {
	start int
	end   int
}

// groupHunks merges changed ops that are within 2*context equal lines of each
// other and pads each group with up to context equal lines on both sides.
func groupHunks(ops []lineOp, context int) []hunk {
	var changes []int
	for i, op := range ops {
		if op.kind != ' ' {
			changes = append(changes, i)
		}
	}
	if len(changes) == 0 {
		return nil
	}
	var hunks []hunk
	start, prev := changes[0], changes[0]
	for _, c := range changes[1:] {
		if c-prev-1 > 2*context {
			hunks = append(hunks, hunk{max(0, start-context), min(len(ops), prev+context+1)})
			start = c
		}
		prev = c
	}
	hunks = append(hunks, hunk{max(0, start-context), min(len(ops), prev+context+1)})
	return hunks
}
