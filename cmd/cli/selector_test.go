package main

import (
	"errors"
	"os"
	"pj/internal/catalog"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func parseList(t *testing.T, args ...string) (ListCmd, error) {
	t.Helper()
	var cli struct {
		List ListCmd `cmd:""`
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	_, err = parser.Parse(append([]string{"list"}, args...))
	return cli.List, err
}

func TestListParsing(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		filters []string
		output  Output
		names   bool
	}{
		{nil, nil, OutputTable, false},
		{[]string{"-f", "tag:a,b", "--filter", "no:editor", "-o", "paths"}, []string{"tag:a,b", "no:editor"}, OutputPaths, false},
		{[]string{"--filter=-editor:vim"}, []string{"-editor:vim"}, OutputTable, false},
		{[]string{"-f", "api -editor:vim"}, []string{"api -editor:vim"}, OutputTable, false},
		{[]string{"-f", "NOT editor:vim"}, []string{"NOT editor:vim"}, OutputTable, false},
		{[]string{"-n"}, nil, OutputTable, true},
		{[]string{"--names", "--output", "json"}, nil, OutputJSON, true},
	} {
		cmd, err := parseList(t, tc.args...)
		require.NoError(t, err)
		require.Equal(t, tc.filters, cmd.Filters)
		require.Equal(t, tc.output, cmd.Output)
		require.Equal(t, tc.names, cmd.Names)
	}
	for _, args := range [][]string{{"-o", "bogus"}, {"-f", "-editor:vim"}, {"--filter", "-editor:vim"}} {
		_, err := parseList(t, args...)
		require.Error(t, err, args)
	}
}

func selectNames(t *testing.T, g *Globals, filters ...string) []string {
	t.Helper()
	projects, _, err := Selector{Filters: filters}.Select(g.Cat)
	require.NoError(t, err)
	names := []string{}
	for _, p := range byNameOrder(projects) {
		names = append(names, p.Name)
	}
	return names
}

func byNameOrder(projects []catalog.Project) []catalog.Project {
	return sortProjects(projects, catalog.Sort{Key: catalog.SortName}, nil)
}

func TestListSelection(t *testing.T) {
	g, out := newTestGlobals(t)
	createTestProject(t, g, "beta")
	alpha := createTestProject(t, g, "alpha")
	gone := createTestProject(t, g, "gone")
	require.NoError(t, os.Remove(gone))

	require.Equal(t, []string{"alpha"}, selectNames(t, g, "ALP"))
	require.Equal(t, []string{"alpha", "beta"}, selectNames(t, g, "name:alpha,beta"))
	require.Equal(t, []string{"alpha", "beta", "gone"}, selectNames(t, g, "no:editor"))
	require.Equal(t, []string{"gone"}, selectNames(t, g, "is:missing"))
	require.Equal(t, []string{"alpha"}, selectNames(t, g, "NOT is:missing", "-beta"))
	require.Equal(t, []string{"alpha"}, selectNames(t, g, "path:"+alpha))

	out.Reset()
	cmd := ListCmd{Selector: Selector{Filters: []string{"alp", "no:editor"}}, Output: OutputPaths}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, alpha+"\n", out.String())
	out.Reset()
	cmd.Names = true
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "alpha\n", out.String())

	for _, tc := range []struct {
		filters []string
		err     error
	}{
		{[]string{"bogus:1"}, catalog.ErrUnknownQualifier},
		{[]string{""}, catalog.ErrQuerySyntax},
		{[]string{"sort:name", "sort:added"}, catalog.ErrBadSort},
	} {
		out.Reset()
		cmd := ListCmd{Selector: Selector{Filters: tc.filters}, Output: OutputNames}
		require.ErrorIs(t, cmd.Run(g), tc.err)
		require.Empty(t, out.String())
	}
	cmd = ListCmd{Output: "bogus"}
	require.ErrorContains(t, cmd.Run(g), "unknown output")
}

func TestListSort(t *testing.T) {
	g, out := newTestGlobals(t)
	paths := map[string]string{}
	for _, name := range []string{"b", "a", "c"} {
		paths[name] = createTestProject(t, g, name)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, offset := range map[string]int{"a": 2, "b": 3, "c": 1} {
		p, err := g.Cat.GetByPath(paths[name])
		require.NoError(t, err)
		p.LastAccessed = base.Add(time.Duration(offset) * time.Hour)
		p.AddedAt = base.Add(-time.Duration(offset) * time.Hour)
		require.NoError(t, g.Cat.Update(p))
		require.NoError(t, os.Chtimes(paths[name], base, base.Add(time.Duration(offset)*time.Minute)))
	}
	for _, tc := range []struct {
		filter string
		want   string
	}{
		{"", "a\nb\nc\n"},
		{"sort:name", "a\nb\nc\n"},
		{"sort:name-desc", "c\nb\na\n"},
		{"sort:opened", "b\na\nc\n"},
		{"sort:opened-asc", "c\na\nb\n"},
		{"sort:added", "c\na\nb\n"},
		{"sort:modified", "b\na\nc\n"},
		{"sort:modified-asc", "c\na\nb\n"},
	} {
		out.Reset()
		cmd := ListCmd{Output: OutputNames}
		if tc.filter != "" {
			cmd.Filters = []string{tc.filter}
		}
		require.NoError(t, cmd.Run(g))
		require.Equal(t, tc.want, out.String(), tc.filter)
	}
	out.Reset()
	cmd := ListCmd{Selector: Selector{Filters: []string{"sort:name"}}, Output: OutputTable}
	require.NoError(t, cmd.Run(g))
	table := out.String()
	require.Less(t, strings.Index(table, paths["a"]), strings.Index(table, paths["c"]))
	out.Reset()
	cmd.Filters = nil
	require.NoError(t, cmd.Run(g))
	table = out.String()
	require.Less(t, strings.Index(table, paths["b"]), strings.Index(table, paths["c"]))
}

func TestEditDescription(t *testing.T) {
	g, _ := newTestGlobals(t)
	createTestProject(t, g, "alpha")
	edit := func(args ...string) string {
		var cli struct {
			Edit EditCmd `cmd:""`
		}
		parser, err := kong.New(&cli)
		require.NoError(t, err)
		_, err = parser.Parse(append([]string{"edit", "alpha"}, args...))
		require.NoError(t, err)
		require.NoError(t, cli.Edit.Run(g))
		require.NoError(t, g.Cat.Load())
		return g.Cat.List()[0].Description
	}
	require.Equal(t, "API service", edit("--desc", "API service"))
	require.Equal(t, []string{"alpha"}, selectNames(t, g, "desc:api"))
	require.Equal(t, "API service", edit("--editor", "nvim"))
	require.Empty(t, edit("--desc", ""))
	require.Empty(t, edit("--desc="))
}

type failingWriter struct{}

var errOutput = errors.New("output failed")

func (failingWriter) Write([]byte) (int, error) { return 0, errOutput }

func TestProjectionWriteError(t *testing.T) {
	g, _ := newTestGlobals(t)
	for _, output := range []Output{OutputTable, OutputNames, OutputPaths, OutputJSON} {
		require.ErrorIs(t, printProjects(failingWriter{}, g.Render, []catalog.Project{{Name: "alpha"}}, output, nil), errOutput)
	}
}
