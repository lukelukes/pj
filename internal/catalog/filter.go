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

// Term is a field, operator and verbatim value. Use ParseTerm to validate input.
type Term struct{ Field, Op, Value string }

func (t Term) String() string { return t.Field + t.Op + t.Value }

// ParseTerm parses field op value, preserving everything after the operator.
func ParseTerm(raw string) (Term, error) {
	i := 0
	for i < len(raw) && raw[i] >= 'a' && raw[i] <= 'z' {
		i++
	}
	field := raw[:i]
	if _, ok := fields[field]; !ok {
		return Term{}, fmt.Errorf("%w %q in %q (valid fields: %s)", ErrUnknownField, field, raw, strings.Join(FilterFields(), ", "))
	}
	for _, op := range []string{"!=", "!~", "=", "~"} {
		if strings.HasPrefix(raw[i:], op) {
			term := Term{Field: field, Op: op, Value: raw[i+len(op):]}
			if err := term.validateGlob(); err != nil {
				return Term{}, err
			}
			return term, nil
		}
	}
	return Term{}, fmt.Errorf("%w %q in %q (valid operators: =, !=, ~, !~)", ErrUnknownOp, raw[i:], raw)
}

func (t Term) validateGlob() error {
	if (t.Op == "=" || t.Op == "!=") && strings.ContainsAny(t.Value, "*?[") {
		if _, err := path.Match(t.Value, ""); err != nil {
			return fmt.Errorf("%w %q: %v", ErrBadGlob, t.Value, err)
		}
	}
	return nil
}

// Filter compiles a valid term. Negation complements the whole field match.
func (t Term) Filter() Filter {
	match := t.matcher()
	f := func(p Project) bool { return slices.ContainsFunc(fields[t.Field](p), match) }
	if t.Op == "!=" || t.Op == "!~" {
		return Not(f)
	}
	return f
}

func (t Term) matcher() func(string) bool {
	if t.Op == "~" || t.Op == "!~" {
		value := strings.ToLower(t.Value)
		return func(s string) bool { return strings.Contains(strings.ToLower(s), value) }
	}
	if strings.ContainsAny(t.Value, "*?[") {
		return func(s string) bool { matched, _ := path.Match(t.Value, s); return matched }
	}
	return func(s string) bool { return s == t.Value }
}
