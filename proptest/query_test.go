package proptest

import (
	"errors"
	"pj/internal/catalog"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func normalize(q catalog.Query) catalog.Query {
	if len(q.Terms) == 0 {
		q.Terms = nil
	}
	return q
}

func TestProperty_QueryRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := normalize(queryGen.Draw(t, "query"))
		got, err := catalog.ParseQuery(q.String())
		require.NoError(t, err, InvQueryRoundTrip)
		require.Equal(t, q, got, InvQueryRoundTrip)
	})
}

var syntaxRuneGen = rapid.SampledFrom([]rune("anNOTsi:-,\"\\ \t*?[]^~/é"))

func TestProperty_QueryParseTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.OneOf(
			rapid.String(),
			rapid.Map(rapid.SliceOfN(syntaxRuneGen, 0, 20), func(rs []rune) string { return string(rs) }),
		).Draw(t, "raw")
		q, err := catalog.ParseQuery(raw)
		if err != nil {
			require.True(t, errors.Is(err, catalog.ErrQuerySyntax) || errors.Is(err, catalog.ErrUnknownQualifier) || errors.Is(err, catalog.ErrBadSort), InvQueryParseTotal)
			return
		}
		again, err := catalog.ParseQuery(q.String())
		require.NoError(t, err, InvQueryParseTotal)
		require.Equal(t, q, again, InvQueryParseTotal)
		f, err := q.Compile(testEnv)
		if err != nil {
			require.True(t, errors.Is(err, catalog.ErrBadGlob) || errors.Is(err, catalog.ErrBadValue), InvQueryParseTotal)
			return
		}
		_ = f(projectGen.Draw(t, "project"))
	})
}

func TestProperty_QueryMatchesOracle(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		q := queryGen.Draw(t, "query")
		f, err := q.Compile(testEnv)
		require.NoError(t, err)
		for _, p := range ps {
			require.Equal(t, oracleQuery(q, p), f(p), "%s on %+v: %s", q, p, InvQueryMatchesOracle)
		}
	})
}

func TestProperty_NegationIsComplement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := termGen.Draw(t, "term")
		p := projectGen.Draw(t, "project")
		flipped := term
		flipped.Negated = !term.Negated
		require.NotEqual(t, mustCompile(t, term)(p), mustCompile(t, flipped)(p), InvNegationIsComplement)
	})
}

func TestProperty_TermIsOrOfValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		term := termGen.Draw(t, "term")
		term.Negated = false
		p := projectGen.Draw(t, "project")
		single := make([]catalog.Filter, len(term.Values))
		for i, v := range term.Values {
			single[i] = mustCompile(t, catalog.Term{Qualifier: term.Qualifier, Values: []catalog.Value{v}})
		}
		require.Equal(t, catalog.Or(single...)(p), mustCompile(t, term)(p), InvTermIsOrOfValues)
		term.Negated = true
		require.Equal(t, !catalog.Or(single...)(p), mustCompile(t, term)(p), InvTermIsOrOfValues)
	})
}

func projectField(p catalog.Project, qualifier string) string {
	switch qualifier {
	case "name":
		return p.Name
	case "path":
		return p.Path
	case "editor":
		return p.Editor
	case "desc":
		return p.Description
	default:
		panic("missing generated field: " + qualifier)
	}
}

func TestProperty_SubstringSelfMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := projectGen.Draw(t, "project")
		p.Description = rapid.StringN(0, 12, -1).Draw(t, "description")
		for _, qualifier := range []string{"name", "desc", "editor", "path"} {
			value := []rune(projectField(p, qualifier))
			if len(value) == 0 {
				continue
			}
			start := rapid.IntRange(0, len(value)-1).Draw(t, "start")
			end := rapid.IntRange(start+1, len(value)).Draw(t, "end")
			sub := string(value[start:end])
			term := catalog.Term{Qualifier: qualifier, Values: []catalog.Value{{Text: sub, Quoted: true}}}
			require.True(t, mustCompile(t, term)(p), InvSubstringSelfMatch)
			if qualifier != "path" {
				variant := []rune(sub)
				for i, r := range variant {
					for range rapid.IntRange(0, 2).Draw(t, "folds") {
						r = unicode.SimpleFold(r)
					}
					variant[i] = r
				}
				term.Values[0].Text = string(variant)
				require.True(t, mustCompile(t, term)(p), InvSubstringSelfMatch)
			}
		}
	})
}

func TestProperty_SentinelNeverMatches(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		sentinel := catalog.Value{Text: rapid.StringMatching(`[n-z]{4}`).Draw(t, "sentinel")}
		for _, qualifier := range []string{"", "name", "desc", "editor", "path"} {
			term := catalog.Term{Qualifier: qualifier, Values: []catalog.Value{sentinel}}
			require.Empty(t, catalog.Apply(ps, mustCompile(t, term)), InvSentinelNeverMatches)
			term.Negated = true
			require.Len(t, catalog.Apply(ps, mustCompile(t, term)), len(ps), InvSentinelNeverMatches)
		}
	})
}

func TestProperty_MergeIsAnd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		a, b := queryGen.Draw(t, "a"), queryGen.Draw(t, "b")
		b.Sort = nil
		merged, err := catalog.Merge(a, b)
		require.NoError(t, err, InvMergeIsAnd)
		require.Equal(t, a.Sort, merged.Sort, InvMergeIsAnd)
		fa, err := a.Compile(testEnv)
		require.NoError(t, err)
		fb, err := b.Compile(testEnv)
		require.NoError(t, err)
		fm, err := merged.Compile(testEnv)
		require.NoError(t, err)
		require.Equal(t, catalog.Apply(ps, catalog.And(fa, fb)), catalog.Apply(ps, fm), InvMergeIsAnd)
		if a.Sort != nil {
			_, err = catalog.Merge(a, a)
			require.ErrorIs(t, err, catalog.ErrBadSort, InvMergeIsAnd)
		}
	})
}

