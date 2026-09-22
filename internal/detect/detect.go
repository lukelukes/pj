// Package detect infers project tags from marker files in a directory.
package detect

import (
	"io/fs"
	"slices"
)

type Rule struct {
	Tag     string
	Markers []string
}

var rules = []Rule{
	{"lang:go", []string{"go.mod"}},
	{"lang:rust", []string{"Cargo.toml"}},
	{"lang:js", []string{"package.json"}},
	{"lang:ts", []string{"tsconfig.json"}},
	{"lang:python", []string{"pyproject.toml", "setup.py", "requirements.txt"}},
	{"lang:ruby", []string{"Gemfile"}},
	{"lang:java", []string{"pom.xml", "build.gradle", "build.gradle.kts"}},
	{"lang:elixir", []string{"mix.exs"}},
	{"lang:zig", []string{"build.zig"}},
}

func Detect(fsys fs.FS) []string {
	var tags []string
	for _, rule := range rules {
		for _, marker := range rule.Markers {
			if info, err := fs.Stat(fsys, marker); err == nil && !info.IsDir() {
				tags = append(tags, rule.Tag)
				break
			}
		}
	}
	slices.Sort(tags)
	return slices.Compact(tags)
}
