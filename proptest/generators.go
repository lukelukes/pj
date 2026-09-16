package proptest

import (
	"fmt"
	"pj/internal/catalog"
	"slices"

	"pgregory.net/rapid"
)

var (
	pathSegmentGen = rapid.StringMatching(`[a-z]{5,10}`)
	iterDirGen     = rapid.StringMatching(`[a-z]{8}`)
	subdirGen      = rapid.StringMatching(`[a-z]{6}`)
	shortQueryGen  = rapid.StringMatching(`[a-z]{1,5}`)
	queryGen       = rapid.StringMatching(`[a-z]{1,10}`)
)

func validNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]{0,30}`)
}

func malformedYAMLGen() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just("{{{{"),
		rapid.Just("}}}}"),
		rapid.Just("- - - -"),
		rapid.Just(":::"),
		rapid.Just("[\n["),
		rapid.Just("key: [unclosed"),
		rapid.Just("key: {unclosed"),
		rapid.Just("- item\n  bad indent"),
		rapid.Just("\t\ttabs: everywhere"),
		rapid.Just("version: \"unmatched quote"),
		rapid.Just("projects:\n  - id: missing\n  name: value"),
		rapid.StringMatching(`[^a-zA-Z0-9\s]{10,50}`),
		rapid.Custom(func(t *rapid.T) string {
			size := rapid.IntRange(10, 100).Draw(t, "size")
			bytes := make([]byte, size)
			for i := range bytes {
				bytes[i] = byte(rapid.IntRange(0, 255).Draw(t, "byte"))
			}
			return string(bytes)
		}),
	)
}

func missingFieldsGen() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just("version: 1\nprojects:\n  - name: test\n"),
		rapid.Just("version: 1\nprojects:\n  - id: abc123\n"),
		rapid.Just("version: 1\nprojects:\n  - path: /some/path\n"),
		rapid.Just("version: 1\nprojects:\n  - {}\n"),
		rapid.Just("version: 1\nprojects:\n  - name: test\n    path: /path\n"),
		rapid.Just("projects:\n  - name: test\n    id: abc\n    path: /path\n"),
	)
}

func extraFieldsGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		extraField := rapid.SampledFrom([]string{
			"unknown_field",
			"extra",
			"foo",
			"bar_baz",
			"randomField123",
		}).Draw(t, "fieldName")
		extraValue := rapid.SampledFrom([]string{
			"string_value",
			"123",
			"true",
			"[1, 2, 3]",
			"{nested: value}",
		}).Draw(t, "fieldValue")

		return fmt.Sprintf(`version: 1
%s: %s
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    %s: %s
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`, extraField, extraValue, extraField, extraValue)
	})
}

func invalidTypesGen() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just(`version: "not_a_number"
projects: []
`),
		rapid.Just(`version: 1
projects:
  - id: 12345
    name: test-project
    path: /tmp/test
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: [not, a, string]
    path: /tmp/test
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    added_at: "not-a-date"
`),
	)
}

var (
	fieldNameGen     = rapid.SampledFrom(catalog.FilterFields())
	opGen            = rapid.SampledFrom([]string{"=", "!=", "~", "!~"})
	optionalFieldGen = rapid.OneOf(rapid.Just(""), rapid.StringMatching(`[a-m]{1,10}`))
	termValueGen     = rapid.OneOf(
		rapid.Just(""), rapid.StringMatching(`[a-m0-9]{1,8}`),
		rapid.SampledFrom([]string{"prefix*", "*suffix", "a?c", "[ab]*", "*"}),
		rapid.SampledFrom([]string{"a=b", "a~b", "a!b", "a:b", "a=b~!", "a,b"}),
		rapid.SampledFrom([]string{"AbC", "Éclair", "你好", " leading", "trailing "}),
	)
	termGen = rapid.Custom(func(t *rapid.T) catalog.Term {
		return catalog.Term{Field: fieldNameGen.Draw(t, "field"), Op: opGen.Draw(t, "op"), Value: termValueGen.Draw(t, "value")}
	})
	projectGen = rapid.Custom(func(t *rapid.T) catalog.Project {
		name := rapid.StringMatching(`[a-m]{1,10}`).Draw(t, "name")
		segment := rapid.StringMatching(`[a-m]{1,10}`).Draw(t, "segment")
		return projectMetadata(t, &projectGenConfig{}, catalog.NewProject(name, "/p/"+segment))
	})
	projectsGen = rapid.Custom(func(t *rapid.T) []catalog.Project {
		projects := rapid.SliceOfN(projectGen, 0, 15).Draw(t, "projects")
		for i := range projects {
			projects[i].Path = fmt.Sprintf("/p/%d-%s", i, projects[i].Name)
		}
		return projects
	})
)

// ProjectsGen generates small in-memory project catalogs with unique paths.
func ProjectsGen() *rapid.Generator[[]catalog.Project] { return projectsGen }

// TermGen generates valid terms including empty, glob and punctuation values.
func TermGen() *rapid.Generator[catalog.Term] { return termGen }

// Permute returns a shuffled copy using shrinkable Fisher–Yates draws.
func Permute[T any](t *rapid.T, xs []T) []T {
	result := slices.Clone(xs)
	for i := len(result) - 1; i > 0; i-- {
		j := rapid.IntRange(0, i).Draw(t, "swap")
		result[i], result[j] = result[j], result[i]
	}
	return result
}

type generatedFilter struct {
	filter   catalog.Filter
	root     string
	depth    int
	children int
}

func filterTreeGen(depth int) *rapid.Generator[generatedFilter] {
	return rapid.Custom(func(t *rapid.T) generatedFilter {
		kind := "term"
		if depth > 0 {
			kind = rapid.SampledFrom([]string{"term", "and", "or", "not"}).Draw(t, "kind")
		}
		if kind == "term" {
			return generatedFilter{filter: termGen.Draw(t, "term").Filter(), root: kind}
		}
		if kind == "not" {
			child := filterTreeGen(depth-1).Draw(t, "child")
			return generatedFilter{filter: catalog.Not(child.filter), root: kind, depth: child.depth + 1, children: 1}
		}
		children := rapid.SliceOfN(filterTreeGen(depth-1), 0, 3).Draw(t, "children")
		filters := make([]catalog.Filter, len(children))
		actualDepth := 0
		for i, child := range children {
			filters[i] = child.filter
			actualDepth = max(actualDepth, child.depth+1)
		}
		f := catalog.And(filters...)
		if kind == "or" {
			f = catalog.Or(filters...)
		}
		return generatedFilter{filter: f, root: kind, depth: actualDepth, children: len(children)}
	})
}

func filterGen(depth int) *rapid.Generator[catalog.Filter] {
	return rapid.Map(filterTreeGen(depth), func(f generatedFilter) catalog.Filter { return f.filter })
}
