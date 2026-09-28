package proptest

import (
	"pj/internal/catalog"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestProperty_ApplyStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		// Duplicate input elements exercise conservation as a multiset.
		if len(ps) > 0 && rapid.Bool().Draw(t, "duplicate") {
			ps = append(ps, ps[0])
		}
		original := slices.Clone(ps)
		f := filterGen(rapid.IntRange(0, 3).Draw(t, "depth")).Draw(t, "filter")
		got := catalog.Apply(ps, f)
		counts := map[string]int{}
		for _, p := range ps {
			counts[p.ID]++
		}
		cursor := 0
		for _, p := range got {
			counts[p.ID]--
			require.GreaterOrEqual(t, counts[p.ID], 0, InvFilterSubsetOfList)
			for cursor < len(ps) && !reflect.DeepEqual(ps[cursor], p) {
				cursor++
			}
			require.Less(t, cursor, len(ps), InvApplyPreservesOrder)
			cursor++
		}
		require.True(t, reflect.DeepEqual(original, ps), "Apply mutated input")
		require.Equal(t, got, catalog.Apply(got, f), InvApplyIdempotent)
	})
}

func TestProperty_FilterAlgebra(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		f, g := filterGen(3).Draw(t, "f"), filterGen(3).Draw(t, "g")
		left, right := catalog.Apply(ps, f), catalog.Apply(ps, g)
		intersection, union, complement := []catalog.Project{}, []catalog.Project{}, []catalog.Project{}
		for _, p := range ps {
			a, b := slices.ContainsFunc(left, func(q catalog.Project) bool { return q.ID == p.ID }), slices.ContainsFunc(right, func(q catalog.Project) bool { return q.ID == p.ID })
			if a && b {
				intersection = append(intersection, p)
			}
			if a || b {
				union = append(union, p)
			}
			if !a {
				complement = append(complement, p)
			}
		}
		require.Equal(t, intersection, catalog.Apply(ps, catalog.And(f, g)), InvAndIsIntersection)
		require.Equal(t, union, catalog.Apply(ps, catalog.Or(f, g)), InvOrIsUnion)
		require.Equal(t, complement, catalog.Apply(ps, catalog.Not(f)), InvNotIsComplement)
		require.True(t, reflect.DeepEqual(ps, catalog.Apply(ps, catalog.And())), InvEmptyAndMatchesAll)
		require.Empty(t, catalog.Apply(ps, catalog.Or()), InvEmptyAndMatchesAll)
		require.Equal(t, left, catalog.Apply(ps, catalog.And(f)))
		require.Equal(t, left, catalog.Apply(ps, catalog.Or(f)))
		require.Equal(t, catalog.Apply(ps, catalog.Not(catalog.And(f, g))), catalog.Apply(ps, catalog.Or(catalog.Not(f), catalog.Not(g))), InvDeMorgan)
		fs := rapid.SliceOfN(filterGen(3), 0, 3).Draw(t, "filters")
		shuffled := Permute(t, fs)
		require.Equal(t, catalog.Apply(ps, catalog.And(fs...)), catalog.Apply(ps, catalog.And(shuffled...)), InvAndOrCommutative)
		require.Equal(t, catalog.Apply(ps, catalog.Or(fs...)), catalog.Apply(ps, catalog.Or(shuffled...)), InvAndOrCommutative)
	})
}
