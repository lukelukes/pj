package main

import (
	"pj/internal/catalog"
	"pj/proptest"
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
			filters[i], err = parsed.Compile()
			require.NoError(t, err)
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
		require.Equal(t, catalog.Apply(ps, expected), selected)
	})
}
