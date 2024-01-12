package helpers

import "strings"

func renderScrollBar(rows, height, position int) string {
	if rows <= height {
		return strings.Repeat(" \n", rows)
	}

	pos := int(float64(position) * float64(height) / float64(rows-height))
	if height == pos {
		return strings.Repeat("\n", height-1) + "█"
	}
	return strings.Repeat("\n", pos) + "█" + strings.Repeat("\n", height-pos-1)
}
