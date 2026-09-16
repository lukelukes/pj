package main

import (
	"cmp"
	"pj/internal/catalog"
	"slices"
)

type Selector struct {
	Filters []string `name:"filter" short:"f" sep:"none" help:"Match field op value (name, path, editor, desc; =, !=, ~, !~). Repeat to AND."`
}

func (s Selector) Filter() (catalog.Filter, error) {
	filters := make([]catalog.Filter, 0, len(s.Filters))
	for _, raw := range s.Filters {
		term, err := catalog.ParseTerm(raw)
		if err != nil {
			return nil, err
		}
		filters = append(filters, term.Filter())
	}
	return catalog.And(filters...), nil
}

func (s Selector) Select(cat catalog.Catalog) ([]catalog.Project, error) {
	f, err := s.Filter()
	if err != nil {
		return nil, err
	}
	projects := catalog.Apply(cat.List(), f)
	sortProjects(projects)
	return projects, nil
}

func sortProjects(projects []catalog.Project) {
	slices.SortFunc(projects, func(a, b catalog.Project) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID))
	})
}
