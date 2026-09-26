package catalog

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
)

var (
	ErrUnknownField = errors.New("unknown filter field")
	ErrUnknownOp    = errors.New("unknown filter operator")
	ErrBadGlob      = errors.New("invalid filter glob")
)

// Filter is a predicate over a project.
type Filter func(Project) bool

// And matches when every filter matches, including when no filters are supplied.
func And(filters ...Filter) Filter {
	return func(p Project) bool {
		for _, f := range filters {
			if !f(p) {
				return false
			}
		}
		return true
	}
}

// Or matches when any filter matches. With no filters it matches nothing.
func Or(filters ...Filter) Filter {
	return func(p Project) bool {
		for _, f := range filters {
			if f(p) {
				return true
			}
		}
		return false
	}
}

// Not complements a filter.
func Not(f Filter) Filter { return func(p Project) bool { return !f(p) } }

// Apply returns matching projects in input order without modifying the input.
func Apply(projects []Project, f Filter) []Project {
	result := make([]Project, 0, len(projects))
	for _, p := range projects {
		if f(p) {
			result = append(result, p)
		}
	}
	return result
}

var fields = map[string]func(Project) []string{
	"name":   func(p Project) []string { return []string{p.Name} },
	"path":   func(p Project) []string { return []string{p.Path} },
	"editor": func(p Project) []string { return []string{p.Editor} },
	"desc":   func(p Project) []string { return []string{p.Description} },
}

// FilterFields returns the supported field names in sorted order.
func FilterFields() []string { return slices.Sorted(maps.Keys(fields)) }

// Op is a filter operator.
type Op string

const (
	OpEqual       Op = "="
	OpNotEqual    Op = "!="
	OpContains    Op = "~"
	OpNotContains Op = "!~"
)

type opSpec struct {
	op                 Op
	negated, substring bool
}

var ops = []opSpec{
	{OpEqual, false, false},
	{OpNotEqual, true, false},
	{OpContains, false, true},
	{OpNotContains, true, true},
}

// FilterOps returns the supported operators.
func FilterOps() []Op {
	result := make([]Op, len(ops))
	for i, o := range ops {
		result[i] = o.op
	}
	return result
}

func joinOps() string {
	names := make([]string, len(ops))
	for i, o := range ops {
		names[i] = string(o.op)
	}
	return strings.Join(names, ", ")
}

// Term is a field, operator and verbatim value. Compile validates it.
type Term struct {
	Field string
	Op    Op
	Value string
}

func (t Term) String() string { return t.Field + string(t.Op) + t.Value }

// ParseTerm parses field op value, preserving everything after the operator.
func ParseTerm(raw string) (Term, error) {
	i := 0
	for i < len(raw) && raw[i] >= 'a' && raw[i] <= 'z' {
		i++
	}
	rest := raw[i:]
	op := Op(rest)
	for _, o := range ops {
		if strings.HasPrefix(rest, string(o.op)) {
			op = o.op
			break
		}
	}
	term := Term{Field: raw[:i], Op: op, Value: rest[len(op):]}
	if _, err := term.Compile(); err != nil {
		return Term{}, fmt.Errorf("%w in %q", err, raw)
	}
	return term, nil
}

// Compile validates the term and returns its filter. Negation complements the whole field match.
func (t Term) Compile() (Filter, error) {
	get, ok := fields[t.Field]
	if !ok {
		return nil, fmt.Errorf("%w %q (valid fields: %s)", ErrUnknownField, t.Field, strings.Join(FilterFields(), ", "))
	}
	i := slices.IndexFunc(ops, func(o opSpec) bool { return o.op == t.Op })
	if i < 0 {
		return nil, fmt.Errorf("%w %q (valid operators: %s)", ErrUnknownOp, t.Op, joinOps())
	}
	match, err := matcher(t.Value, ops[i].substring)
	if err != nil {
		return nil, err
	}
	f := func(p Project) bool { return slices.ContainsFunc(get(p), match) }
	if ops[i].negated {
		return Not(f), nil
	}
	return f, nil
}

func matcher(value string, substring bool) (func(string) bool, error) {
	if substring {
		value = strings.ToLower(value)
		return func(s string) bool { return strings.Contains(strings.ToLower(s), value) }, nil
	}
	if !strings.ContainsAny(value, "*?[") {
		return func(s string) bool { return s == value }, nil
	}
	if _, err := path.Match(value, ""); err != nil {
		return nil, fmt.Errorf("%w %q: %v", ErrBadGlob, value, err)
	}
	return func(s string) bool { matched, _ := path.Match(value, s); return matched }, nil
}
