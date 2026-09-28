package main

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func TestRmMultiple(t *testing.T) {
	g, out := newTestGlobals(t)
	alpha := createTestProject(t, g, "alpha")
	createTestProject(t, g, "beta")
	createTestProject(t, g, "gamma")
	createTestProject(t, g, "gamma-2")

	out.Reset()
	require.ErrorContains(t, (&RmCmd{Projects: []string{"alpha", "nonexistent"}}).Run(g), "nonexistent")
	require.Equal(t, 4, g.Cat.Count())

	require.NoError(t, (&RmCmd{Projects: []string{"alpha", "gam"}}).Run(g))
	require.Equal(t, 4, g.Cat.Count())
	require.Contains(t, out.String(), "Multiple projects match")

	out.Reset()
	require.NoError(t, (&RmCmd{Projects: []string{alpha, "beta", "alpha"}}).Run(g))
	require.Equal(t, "Removed: alpha\nRemoved: beta\n", out.String())
	require.NoError(t, g.Cat.Load())
	require.Equal(t, []string{"gamma", "gamma-2"}, selectNames(t, g, "gam"))

	var cli struct {
		Rm RmCmd `cmd:""`
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"rm"})
	require.Error(t, err)
}
