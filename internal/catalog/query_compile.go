package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"
)

var ErrNoHome = errors.New("cannot expand ~: home directory unknown")

// QueryEnv supplies the facts a query needs beyond the project itself.
type QueryEnv struct {
	Home    string
	Missing func(Project) bool
}

type matchMode struct {
	caseSensitive bool
	slashAware    bool
	tilde         bool
	tag           bool
}

type qualifier struct {
	fields func(Project) []string
	mode   matchMode
	values map[string]func(QueryEnv) (Filter, error)
}

func field(get func(Project) string) func(Project) []string {
	return func(p Project) []string { return []string{get(p)} }
}

var (
	nameField   = field(func(p Project) string { return p.Name })
	descField   = field(func(p Project) string { return p.Description })
	editorField = field(func(p Project) string { return p.Editor })
	pathField   = field(func(p Project) string { return p.Path })
	tagsField   = func(p Project) []string { return p.Tags }
)

func empty(get func(Project) []string) func(QueryEnv) (Filter, error) {
	return func(QueryEnv) (Filter, error) {
		return func(p Project) bool { return !slices.ContainsFunc(get(p), func(s string) bool { return s != "" }) }, nil
	}
}

var qualifiers = map[string]qualifier{
	"name":   {fields: nameField},
	"desc":   {fields: descField},
	"editor": {fields: editorField},
	"path":   {fields: pathField, mode: matchMode{caseSensitive: true, slashAware: true, tilde: true}},
	"tag":    {fields: tagsField, mode: matchMode{tag: true}},
	"no": {values: map[string]func(QueryEnv) (Filter, error){
		"desc":   empty(descField),
		"editor": empty(editorField),
		"tag":    empty(tagsField),
	}},
	"is": {values: map[string]func(QueryEnv) (Filter, error){
		"missing": func(env QueryEnv) (Filter, error) {
			if env.Missing == nil {
				return nil, errors.New("is:missing needs a filesystem snapshot")
			}
			return Filter(env.Missing), nil
		},
	}},
}

var textTerm = qualifier{fields: func(p Project) []string { return []string{p.Name, p.Description} }}

// Compile returns the filter matching projects that satisfy every term.
func (q Query) Compile(env QueryEnv) (Filter, error) {
	filters := make([]Filter, len(q.Terms))
	for i, t := range q.Terms {
		f, err := t.Compile(env)
		if err != nil {
			return nil, fmt.Errorf("term %q: %w", t.String(), err)
		}
		filters[i] = f
	}
	return And(filters...), nil
}

// Compile returns the filter for one term. Negation complements the whole OR of its values.
func (t Term) Compile(env QueryEnv) (Filter, error) {
	q := textTerm
	if t.Qualifier != "" {
		var ok bool
		if q, ok = qualifiers[t.Qualifier]; !ok {
			return nil, fmt.Errorf("%w %q (valid qualifiers: %s)", ErrUnknownQualifier, t.Qualifier, strings.Join(Qualifiers(), ", "))
		}
	}
	if len(t.Values) == 0 {
		return nil, fmt.Errorf("%w: empty value for qualifier %q", ErrQuerySyntax, t.Qualifier)
	}
	alternatives := make([]Filter, len(t.Values))
	for i, v := range t.Values {
		f, err := q.compileValue(t.Qualifier, v, env)
		if err != nil {
			return nil, err
		}
		alternatives[i] = f
	}
	f := Or(alternatives...)
	if t.Negated {
		return Not(f), nil
	}
	return f, nil
}

func (q qualifier) compileValue(name string, v Value, env QueryEnv) (Filter, error) {
	if q.values != nil {
		build, ok := q.values[v.Text]
		if !ok {
			valid := slices.Sorted(maps.Keys(q.values))
			return nil, fmt.Errorf("%w %s:%s (valid values: %s)", ErrBadValue, name, v.Text, strings.Join(valid, ", "))
		}
		return build(env)
	}
	match, err := matcher(v, q.mode, env.Home)
	if err != nil {
		return nil, err
	}
	return func(p Project) bool { return slices.ContainsFunc(q.fields(p), match) }, nil
}

