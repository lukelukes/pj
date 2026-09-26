package proptest

import (
	"errors"
	"pj/internal/catalog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestProperty_TermRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := termGen.Draw(t, "term")
		got, err := catalog.ParseTerm(term.String())
		require.NoError(t, err, InvTermRoundTrip)
		require.Equal(t, term, got, InvTermRoundTrip)
	})
}

func TestProperty_TermParseTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.String().Draw(t, "raw")
		term, err := catalog.ParseTerm(raw)
		if err != nil {
			require.True(t, errors.Is(err, catalog.ErrUnknownField) || errors.Is(err, catalog.ErrUnknownOp) || errors.Is(err, catalog.ErrBadGlob), InvTermParseTotal)
			return
		}
		require.Equal(t, raw, term.String(), InvTermParseTotal)
		_ = mustCompile(t, term)(projectGen.Draw(t, "project"))
	})
}

func TestProperty_TermCompileValidates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := catalog.Term{
			Field: rapid.OneOf(fieldNameGen, rapid.String()).Draw(t, "field"),
			Op:    rapid.OneOf(opGen, rapid.Map(rapid.String(), func(s string) catalog.Op { return catalog.Op(s) })).Draw(t, "op"),
			Value: rapid.OneOf(termValueGen, rapid.String()).Draw(t, "value"),
		}
		f, err := term.Compile()
		if err != nil {
			require.Nil(t, f, InvTermCompileValidates)
			require.True(t, errors.Is(err, catalog.ErrUnknownField) || errors.Is(err, catalog.ErrUnknownOp) || errors.Is(err, catalog.ErrBadGlob), InvTermCompileValidates)
			return
		}
		require.True(t, slices.Contains(catalog.FilterFields(), term.Field), InvTermCompileValidates)
		require.True(t, slices.Contains(catalog.FilterOps(), term.Op), InvTermCompileValidates)
		parsed, err := catalog.ParseTerm(term.String())
		require.NoError(t, err, InvTermCompileValidates)
		require.Equal(t, term, parsed, InvTermCompileValidates)
		_ = f(projectGen.Draw(t, "project"))
	})
}

func TestTermCompleteness(t *testing.T) {
	for _, field := range catalog.FilterFields() {
		for _, op := range catalog.FilterOps() {
			term := catalog.Term{Field: field, Op: op}
			got, err := catalog.ParseTerm(term.String())
			require.NoError(t, err)
			require.Equal(t, term, got)
		}
	}
}

func projectField(p catalog.Project, field string) string {
	switch field {
	case "name":
		return p.Name
	case "path":
		return p.Path
	case "editor":
		return p.Editor
	case "desc":
		return p.Description
	default:
		panic("missing generated field: " + field)
	}
}

func TestProperty_TermComplements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := termGen.Draw(t, "term")
		p := projectGen.Draw(t, "project")
		for _, pair := range [][2]catalog.Op{{catalog.OpEqual, catalog.OpNotEqual}, {catalog.OpContains, catalog.OpNotContains}} {
			term.Op = pair[0]
			positive := mustCompile(t, term)(p)
			term.Op = pair[1]
			require.NotEqual(t, positive, mustCompile(t, term)(p), InvNegatedOpIsComplement)
		}
	})
}

