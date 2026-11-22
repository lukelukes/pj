package proptest

import (
	"fmt"
	"pj/internal/catalog"
	"regexp"
	"slices"
	"strings"

	"pgregory.net/rapid"
)

var (
	pathSegmentGen = rapid.StringMatching(`[a-z]{5,10}`)
	iterDirGen     = rapid.StringMatching(`[a-z]{8}`)
	subdirGen      = rapid.StringMatching(`[a-z]{6}`)
	shortQueryGen  = rapid.StringMatching(`[a-z]{1,5}`)
	searchQueryGen = rapid.StringMatching(`[a-z]{1,10}`)
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
	qualifierPrefix  = regexp.MustCompile(`^[A-Za-z]+:`)
	tagGen           = rapid.StringMatching(`[a-m0-9]([a-m0-9_.:/+-]{0,8}[a-m0-9_./+-])?`)
	tagsGen          = rapid.SliceOfN(tagGen, 0, 5)
	optionalFieldGen = rapid.OneOf(rapid.Just(""), rapid.StringMatching(`[a-m]{1,10}`))
	globValueGen     = rapid.SampledFrom([]string{"prefix*", "*suffix", "a?c", "[ab]*", "*", "*a*", "?"})
	punctValueGen    = rapid.SampledFrom([]string{"a=b", "a~b", "a!b", "a:b", `a\b`, "a-b", "é", "你好", "AbC"})
	listValueGen     = rapid.OneOf(rapid.StringMatching(`[a-m0-9]{1,6}`), globValueGen, punctValueGen, rapid.Just("-x"))
	textValueGen     = rapid.OneOf(rapid.StringMatching(`[a-m0-9]{1,6}`), globValueGen, punctValueGen, rapid.Just("a,b")).
				Filter(func(s string) bool { return !qualifierPrefix.MatchString(s) && s != "NOT" })
	quotedValueGen = rapid.OneOf(
		rapid.StringMatching(`[a-m]{1,6}`),
		rapid.SampledFrom([]string{"web server", `say "hi"`, `back\slash`, "a,b", "why?", "*", "[x", "NOT", "-x", "name:x", "tab\tthere", "Éclair"}),
		rapid.StringN(1, 8, -1),
	)
	pathValueGen = rapid.OneOf(listValueGen, rapid.SampledFrom([]string{"~", "~/", "~/a*", "/p/*", "/p/?-a", "p/"}))
	valueGen     = func(bare *rapid.Generator[string]) *rapid.Generator[catalog.Value] {
		return rapid.Custom(func(t *rapid.T) catalog.Value {
			if rapid.Bool().Draw(t, "quoted") {
				return catalog.Value{Text: quotedValueGen.Draw(t, "text"), Quoted: true}
			}
			return catalog.Value{Text: bare.Draw(t, "text")}
		})
	}
	fieldQualifierGen = rapid.SampledFrom([]string{"name", "desc", "editor", "path"})
	tagValueGen       = rapid.Custom(func(t *rapid.T) catalog.Value {
		switch rapid.IntRange(0, 3).Draw(t, "form") {
		case 0:
			return catalog.Value{Text: tagGen.Draw(t, "tag"), Quoted: true}
		case 1:
			return catalog.Value{Text: strings.ToUpper(tagGen.Draw(t, "tag"))}
		case 2:
			return catalog.Value{Text: rapid.SampledFrom([]string{"*", "a*", "*/*", "?", "[ab]*", "*:*", "*a*"}).Draw(t, "glob")}
		default:
			return catalog.Value{Text: tagGen.Draw(t, "tag")}
		}
	})
	termGen = rapid.Custom(func(t *rapid.T) catalog.Term {
		term := catalog.Term{Negated: rapid.Bool().Draw(t, "negated")}
		switch kind := rapid.SampledFrom([]string{"text", "field", "tag", "no", "is"}).Draw(t, "kind"); kind {
		case "text":
			term.Values = []catalog.Value{valueGen(textValueGen).Draw(t, "value")}
		case "field":
			term.Qualifier = fieldQualifierGen.Draw(t, "qualifier")
			bare := listValueGen
			if term.Qualifier == "path" {
				bare = pathValueGen
			}
			term.Values = rapid.SliceOfN(valueGen(bare), 1, 3).Draw(t, "values")
		case "tag":
			term.Qualifier = kind
			term.Values = rapid.SliceOfN(tagValueGen, 1, 3).Draw(t, "values")
		case "no":
			term.Qualifier = kind
			term.Values = rapid.SliceOfN(rapid.Map(rapid.SampledFrom([]string{"desc", "editor", "tag"}), func(s string) catalog.Value { return catalog.Value{Text: s} }), 1, 2).Draw(t, "values")
		case "is":
			term.Qualifier = kind
			term.Values = []catalog.Value{{Text: "missing"}}
		}
		return term
	})
	sortGen = rapid.Custom(func(t *rapid.T) *catalog.Sort {
		if rapid.Bool().Draw(t, "unsorted") {
			return nil
		}
		return &catalog.Sort{
			Key:  rapid.SampledFrom([]catalog.SortKey{catalog.SortName, catalog.SortOpened, catalog.SortAdded, catalog.SortModified}).Draw(t, "key"),
			Desc: rapid.Bool().Draw(t, "desc"),
		}
	})
	queryGen = rapid.Custom(func(t *rapid.T) catalog.Query {
		q := catalog.Query{Terms: rapid.SliceOfN(termGen, 0, 4).Draw(t, "terms"), Sort: sortGen.Draw(t, "sort")}
		if len(q.Terms) == 0 && q.Sort == nil {
			q.Terms = []catalog.Term{termGen.Draw(t, "term")}
		}
		return q
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

// QueryGen generates parseable queries covering every qualifier, negation, quoting, globs and sorts.
func QueryGen() *rapid.Generator[catalog.Query] { return queryGen }

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
			return generatedFilter{filter: mustCompile(t, termGen.Draw(t, "term")), root: kind}
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
