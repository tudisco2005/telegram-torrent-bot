package utils

import "strings"

// ProgressBar returns a fixed-width text progress bar using '#' for filled and '-' for remaining.
func ProgressBar(progress float64, width int) string {
	if width <= 0 {
		width = 15
	}

	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress*float64(width) + 0.5)
	if filled > width {
		filled = width
	}

	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}