func TestProperty_TermSelfMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p, other := projectGen.Draw(t, "project"), projectGen.Draw(t, "other")
		for _, field := range catalog.FilterFields() {
			value := projectField(p, field)
			require.Len(t, catalog.Apply([]catalog.Project{p}, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpEqual, Value: value})), 1, InvExactSelfMatch)
			require.Equal(t, value == projectField(other, field), mustCompile(t, catalog.Term{Field: field, Op: catalog.OpEqual, Value: value})(other), InvExactSelfMatch)
			start := rapid.IntRange(0, len(value)).Draw(t, "start")
			end := rapid.IntRange(start, len(value)).Draw(t, "end")
			sub := value[start:end]
			require.Len(t, catalog.Apply([]catalog.Project{p}, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpContains, Value: sub})), 1, InvSubstringSelfMatch)
			require.True(t, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpContains})(p), InvSubstringSelfMatch)
			require.Equal(t, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpContains, Value: sub})(other), mustCompile(t, catalog.Term{Field: field, Op: catalog.OpContains, Value: strings.ToUpper(sub)})(other), InvSubstringCaseInsens)
			require.True(t, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpContains, Value: strings.ToUpper(sub)})(p), InvSubstringCaseInsens)
			// path.Match wildcards cannot span a slash; replace within the final segment.
			base := strings.LastIndex(value, "/") + 1
			a := rapid.IntRange(base, len(value)).Draw(t, "globStart")
			b := rapid.IntRange(a, len(value)).Draw(t, "globEnd")
			glob := value[:a] + "*" + value[b:]
			require.Len(t, catalog.Apply([]catalog.Project{p}, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpEqual, Value: glob})), 1, InvGlobSelfMatch)
			if base < len(value) {
				i := rapid.IntRange(base, len(value)-1).Draw(t, "questionIndex")
				glob = value[:i] + "?" + value[i+1:]
				require.True(t, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpEqual, Value: glob})(p), InvGlobSelfMatch)
			}
		}
	})
}

func TestProperty_TermSentinel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		sentinel := rapid.StringMatching(`[n-z]{4}`).Draw(t, "sentinel")
		for _, field := range catalog.FilterFields() {
			for _, op := range []catalog.Op{catalog.OpEqual, catalog.OpContains} {
				require.Empty(t, catalog.Apply(ps, mustCompile(t, catalog.Term{Field: field, Op: op, Value: sentinel})), InvSentinelNeverMatches)
			}
			require.Len(t, catalog.Apply(ps, mustCompile(t, catalog.Term{Field: field, Op: catalog.OpNotContains, Value: sentinel})), len(ps), InvSentinelNeverMatches)
		}
	})
}

func TestProperty_SearchEqualsFilter(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		ps := h.AddProjects(0, 10)
		q := rapid.OneOf(rapid.Just(""), shortQueryGen).Draw(h.T, "query")
		if len(ps) > 0 && rapid.Bool().Draw(h.T, "selfQuery") {
			q = strings.ToUpper(ps[0].Name[:1])
		}
		got := catalog.Apply(h.Catalog.List(), catalog.Or(mustCompile(h.T, catalog.Term{Field: "name", Op: catalog.OpContains, Value: q}), mustCompile(h.T, catalog.Term{Field: "path", Op: catalog.OpContains, Value: q})))
		assertSameIDs(h.T, h.Catalog.Search(q), got)
	})
}

func TestGenerators_Coverage(t *testing.T) {
	buckets := map[string]bool{}
	for i := range 1000 {
		term := termGen.Example(i)
		buckets["field:"+term.Field], buckets["op:"+string(term.Op)] = true, true
		if term.Value == "" {
			buckets["empty"] = true
		}
		if strings.ContainsAny(term.Value, "*?[") {
			buckets["glob"] = true
		}
		for _, char := range []string{"=", "~", "!"} {
			if strings.Contains(term.Value, char) {
				buckets[char] = true
			}
		}
		p := projectGen.Example(i)
		if p.Editor == "" {
			buckets["emptyEditor"] = true
		}
		if p.Description != "" {
			buckets["description"] = true
		}
		f := filterTreeGen(3).Example(i)
		require.LessOrEqual(t, f.depth, 3)
		if f.root == "term" && f.depth == 0 {
			buckets["bareTerm"] = true
		}
		if f.root == "and" && f.children == 0 {
			buckets["emptyAnd"] = true
		}
		if f.root == "not" {
			buckets["not"] = true
		}
	}
	expected := []string{"empty", "glob", "=", "~", "!", "emptyEditor", "description", "bareTerm", "emptyAnd", "not"}
	for _, field := range catalog.FilterFields() {
		expected = append(expected, "field:"+field)
	}
	for _, op := range catalog.FilterOps() {
		expected = append(expected, "op:"+string(op))
	}
	for _, bucket := range expected {
		require.True(t, buckets[bucket], "generator missed %s", bucket)
	}
}
