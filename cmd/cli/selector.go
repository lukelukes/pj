package main

import "pj/internal/catalog"

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
		f, err := term.Compile()
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return catalog.And(filters...), nil
}

func (s Selector) Select(cat catalog.Catalog) ([]catalog.Project, error) {
	f, err := s.Filter()
	if err != nil {
		return nil, err
	}
	return catalog.Apply(cat.List(), f), nil
}
