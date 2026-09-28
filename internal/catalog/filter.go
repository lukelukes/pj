package catalog

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
