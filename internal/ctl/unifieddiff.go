package ctl

import (
	"fmt"
	"strings"
)

const (
	diffContextLines = 3
	// diffMaxLines bounds the table below: a compose file is a few hundred lines, so more is not a compose file
	diffMaxLines = 4000
)

// unifiedDiff renders the change from a to b in unified diff format, or "" when they are equal or too large to compare.
func unifiedDiff(nameA, nameB, a, b string) string {
	if a == b {
		return ""
	}
	x, y := splitLines(a), splitLines(b)
	if len(x) > diffMaxLines || len(y) > diffMaxLines {
		return ""
	}
	// longest common subsequence table, filled from the end so the walk below reads forward
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type op struct {
		kind byte // ' ', '-', '+'
		text string
	}
	var ops []op
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] == y[j]:
			ops = append(ops, op{' ', x[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, op{'-', x[i]})
			i++
		default:
			ops = append(ops, op{'+', y[j]})
			j++
		}
	}
	for ; i < len(x); i++ {
		ops = append(ops, op{'-', x[i]})
	}
	for ; j < len(y); j++ {
		ops = append(ops, op{'+', y[j]})
	}

	// group changed ops with their context into hunks
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", nameA, nameB)
	aLine, bLine := 1, 1
	for k := 0; k < len(ops); {
		if ops[k].kind == ' ' {
			aLine++
			bLine++
			k++
			continue
		}
		start := k
		for start > 0 && k-start < diffContextLines && ops[start-1].kind == ' ' {
			start--
		}
		end := k
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := 0
			for run < len(ops)-end && ops[end+run].kind == ' ' {
				run++
			}
			if end+run >= len(ops) || run > 2*diffContextLines {
				end += min(run, diffContextLines)
				break
			}
			end += run
		}
		aStart, bStart := aLine-(k-start), bLine-(k-start)
		aCount, bCount := 0, 0
		var body strings.Builder
		for _, o := range ops[start:end] {
			body.WriteString(string(o.kind) + o.text + "\n")
			if o.kind != '+' {
				aCount++
			}
			if o.kind != '-' {
				bCount++
			}
		}
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n%s", aStart, aCount, bStart, bCount, body.String())
		for _, o := range ops[k:end] {
			if o.kind != '+' {
				aLine++
			}
			if o.kind != '-' {
				bLine++
			}
		}
		k = end
	}
	return out.String()
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
