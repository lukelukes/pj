package proptest

import (
	"fmt"
	"pj/internal/catalog"

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

func filterOptionsGen() *rapid.Generator[catalog.FilterOptions] {
	return rapid.Custom(func(t *rapid.T) catalog.FilterOptions {
		var query string
		if rapid.Bool().Draw(t, "hasQuery") {
			query = queryGen.Draw(t, "query")
		}

		sortFields := []catalog.SortField{"", catalog.SortByName, catalog.SortByPath, catalog.SortByLastAccessed, catalog.SortByAddedAt}

		return catalog.FilterOptions{
			Query:      query,
			SortBy:     rapid.SampledFrom(sortFields).Draw(t, "sortBy"),
			Descending: rapid.Bool().Draw(t, "desc"),
		}
	})
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
