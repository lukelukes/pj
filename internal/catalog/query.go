package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

var (
	ErrQuerySyntax      = errors.New("invalid query")
	ErrUnknownQualifier = errors.New("unknown qualifier")
	ErrBadValue         = errors.New("invalid qualifier value")
	ErrBadGlob          = errors.New("invalid glob")
	ErrBadSort          = errors.New("invalid sort")
)

// Value is one qualifier or text value. Quoted values are always literal.
type Value struct {
	Text   string
	Quoted bool
}

// Term is one whitespace-separated piece of a query. An empty Qualifier makes it a text term.
// Values are alternatives: the term matches when any value matches.
type Term struct {
	Negated   bool
	Qualifier string
	Values    []Value
}

// SortKey names a project ordering.
type SortKey string

const (
	SortName     SortKey = "name"
	SortOpened   SortKey = "opened"
	SortAdded    SortKey = "added"
	SortModified SortKey = "modified"
)

// Sort is a requested project ordering.
type Sort struct {
	Key  SortKey
	Desc bool
}

// Query selects projects matching every term, in the order given by Sort when set.
type Query struct {
	Terms []Term
	Sort  *Sort
}

const qualifierSort = "sort"

var sortKeys = []SortKey{SortName, SortOpened, SortAdded, SortModified}