func TestGenerators_Coverage(t *testing.T) {
	buckets := map[string]bool{}
	for i := range 2000 {
		q := queryGen.Example(i)
		if q.Sort != nil {
			buckets["sort:"+string(q.Sort.Key)] = true
		}
		for _, term := range q.Terms {
			buckets["qualifier:"+term.Qualifier] = true
			if term.Negated {
				buckets["negated"] = true
			}
			if len(term.Values) > 1 {
				buckets["list"] = true
			}
			for _, v := range term.Values {
				switch {
				case v.Quoted && strings.ContainsAny(v.Text, "\"\\"):
					buckets["escaped"] = true
				case v.Quoted && strings.ContainsFunc(v.Text, unicode.IsSpace):
					buckets["quotedSpace"] = true
				case !v.Quoted && strings.ContainsAny(v.Text, "*?["):
					buckets["glob"] = true
				case !v.Quoted && strings.HasPrefix(v.Text, "~"):
					buckets["tilde"] = true
				}
			}
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
		p := projectGen.Example(i)
		if p.Editor == "" {
			buckets["emptyEditor"] = true
		}
		if testMissing(p) {
			buckets["missing"] = true
		}
	}
	expected := []string{"negated", "list", "escaped", "quotedSpace", "glob", "tilde", "bareTerm", "emptyAnd", "not", "emptyEditor", "missing"}
	for _, qualifier := range []string{"", "name", "desc", "editor", "path", "no", "is"} {
		expected = append(expected, "qualifier:"+qualifier)
	}
	for _, key := range []catalog.SortKey{catalog.SortName, catalog.SortOpened, catalog.SortAdded, catalog.SortModified} {
		expected = append(expected, "sort:"+string(key))
	}
	for _, bucket := range expected {
		require.True(t, buckets[bucket], "generator missed %s", bucket)
	}
}

func oracleQuery(q catalog.Query, p catalog.Project) bool {
	for _, term := range q.Terms {
		if !oracleTerm(term, p) {
			return false
		}
	}
	return true
}

func oracleTerm(term catalog.Term, p catalog.Project) bool {
	matched := false
	for _, v := range term.Values {
		if oracleValue(term.Qualifier, v, p) {
			matched = true
		}
	}
	return matched != term.Negated
}

func oracleValue(qualifier string, v catalog.Value, p catalog.Project) bool {
	switch qualifier {
	case "":
		return oracleMatch(v, p.Name, false, false) || oracleMatch(v, p.Description, false, false)
	case "no":
		return projectField(p, v.Text) == ""
	case "is":
		return testMissing(p)
	case "path":
		text := v.Text
		if text == "~" || strings.HasPrefix(text, "~/") {
			if !strings.HasPrefix(p.Path, testHome) {
				return false
			}
			rest := catalog.Value{Text: text[1:], Quoted: v.Quoted}
			if v.Quoted || !strings.ContainsAny(rest.Text, "*?[") {
				return strings.Contains(p.Path, testHome+rest.Text)
			}
			return oracleGlob([]rune(rest.Text), []rune(p.Path[len(testHome):]), true, true)
		}
		return oracleMatch(v, p.Path, true, true)
	default:
		return oracleMatch(v, projectField(p, qualifier), false, false)
	}
}

func oracleMatch(v catalog.Value, s string, caseSensitive, slashAware bool) bool {
	if v.Quoted || !strings.ContainsAny(v.Text, "*?[") {
		if caseSensitive {
			return strings.Contains(s, v.Text)
		}
		return oracleContainsFold([]rune(s), []rune(v.Text))
	}
	return oracleGlob([]rune(v.Text), []rune(s), caseSensitive, slashAware)
}

func oracleContainsFold(s, sub []rune) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(string(s[i:i+len(sub)]), string(sub)) {
			return true
		}
	}
	return false
}

func oracleGlob(pattern, s []rune, caseSensitive, slashAware bool) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}
	switch pattern[0] {
	case '*':
		for i := 0; i <= len(s); i++ {
			if oracleGlob(pattern[1:], s[i:], caseSensitive, slashAware) {
				return true
			}
			if i < len(s) && slashAware && s[i] == '/' {
				return false
			}
		}
		return false
	case '?':
		return len(s) > 0 && !(slashAware && s[0] == '/') && oracleGlob(pattern[1:], s[1:], caseSensitive, slashAware)
	case '[':
		end := 1
		for pattern[end] != ']' {
			end++
		}
		return len(s) > 0 && oracleClass(pattern[1:end], s[0], caseSensitive) && oracleGlob(pattern[end+1:], s[1:], caseSensitive, slashAware)
	default:
		return len(s) > 0 && oracleRuneEqual(pattern[0], s[0], caseSensitive) && oracleGlob(pattern[1:], s[1:], caseSensitive, slashAware)
	}
}

func oracleClass(class []rune, r rune, caseSensitive bool) bool {
	negated := len(class) > 0 && class[0] == '^'
	if negated {
		class = class[1:]
	}
	for _, c := range class {
		if oracleRuneEqual(c, r, caseSensitive) {
			return !negated
		}
	}
	return negated
}

func oracleRuneEqual(a, b rune, caseSensitive bool) bool {
	if caseSensitive {
		return a == b
	}
	return strings.EqualFold(string(a), string(b))
}
