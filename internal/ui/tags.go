package ui

import "strings"

func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return "#" + strings.Join(tags, " #")
}
