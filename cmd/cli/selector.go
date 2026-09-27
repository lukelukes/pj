package main

import (
	"os"
	"pj/internal/catalog"
)

type Selector struct {
	Filters []string `name:"filter" short:"f" sep:"none" placeholder:"QUERY" help:"Select projects with a query, e.g. 'api path:~/work/* NOT is:missing sort:opened'. Qualifiers: name, desc, editor, path, tag, no, is, sort. Repeat to AND. Start with NOT, or use --filter=-x, to negate a lone term."`
}

func (s Selector) Query() (catalog.Query, error) {
	queries := make([]catalog.Query, len(s.Filters))
	for i, raw := range s.Filters {
		q, err := catalog.ParseQuery(raw)
		if err != nil {
			return catalog.Query{}, err
		}
		queries[i] = q
	}
	return catalog.Merge(queries...)
}

func (s Selector) Select(cat catalog.Catalog) ([]catalog.Project, *catalog.Sort, error) {
	q, err := s.Query()
	if err != nil {
		return nil, nil, err
	}
	projects := cat.List()
	env := catalog.QueryEnv{}
	env.Home, _ = os.UserHomeDir()
	if q.Uses("is") {
		if env.Missing, err = catalog.MissingSnapshot(projects, os.Stat); err != nil {
			return nil, nil, err
		}
	}
	f, err := q.Compile(env)
	if err != nil {
		return nil, nil, err
	}
	return catalog.Apply(projects, f), q.Sort, nil
}
