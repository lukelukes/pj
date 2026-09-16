package main

import (
	"errors"
	"pj/internal/catalog"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestListParsing(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		filters []string
		output  Output
		names   bool
	}{
		{[]string{"list"}, nil, OutputTable, false},
		{[]string{"list", "-f", "name=a,b", "--filter", "editor=", "-o", "paths"}, []string{"name=a,b", "editor="}, OutputPaths, false},
		{[]string{"list", "-n"}, nil, OutputTable, true},
		{[]string{"list", "--names", "--output", "json"}, nil, OutputJSON, true},
	} {
		var cli struct {
			List ListCmd `cmd:""`
		}
		parser, err := kong.New(&cli)
		require.NoError(t, err)
		_, err = parser.Parse(tc.args)
		require.NoError(t, err)
		require.Equal(t, tc.filters, cli.List.Filters)
		require.Equal(t, tc.output, cli.List.Output)
		require.Equal(t, tc.names, cli.List.Names)
	}
	var cli struct {
		List ListCmd `cmd:""`
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"list", "-o", "bogus"})
	require.ErrorContains(t, err, "bogus")
}

func TestListSelection(t *testing.T) {
	g, out := newTestGlobals(t)
	createTestProject(t, g, "beta")
	alpha := createTestProject(t, g, "alpha")
	out.Reset()
	cmd := ListCmd{Selector: Selector{Filters: []string{"name~ALP", "editor="}}, Output: OutputPaths}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, alpha+"\n", out.String())
	out.Reset()
	cmd.Names = true
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "alpha\n", out.String())
	out.Reset()
	cmd.Filters = []string{"bogus=1"}
	err := cmd.Run(g)
	require.ErrorIs(t, err, catalog.ErrUnknownField)
	require.ErrorContains(t, err, "desc, editor, name, path")
	require.Empty(t, out.String())
	cmd.Filters = nil
	cmd.Names = false
	cmd.Output = "bogus"
	require.ErrorContains(t, cmd.Run(g), "unknown output")
}

func TestEditDescription(t *testing.T) {
	g, _ := newTestGlobals(t)
	createTestProject(t, g, "alpha")
	var cli struct {
		Edit EditCmd `cmd:""`
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"edit", "alpha", "--desc", "API service"})
	require.NoError(t, err)
	require.NoError(t, cli.Edit.Run(g))
	require.NoError(t, g.Cat.Load())
	require.Equal(t, "API service", g.Cat.List()[0].Description)
	selected, err := (Selector{Filters: []string{"desc~api"}}).Select(g.Cat)
	require.NoError(t, err)
	require.Len(t, selected, 1)
}

type failingWriter struct{}

var errOutput = errors.New("output failed")

func (failingWriter) Write([]byte) (int, error) { return 0, errOutput }

func TestProjectionWriteError(t *testing.T) {
	for _, output := range []Output{OutputNames, OutputPaths, OutputJSON} {
		require.ErrorIs(t, printProjects(failingWriter{}, []catalog.Project{{Name: "alpha"}}, output), errOutput)
	}
}
