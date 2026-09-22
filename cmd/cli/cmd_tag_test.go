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
	cmd := TagDetectCmd{}
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

func TestTagDetectSelector(t *testing.T) {
	g, out, cat, _ := tagFixture(t)
	cmd := TagDetectCmd{Selector: Selector{Filters: []string{"name=alpha"}, Tags: []string{"cli"}}, Apply: true}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "  alpha  +lang:go\nTagged 1 projects.\n", out.String())
	for _, p := range cat.List() {
		require.Equal(t, p.Name == "alpha", p.HasTag("lang:go"))
	}
	out.Reset()
	cmd.Filters = []string{"name=nothing"}
	require.NoError(t, cmd.Run(g))
	require.Equal(t, "Nothing to do.\n", out.String())
	cmd.Filters = []string{"bogus=1"}
	require.ErrorIs(t, cmd.Run(g), catalog.ErrUnknownField)
	require.Equal(t, 1, cat.saves)
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
			require.ErrorIs(t, (&TagDetectCmd{Apply: true}).Run(g), boom)
			require.Empty(t, out.String(), "failed writes must not report success")
		})
	}
}
