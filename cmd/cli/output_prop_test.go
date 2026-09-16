package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"pj/internal/catalog"
	"pj/proptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestProperty_PlainProjections(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		original := slices.Clone(ps)
		sorted := slices.Clone(ps)
		sortProjects(sorted)
		require.True(t, slices.IsSortedFunc(sorted, func(a, b catalog.Project) int { return cmp.Compare(a.Name, b.Name) }), "projection must sort by name ascending")
		for _, output := range []Output{OutputNames, OutputPaths} {
			var buf, shuffled bytes.Buffer
			require.NoError(t, printProjects(&buf, ps, output))
			require.NoError(t, printProjects(&shuffled, proptest.Permute(t, ps), output))
			require.Equal(t, buf.String(), shuffled.String(), proptest.InvProjectionLineCount)
			require.Equal(t, len(ps), strings.Count(buf.String(), "\n"), proptest.InvProjectionLineCount)
			lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
			for i, p := range sorted {
				expected := p.Name
				if output == OutputPaths {
					expected = p.Path
				}
				require.Equal(t, expected, lines[i])
			}
		}
		require.True(t, slices.Equal(original, ps), "printing mutated input")
	})
}

func TestProperty_JSONProjection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		// JSON must preserve arbitrary text, even newlines and glob characters.
		for i := range ps {
			ps[i].Editor = rapid.String().Draw(t, "editor")
			ps[i].Description = rapid.String().Draw(t, "description")
		}
		var buf, shuffled bytes.Buffer
		require.NoError(t, printProjects(&buf, ps, OutputJSON))
		require.NoError(t, printProjects(&shuffled, proptest.Permute(t, ps), OutputJSON))
		require.Equal(t, buf.String(), shuffled.String())
		var rows []projectJSON
		require.NoError(t, json.Unmarshal(buf.Bytes(), &rows), proptest.InvJSONRoundTrip)
		require.NotNil(t, rows, proptest.InvJSONRoundTrip)
		require.Len(t, rows, len(ps), proptest.InvJSONRoundTrip)
		sortProjects(ps)
		for i, row := range rows {
			p := ps[i]
			require.Equal(t, p.ID, row.ID)
			require.Equal(t, p.Name, row.Name)
			require.Equal(t, p.Path, row.Path)
			require.Equal(t, p.Editor, row.Editor)
			require.Equal(t, p.Description, row.Description)
			require.True(t, p.AddedAt.Equal(row.AddedAt))
			require.True(t, p.LastAccessed.Equal(row.LastAccessed))
		}
	})
}

func TestEmptyJSONProjection(t *testing.T) {
	for _, ps := range [][]catalog.Project{nil, {}} {
		var buf bytes.Buffer
		require.NoError(t, printProjects(&buf, ps, OutputJSON))
		require.Equal(t, "[]\n", buf.String())
	}
}
