package main

import (
	"cmp"
	"pj/internal/catalog"
	"slices"
)

type Selector struct {
	Tags    []string `name:"tag" short:"t" help:"Match a tag (supports globs). Repeat to AND."`
	Filters []string `name:"filter" short:"f" sep:"none" help:"Match field op value (name, path, editor, desc, tag; =, !=, ~, !~). Repeat to AND."`
}

func (s Selector) Filter() (catalog.Filter, error) {
	filters := make([]catalog.Filter, 0, len(s.Filters))
	terms := slices.Clone(s.Filters)
	for _, tag := range s.Tags {
		terms = append(terms, "tag="+tag)
	}
	for _, raw := range terms {
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