// Qualifiers returns the supported qualifier names in sorted order.
func Qualifiers() []string {
	names := []string{qualifierSort}
	for name := range qualifiers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// SortValues returns every accepted sort: value.
func SortValues() []string {
	var values []string
	for _, key := range sortKeys {
		values = append(values, string(key), string(key)+"-asc", string(key)+"-desc")
	}
	return values
}

// ParseQuery parses one --filter value.
func ParseQuery(raw string) (Query, error) {
	q, err := parseQuery(raw)
	if err != nil {
		return Query{}, fmt.Errorf("query %q: %w", raw, err)
	}
	return q, nil
}

// Merge ANDs queries together. At most one of them may carry a sort.
func Merge(queries ...Query) (Query, error) {
	var merged Query
	for _, q := range queries {
		merged.Terms = append(merged.Terms, q.Terms...)
		if q.Sort != nil {
			if merged.Sort != nil {
				return Query{}, fmt.Errorf("%w: only one sort: is allowed", ErrBadSort)
			}
			merged.Sort = q.Sort
		}
	}
	return merged, nil
}

// Uses reports whether any term has the given qualifier.
func (q Query) Uses(qualifier string) bool {
	return slices.ContainsFunc(q.Terms, func(t Term) bool { return t.Qualifier == qualifier })
}

// String renders a parsed query in canonical form; parsing it yields the same query.
func (q Query) String() string {
	parts := make([]string, 0, len(q.Terms)+1)
	for _, t := range q.Terms {
		parts = append(parts, t.String())
	}
	if q.Sort != nil {
		parts = append(parts, q.Sort.String())
	}
	return strings.Join(parts, " ")
}

func (t Term) String() string {
	var b strings.Builder
	if t.Negated {
		b.WriteByte('-')
	}
	if t.Qualifier != "" {
		b.WriteString(t.Qualifier)
		b.WriteByte(':')
	}
	for i, v := range t.Values {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(v.String())
	}
	return b.String()
}

func (v Value) String() string {
	if !v.Quoted {
		return v.Text
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v.Text) + `"`
}

func (s Sort) String() string {
	dir := "-asc"
	if s.Desc {
		dir = "-desc"
	}
	return qualifierSort + ":" + string(s.Key) + dir
}

func parseSort(t Term) (*Sort, error) {
	if t.Negated {
		return nil, fmt.Errorf("%w: sort: cannot be negated", ErrBadSort)
	}
	if len(t.Values) != 1 {
		return nil, fmt.Errorf("%w: sort: takes a single value", ErrBadSort)
	}
	value := t.Values[0].Text
	key, dir, _ := strings.Cut(value, "-")
	s := Sort{Key: SortKey(key), Desc: key != string(SortName)}
	switch dir {
	case "":
	case "asc":
		s.Desc = false
	case "desc":
		s.Desc = true
	default:
		s.Key = ""
	}
	if !slices.Contains(sortKeys, s.Key) || strings.HasSuffix(value, "-") {
		return nil, fmt.Errorf("%w %q (valid sorts: %s)", ErrBadSort, value, strings.Join(SortValues(), ", "))
	}
	return &s, nil
}

type lexer struct {
	s []rune
	i int
}

func parseQuery(raw string) (Query, error) {
	l := &lexer{s: []rune(raw)}
	var q Query
	for {
		l.skipSpace()
		if l.eof() {
			break
		}
		t, err := l.term()
		if err != nil {
			return Query{}, err
		}
		if t.Qualifier != qualifierSort {
			q.Terms = append(q.Terms, t)
			continue
		}
		if q.Sort != nil {
			return Query{}, fmt.Errorf("%w: only one sort: is allowed", ErrBadSort)
		}
		if q.Sort, err = parseSort(t); err != nil {
			return Query{}, err
		}
	}
	if len(q.Terms) == 0 && q.Sort == nil {
		return Query{}, fmt.Errorf("%w: empty query", ErrQuerySyntax)
	}
	return q, nil
}

func (l *lexer) eof() bool { return l.i >= len(l.s) }

func (l *lexer) atBoundary(i int) bool { return i >= len(l.s) || unicode.IsSpace(l.s[i]) }

func (l *lexer) skipSpace() {
	for !l.eof() && unicode.IsSpace(l.s[l.i]) {
		l.i++
	}
}

func (l *lexer) atNOT() bool {
	return l.i+3 <= len(l.s) && string(l.s[l.i:l.i+3]) == "NOT" && l.atBoundary(l.i+3)
}

func (l *lexer) term() (Term, error) {
	negated := false
	switch {
	case l.s[l.i] == '-':
		l.i++
		negated = true
		if l.atBoundary(l.i) {
			return Term{}, fmt.Errorf("%w: '-' must be followed by a term", ErrQuerySyntax)
		}
	case l.atNOT():
		l.i += 3
		l.skipSpace()
		negated = true
		if l.eof() {
			return Term{}, fmt.Errorf("%w: NOT must be followed by a term", ErrQuerySyntax)
		}
	}
	if negated && (l.s[l.i] == '-' || l.atNOT()) {
		return Term{}, fmt.Errorf("%w: a term can only be negated once", ErrQuerySyntax)
	}
	t, err := l.atom()
	t.Negated = negated
	return t, err
}

func (l *lexer) qualifierEnd() (int, bool) {
	j := l.i
	for j < len(l.s) && (l.s[j] >= 'a' && l.s[j] <= 'z' || l.s[j] >= 'A' && l.s[j] <= 'Z') {
		j++
	}
	return j, j > l.i && j < len(l.s) && l.s[j] == ':'
}

func (l *lexer) atom() (Term, error) {
	if j, ok := l.qualifierEnd(); ok {
		name := string(l.s[l.i:j])
		if _, known := qualifiers[name]; !known && name != qualifierSort {
			return Term{}, fmt.Errorf("%w %q (valid qualifiers: %s)", ErrUnknownQualifier, name, strings.Join(Qualifiers(), ", "))
		}
		l.i = j + 1
		values, err := l.valueList(name)
		return Term{Qualifier: name, Values: values}, err
	}
	v, err := l.value(false)
	if err != nil {
		return Term{}, err
	}
	if !l.atBoundary(l.i) {
		return Term{}, fmt.Errorf("%w: unexpected %q after closing quote", ErrQuerySyntax, l.s[l.i])
	}
	return Term{Values: []Value{v}}, nil
}

func (l *lexer) valueList(qualifier string) ([]Value, error) {
	var values []Value
	for {
		if l.atBoundary(l.i) {
			return nil, fmt.Errorf("%w: empty value for qualifier %q", ErrQuerySyntax, qualifier)
		}
		v, err := l.value(true)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		if l.atBoundary(l.i) {
			return values, nil
		}
		if l.s[l.i] != ',' {
			return nil, fmt.Errorf("%w: unexpected %q after closing quote", ErrQuerySyntax, l.s[l.i])
		}
		l.i++
	}
}

func (l *lexer) value(inList bool) (Value, error) {
	if l.s[l.i] == '"' {
		return l.quoted()
	}
	start := l.i
	for !l.atBoundary(l.i) && !(inList && l.s[l.i] == ',') {
		if l.s[l.i] == '"' {
			return Value{}, fmt.Errorf("%w: quote inside unquoted value", ErrQuerySyntax)
		}
		l.i++
	}
	if l.i == start {
		return Value{}, fmt.Errorf("%w: empty value in list", ErrQuerySyntax)
	}
	return Value{Text: string(l.s[start:l.i])}, nil
}

func (l *lexer) quoted() (Value, error) {
	l.i++
	var b strings.Builder
	for {
		if l.eof() {
			return Value{}, fmt.Errorf("%w: unterminated quote", ErrQuerySyntax)
		}
		c := l.s[l.i]
		if c == '\\' && l.i+1 < len(l.s) && (l.s[l.i+1] == '"' || l.s[l.i+1] == '\\') {
			b.WriteRune(l.s[l.i+1])
			l.i += 2
			continue
		}
		l.i++
		if c == '"' {
			break
		}
		b.WriteRune(c)
	}
	if b.Len() == 0 {
		return Value{}, fmt.Errorf("%w: empty quoted value", ErrQuerySyntax)
	}
	return Value{Text: b.String(), Quoted: true}, nil
}
