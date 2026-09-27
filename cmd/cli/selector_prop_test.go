package main

import (
	"pj/internal/catalog"
	"pj/proptest"
	"strings"
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
	env := catalog.QueryEnv{Home: "/p", Missing: func(p catalog.Project) bool { return len(p.Name)%2 == 0 }}
	compile := func(t *rapid.T, s Selector) catalog.Filter {
		q, err := s.Query()
		require.NoError(t, err)
		f, err := q.Compile(env)
		require.NoError(t, err)
		return f
	}
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		queries := rapid.SliceOfN(proptest.QueryGen(), 0, 4).Draw(t, "queries")
		raw := make([]string, len(queries))
		filters := make([]catalog.Filter, len(queries))
		for i := range queries {
			if i > 0 {
				queries[i].Sort = nil
			}
			if len(queries[i].Terms) == 0 && queries[i].Sort == nil {
				queries[i].Terms = []catalog.Term{{Values: []catalog.Value{{Text: "a", Quoted: true}}}}
			}
			raw[i] = queries[i].String()
			var err error
			filters[i], err = queries[i].Compile(env)
			require.NoError(t, err)
		}
		selected := catalog.Apply(ps, compile(t, Selector{Filters: raw}))
		require.Equal(t, catalog.Apply(ps, catalog.And(filters...)), selected, proptest.InvMergeIsAnd)
		permuted := catalog.Apply(ps, compile(t, Selector{Filters: proptest.Permute(t, raw)}))
		require.Equal(t, selected, permuted, proptest.InvSelectorOrderIrrelevant)
	})
}

func TestProperty_SelectorOnCatalog(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		name := rapid.StringMatching(`[a-m]{1,3}`).Draw(t, "name")
		got, sort, err := Selector{Filters: []string{"name:" + name + " sort:added"}}.Select(selectionCatalog{projects: ps})
		require.NoError(t, err)
		require.Equal(t, &catalog.Sort{Key: catalog.SortAdded, Desc: true}, sort)
		var want []catalog.Project
		for _, p := range ps {
			if strings.Contains(p.Name, name) {
				want = append(want, p)
			}
		}
		require.ElementsMatch(t, want, got)
		missing, _, err := Selector{Filters: []string{"is:missing"}}.Select(selectionCatalog{projects: ps})
		require.NoError(t, err)
		require.Len(t, missing, len(ps))
	})
}
