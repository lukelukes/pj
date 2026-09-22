package main

import (
	"cmp"
	"pj/internal/catalog"
	"pj/proptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

type selectionCatalog struct {
	catalog.Catalog
	projects []catalog.Project
}

func (c selectionCatalog) List() []catalog.Project { return c.projects }

func TestProperty_Selector(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		terms := rapid.SliceOfN(proptest.TermGen(), 0, 5).Draw(t, "terms")
		raw := make([]string, len(terms))
		filters := make([]catalog.Filter, len(terms))
		for i, term := range terms {
			raw[i] = term.String()
			parsed, err := catalog.ParseTerm(raw[i])
			require.NoError(t, err)
			filters[i] = parsed.Filter()
		}
		selector := Selector{Filters: raw}
		f, err := selector.Filter()
		require.NoError(t, err)
		expected := catalog.And(filters...)
		for _, p := range ps {
			require.Equal(t, expected(p), f(p))
		}
		selected, err := selector.Select(selectionCatalog{projects: ps})
		require.NoError(t, err)
		permuted, err := (Selector{Filters: proptest.Permute(t, raw)}).Select(selectionCatalog{projects: ps})
		require.NoError(t, err)
		require.Equal(t, selected, permuted, proptest.InvSelectorOrderIrrelevant)
		require.True(t, slices.IsSortedFunc(selected, func(a, b catalog.Project) int { return cmp.Compare(a.Name, b.Name) }), "selection must sort by name ascending")
		want := catalog.Apply(ps, expected)
		sortProjects(want)
		require.Equal(t, want, selected)
	})
}
