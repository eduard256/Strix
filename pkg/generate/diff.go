package generate

import "strings"

func fullDiff(config string) []DiffLine {
	lines := strings.Split(config, "\n")
	diff := make([]DiffLine, len(lines))
	for i, line := range lines {
		diff[i] = DiffLine{Line: i + 1, Text: line, Type: "added"}
	}
	return diff
}

func diffWithContext(lines []string, added map[int]bool, ctx int) []DiffLine {
	visible := make(map[int]bool)
	for idx := range added {
		for c := -ctx; c <= ctx; c++ {
			if j := idx + c; j >= 0 && j < len(lines) {
				visible[j] = true
			}
		}
	}

	var diff []DiffLine
	for i, line := range lines {
		if !visible[i] {
			continue
		}
		t := "context"
		if added[i] {
			t = "added"
		}
		diff = append(diff, DiffLine{Line: i + 1, Text: line, Type: t})
	}
	return diff
}
