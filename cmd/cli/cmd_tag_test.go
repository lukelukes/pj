package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/require"
)

type tagCatalog struct {
	catalog.Catalog
	saves, updates     int
	saveErr, updateErr error
}

func (c *tagCatalog) Save() error {
	c.saves++
	if c.saveErr != nil {
		return c.saveErr
	}
	return c.Catalog.Save()
}

func (c *tagCatalog) Update(p catalog.Project) error {
	c.updates++
	if c.updateErr != nil {
		return c.updateErr
	}
	return c.Catalog.Update(p)
}

func tagFixture(t *testing.T) (*Globals, *bytes.Buffer, *tagCatalog, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	cat, err := catalog.NewYAMLCatalog(path)
	require.NoError(t, err)
	for _, name := range []string{"alpha", "beta", "missing"} {
		projectDir := filepath.Join(dir, name)
		require.NoError(t, os.Mkdir(projectDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), nil, 0o600))
		require.NoError(t, cat.Add(catalog.NewProject(name, projectDir).WithTags([]string{"cli"})))
		if name == "missing" {
			require.NoError(t, os.RemoveAll(projectDir))
		}
	}
	require.NoError(t, cat.Save())
	wrapped := &tagCatalog{Catalog: cat}
	var out bytes.Buffer
	return &Globals{Cat: wrapped, Out: &out}, &out, wrapped, path
}

func TestTagDetectDryRunAndApply(t *testing.T) {
	g, out, cat, path := tagFixture(t)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	cmd := TagDetectCmd{All: true}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "  alpha  +lang:go\n  beta  +lang:go\n2 projects would change. Re-run with --apply to write.\n", out.String())
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Zero(t, cat.saves)
	require.Zero(t, cat.updates)
	for _, p := range cat.List() {
		require.Equal(t, []string{"cli"}, p.Tags)
	}
	out.Reset()
	cmd.Apply = true
	require.NoError(t, cmd.Run(g))
	require.Equal(t, 1, cat.saves)
	require.Equal(t, 2, cat.updates)
	require.Contains(t, out.String(), "Tagged 2 projects.")
	require.NoError(t, cat.Load())
	for _, p := range cat.List() {
		require.Equal(t, p.Name != "missing", p.HasTag("lang:go"))
		require.True(t, p.HasTag("cli"))
	}
	out.Reset()
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "Nothing to do.\n", out.String())
	require.Equal(t, 1, cat.saves)
}

func TestTagDetectTargets(t *testing.T) {
	g, out, cat, _ := tagFixture(t)
	cmd := TagDetectCmd{Projects: []string{"alpha", "alpha"}, Apply: true}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "  alpha  +lang:go\nTagged 1 projects.\n", out.String())
	for _, p := range cat.List() {
		require.Equal(t, p.Name == "alpha", p.HasTag("lang:go"))
	}
	out.Reset()
	cmd.Projects = []string{"missing"}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "Nothing to do.\n", out.String())
	for _, tc := range []struct {
		cmd  TagDetectCmd
		want string
	}{
		{TagDetectCmd{Apply: true}, "--all"},
		{TagDetectCmd{Projects: []string{"beta"}, All: true, Apply: true}, "cannot be combined"},
		{TagDetectCmd{Projects: []string{"beta", "nothing"}, Apply: true}, "nothing"},
	} {
		out.Reset()
		require.ErrorContains(t, tc.cmd.Run(g), tc.want)
		require.Empty(t, out.String())
	}
	out.Reset()
	require.NoError(t, (&TagDetectCmd{Projects: []string{"beta", "a"}, Apply: true}).Run(g))
	require.Contains(t, out.String(), "Multiple projects match")
	require.Equal(t, 1, cat.saves)
	require.NoError(t, cat.Load())
	for _, p := range cat.List() {
		require.Equal(t, p.Name == "alpha", p.HasTag("lang:go"))
	}
}

func TestTagDetectErrors(t *testing.T) {
	boom := errors.New("storage failed")
	for _, stage := range []string{"update", "save"} {
		t.Run(stage, func(t *testing.T) {
			g, out, cat, _ := tagFixture(t)
			if stage == "update" {
				cat.updateErr = boom
			} else {
				cat.saveErr = boom
			}
			require.ErrorIs(t, (&TagDetectCmd{All: true, Apply: true}).Run(g), boom)
			require.Empty(t, out.String(), "failed writes must not report success")
		})
	}
}
