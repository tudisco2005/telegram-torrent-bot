package utils

import (
	"regexp"
	"strings"
)

// MarkdownReplacer replaces markdown special chars to prevent formatting issues
var MarkdownReplacer = strings.NewReplacer("*", "•",
	"[", "(",
	"]", ")",
	"_", "-",
	"`", "'")

// TrackerRegex extracts tracker hostnames from URLs
var TrackerRegex = regexp.MustCompile(`(?:https?|udp)://([^:/]*)`)

// EscapeMarkdown escapes markdown special characters in text
func EscapeMarkdown(text string) string {
	return MarkdownReplacer.Replace(text)
}

func EscapeFileMD(name string) string {
	// escape markdown special characters in filename
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(name)
}
