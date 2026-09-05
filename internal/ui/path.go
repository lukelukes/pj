package ui

import "strings"

func DisplayPath(path, home string) string {
	if home == "" || home == "/" || !strings.HasPrefix(path, home) {
		return path
	}
	rest := path[len(home):]
	switch {
	case rest == "":
		return "~"
	case rest[0] == '/':
		return "~" + rest
	default:
		return path
	}
}