func tagMatcher(v Value) (func(string) bool, error) {
	text := strings.ToLower(strings.TrimSpace(v.Text))
	if !v.Quoted && strings.ContainsAny(text, "*?[") {
		re, err := globRegexp("", text, matchMode{})
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}
	tag, err := NormalizeTag(text)
	if err != nil {
		return nil, err
	}
	return func(s string) bool { return s == tag }, nil
}

func matcher(v Value, mode matchMode, home string) (func(string) bool, error) {
	if mode.tag {
		return tagMatcher(v)
	}
	text, prefix := v.Text, ""
	if mode.tilde && (text == "~" || strings.HasPrefix(text, "~/")) {
		if home == "" {
			return nil, ErrNoHome
		}
		text, prefix = text[1:], home
	}
	if v.Quoted || !strings.ContainsAny(text, "*?[") {
		needle := prefix + text
		if mode.caseSensitive {
			return func(s string) bool { return strings.Contains(s, needle) }, nil
		}
		return regexp.MustCompile("(?i)" + regexp.QuoteMeta(needle)).MatchString, nil
	}
	re, err := globRegexp(prefix, text, mode)
	if err != nil {
		return nil, err
	}
	return re.MatchString, nil
}

func globSyntax(mode matchMode) (flags, star, one string) {
	flags, star, one = "(?si)^", ".*", "."
	if mode.caseSensitive {
		flags = "(?s)^"
	}
	if mode.slashAware {
		star, one = "[^/]*", "[^/]"
	}
	return flags, star, one
}

func globRegexp(prefix, pattern string, mode matchMode) (*regexp.Regexp, error) {
	flags, star, one := globSyntax(mode)
	var b strings.Builder
	b.WriteString(flags + regexp.QuoteMeta(prefix))
	s := []rune(pattern)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '*':
			if i+1 < len(s) && s[i+1] == '*' {
				return nil, fmt.Errorf("%w %q: ** is not supported", ErrBadGlob, pattern)
			}
			b.WriteString(star)
		case '?':
			b.WriteString(one)
		case '[':
			end, class, err := globClass(s, i)
			if err != nil {
				return nil, fmt.Errorf("%w %q: %w", ErrBadGlob, pattern, err)
			}
			b.WriteString(class)
			i = end
		default:
			b.WriteString(regexp.QuoteMeta(string(s[i])))
		}
	}
	return regexp.Compile(b.String() + "$")
}

func globClass(s []rune, start int) (int, string, error) {
	var b strings.Builder
	b.WriteByte('[')
	i := start + 1
	if i < len(s) && s[i] == '^' {
		b.WriteByte('^')
		i++
	}
	first := i
	for i < len(s) && s[i] != ']' {
		next, item, err := classItem(s, i)
		if err != nil {
			return 0, "", err
		}
		b.WriteString(item)
		i = next
	}
	if i >= len(s) {
		return 0, "", errors.New("unterminated [")
	}
	if i == first {
		return 0, "", errors.New("empty character class")
	}
	b.WriteByte(']')
	return i, b.String(), nil
}

func classItem(s []rune, i int) (int, string, error) {
	lo := s[i]
	if i+2 >= len(s) || s[i+1] != '-' || s[i+2] == ']' {
		return i + 1, fmt.Sprintf(`\x{%x}`, lo), nil
	}
	hi := s[i+2]
	if hi < lo {
		return 0, "", fmt.Errorf("range %c-%c is reversed", lo, hi)
	}
	return i + 3, fmt.Sprintf(`\x{%x}-\x{%x}`, lo, hi), nil
}

// MissingSnapshot stats every project path once. Stat failures other than not-exist are returned.
func MissingSnapshot(projects []Project, stat func(string) (fs.FileInfo, error)) (func(Project) bool, error) {
	missing := map[string]bool{}
	for _, p := range projects {
		_, err := stat(p.Path)
		switch {
		case err == nil:
		case errors.Is(err, fs.ErrNotExist):
			missing[p.ID] = true
		default:
			return nil, fmt.Errorf("checking %s: %w", p.Path, err)
		}
	}
	return func(p Project) bool { return missing[p.ID] }, nil
}
