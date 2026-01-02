package proptest

import (
	"fmt"
	"pj/internal/catalog"

	"pgregory.net/rapid"
)

var allProjectTypes = []catalog.ProjectType{
	catalog.TypeUnknown, catalog.TypeGo, catalog.TypeRust, catalog.TypeNode, catalog.TypePython,
	catalog.TypeElixir, catalog.TypeRuby, catalog.TypeJava, catalog.TypeGeneric,
}

var allStatuses = []catalog.Status{catalog.StatusActive, catalog.StatusArchived, catalog.StatusAbandoned}

var (
	pathSegmentGen = rapid.StringMatching(`[a-z]{5,10}`)
	iterDirGen     = rapid.StringMatching(`[a-z]{8}`)
	subdirGen      = rapid.StringMatching(`[a-z]{6}`)
	shortQueryGen  = rapid.StringMatching(`[a-z]{1,5}`)
	queryGen       = rapid.StringMatching(`[a-z]{1,10}`)
	notesGen       = rapid.StringMatching(`[a-zA-Z0-9 _.,-]{0,50}`)
	numSuffixGen   = rapid.StringMatching(`[0-9]{10}`)
)

func validNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]{0,30}`)
}

func tagGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9-]{0,15}`)
}

func statusGen() *rapid.Generator[catalog.Status] {
	return rapid.SampledFrom(allStatuses)
}

func projectTypeGen() *rapid.Generator[catalog.ProjectType] {
	return rapid.SampledFrom(allProjectTypes)
}

func filterOptionsGen() *rapid.Generator[catalog.FilterOptions] {
	return rapid.Custom(func(t *rapid.T) catalog.FilterOptions {
		var status catalog.Status
		if rapid.Bool().Draw(t, "hasStatus") {
			status = statusGen().Draw(t, "status")
		}

		var types []catalog.ProjectType
		if rapid.Bool().Draw(t, "hasTypes") {
			types = rapid.SliceOfN(projectTypeGen(), filterMinSlice, filterMaxSlice).Draw(t, "types")
		}

		var tags []string
		if rapid.Bool().Draw(t, "hasTags") {
			tags = rapid.SliceOfN(tagGen(), filterMinSlice, filterMaxSlice).Draw(t, "tags")
		}

		var query string
		if rapid.Bool().Draw(t, "hasQuery") {
			query = queryGen.Draw(t, "query")
		}

		sortFields := []catalog.SortField{"", catalog.SortByName, catalog.SortByPath, catalog.SortByLastAccessed, catalog.SortByAddedAt, catalog.SortByTypes}

		return catalog.FilterOptions{
			Status:     status,
			Types:      types,
			Tags:       tags,
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
    status: active
    %s: %s
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`, extraField, extraValue, extraField, extraValue)
	})
}

func invalidTypesGen() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: invalid_status_value
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: active
    types:
      - invalid_type_value
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`),
		rapid.Just(`version: "not_a_number"
projects: []
`),
		rapid.Just(`version: 1
projects:
  - id: 12345
    name: test-project
    path: /tmp/test
    status: active
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: [not, a, string]
    path: /tmp/test
    status: active
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: active
    tags: "not-a-list"
`),
		rapid.Just(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: active
    added_at: "not-a-date"
`),
		rapid.Custom(func(t *rapid.T) string {
			invalidStatus := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "invalidStatus")
			return fmt.Sprintf(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: %s
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`, invalidStatus)
		}),
		rapid.Custom(func(t *rapid.T) string {
			invalidType := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "invalidType")
			return fmt.Sprintf(`version: 1
projects:
  - id: test-id
    name: test-project
    path: /tmp/test
    status: active
    types:
      - %s
    added_at: 2024-01-01T00:00:00Z
    last_accessed: 2024-01-01T00:00:00Z
`, invalidType)
		}),
	)
}
