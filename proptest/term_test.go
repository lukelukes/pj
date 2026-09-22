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
		_ = term.Filter()(projectGen.Draw(t, "project"))
	})
}

func TestTermCompleteness(t *testing.T) {
	for _, field := range catalog.FilterFields() {
		for _, op := range []string{"=", "!=", "~", "!~"} {
			term := catalog.Term{Field: field, Op: op}
			got, err := catalog.ParseTerm(term.String())
			require.NoError(t, err)
			require.Equal(t, term, got)
		}
	}
}

func projectField(p catalog.Project, field string) []string {
	switch field {
	case "tag":
		return p.Tags
	case "name":
		return []string{p.Name}
	case "path":
		return []string{p.Path}
	case "editor":
		return []string{p.Editor}
	case "desc":
		return []string{p.Description}
	default:
		panic("missing generated field: " + field)
	}
}

func TestProperty_TermComplements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := termGen.Draw(t, "term")
		p := projectGen.Draw(t, "project")
		for _, op := range []string{"=", "~"} {
			term.Op = op
			positive := term.Filter()(p)
			term.Op = "!" + op
			require.NotEqual(t, positive, term.Filter()(p), InvNegatedOpIsComplement)
		}
	})
}

func TestProperty_TermSelfMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p, other := projectGen.Draw(t, "project"), projectGen.Draw(t, "other")
		for _, field := range catalog.FilterFields() {
			values := projectField(p, field)
			require.Equal(t, len(values) > 0, catalog.Term{Field: field, Op: "~"}.Filter()(p), InvSubstringSelfMatch)
			for _, value := range values {
				require.Len(t, catalog.Apply([]catalog.Project{p}, catalog.Term{Field: field, Op: "=", Value: value}.Filter()), 1, InvExactSelfMatch)
				require.Equal(t, slices.Contains(projectField(other, field), value), catalog.Term{Field: field, Op: "=", Value: value}.Filter()(other), InvExactSelfMatch)
				start := rapid.IntRange(0, len(value)).Draw(t, "start")
				end := rapid.IntRange(start, len(value)).Draw(t, "end")
				sub := value[start:end]
				require.Len(t, catalog.Apply([]catalog.Project{p}, catalog.Term{Field: field, Op: "~", Value: sub}.Filter()), 1, InvSubstringSelfMatch)
				require.True(t, catalog.Term{Field: field, Op: "~"}.Filter()(p), InvSubstringSelfMatch)
				require.Equal(t, catalog.Term{Field: field, Op: "~", Value: sub}.Filter()(other), catalog.Term{Field: field, Op: "~", Value: strings.ToUpper(sub)}.Filter()(other), InvSubstringCaseInsens)
				require.True(t, catalog.Term{Field: field, Op: "~", Value: strings.ToUpper(sub)}.Filter()(p), InvSubstringCaseInsens)
				// path.Match wildcards cannot span a slash; replace within the final segment.
				base := strings.LastIndex(value, "/") + 1
				a := rapid.IntRange(base, len(value)).Draw(t, "globStart")
				b := rapid.IntRange(a, len(value)).Draw(t, "globEnd")
				glob := value[:a] + "*" + value[b:]
				require.Len(t, catalog.Apply([]catalog.Project{p}, catalog.Term{Field: field, Op: "=", Value: glob}.Filter()), 1, InvGlobSelfMatch)
				if base < len(value) {
					i := rapid.IntRange(base, len(value)-1).Draw(t, "questionIndex")
					glob = value[:i] + "?" + value[i+1:]
					require.True(t, catalog.Term{Field: field, Op: "=", Value: glob}.Filter()(p), InvGlobSelfMatch)
				}
			}
		}
	})
}

func TestProperty_TermSentinel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		sentinel := rapid.StringMatching(`[n-z]{4}`).Draw(t, "sentinel")
		for _, field := range catalog.FilterFields() {
			for _, op := range []string{"=", "~"} {
				require.Empty(t, catalog.Apply(ps, catalog.Term{Field: field, Op: op, Value: sentinel}.Filter()), InvSentinelNeverMatches)
			}
			require.Len(t, catalog.Apply(ps, catalog.Term{Field: field, Op: "!~", Value: sentinel}.Filter()), len(ps), InvSentinelNeverMatches)
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
		got := catalog.Apply(h.Catalog.List(), catalog.Or(catalog.Term{Field: "name", Op: "~", Value: q}.Filter(), catalog.Term{Field: "path", Op: "~", Value: q}.Filter()))
		assertSameIDs(h.T, h.Catalog.Search(q), got)
	})
}

func TestGenerators_Coverage(t *testing.T) {
	buckets := map[string]bool{}
	for i := range 1000 {
		term := termGen.Example(i)
		buckets["field:"+term.Field], buckets["op:"+term.Op] = true, true
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
		if len(p.Tags) == 0 {
			buckets["emptyTags"] = true
		}
		if len(p.Tags) > 1 {
			buckets["multipleTags"] = true
		}
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
	expected := []string{"field:tag", "emptyTags", "multipleTags", "empty", "glob", "=", "~", "!", "emptyEditor", "description", "bareTerm", "emptyAnd", "not"}
	for _, field := range catalog.FilterFields() {
		expected = append(expected, "field:"+field)
	}
	for _, op := range []string{"=", "!=", "~", "!~"} {
		expected = append(expected, "op:"+op)
	}
	for _, bucket := range expected {
		require.True(t, buckets[bucket], "generator missed %s", bucket)
	}
}
