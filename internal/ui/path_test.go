package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		home string
		want string
	}{
		{"replaces home prefix with tilde", "/home/luke/dev/pj", "/home/luke", "~/dev/pj"},
		{"home itself collapses to tilde", "/home/luke", "/home/luke", "~"},
		{"unrelated path is untouched", "/opt/tools", "/home/luke", "/opt/tools"},
		{"sibling with shared prefix is untouched", "/home/luke2/dev", "/home/luke", "/home/luke2/dev"},
		{"empty home is a no-op", "/home/luke/dev", "", "/home/luke/dev"},
		{"root home is a no-op", "/etc/hosts", "/", "/etc/hosts"},
		{"trailing slash inside home", "/home/luke/dev/", "/home/luke", "~/dev/"},
		{"relative path is untouched", "dev/pj", "/home/luke", "dev/pj"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DisplayPath(tc.path, tc.home))
		})
	}
}

func TestDisplayPathNeverShortensAcrossASegment(t *testing.T) {
	const home = "/home/luke"
	for _, path := range []string{"/home/lukeXX", "/home/luke-old/x", "/home/luke.bak"} {
		assert.Equal(t, path, DisplayPath(path, home),
			"%s does not live under %s and must not be abbreviated", path, home)
	}
}
