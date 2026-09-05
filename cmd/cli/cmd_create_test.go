package main

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/create"
	"pj/internal/ui"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHome = "/home/luke"

func planFor(t *testing.T, g *Globals, location, name string, git bool) create.Plan {
	t.Helper()
	return create.BuildPlan(create.Request{Name: name, Location: location, Git: git}, create.CatalogEnv{Cat: g.Cat})
}

func TestToPreview(t *testing.T) {
	t.Run("clean plan shows the resolved path and what will happen", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()

		p := planFor(t, g, location, "api", true)
		p.Editor = "nvim"
		preview := toPreview(p, testHome)

		assert.Equal(t, ui.SeverityOK, preview.Severity)
		assert.Equal(t, filepath.Join(location, "api"), preview.Path)
		assert.Equal(t, []string{"git init", "nvim"}, preview.Facts)
		assert.False(t, preview.NeedsAdopt)
	})

	t.Run("home directory is abbreviated", func(t *testing.T) {
		preview := toPreview(create.Plan{Path: testHome + "/dev/api"}, testHome)

		assert.Equal(t, "~/dev/api", preview.Path)
	})

	t.Run("validation failure blocks with the reason", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		preview := toPreview(planFor(t, g, t.TempDir(), "", true), testHome)

		assert.True(t, preview.Blocked())
		assert.Equal(t, create.ErrEmptyName.Error(), preview.Message)
	})

	t.Run("already tracked path blocks and names the owner", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		existing := filepath.Join(location, "dup")
		require.NoError(t, os.Mkdir(existing, 0o755))
		require.NoError(t, g.Cat.Add(catalog.NewProject("dup", existing)))

		preview := toPreview(planFor(t, g, location, "dup", true), testHome)

		assert.True(t, preview.Blocked())
		assert.Contains(t, preview.Message, `already tracked as "dup"`)
		assert.False(t, preview.NeedsAdopt, "a tracked path is not adoptable")
	})

	t.Run("existing directory offers adoption", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		existing := filepath.Join(location, "legacy")
		require.NoError(t, os.Mkdir(existing, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(existing, "a.txt"), nil, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(existing, "b.txt"), nil, 0o644))

		preview := toPreview(planFor(t, g, location, "legacy", true), testHome)

		assert.Equal(t, ui.SeverityNotice, preview.Severity)
		assert.True(t, preview.NeedsAdopt)
		assert.Contains(t, preview.Message, "exists · 2 files")
		assert.NotEmpty(t, preview.AdoptHint)
	})

	t.Run("single file is not pluralized", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		existing := filepath.Join(location, "legacy")
		require.NoError(t, os.Mkdir(existing, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(existing, "a.txt"), nil, 0o644))

		preview := toPreview(planFor(t, g, location, "legacy", true), testHome)

		assert.Contains(t, preview.Message, "exists · 1 file")
	})

	t.Run("duplicate name is a notice, not a block", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		other := t.TempDir()
		require.NoError(t, g.Cat.Add(catalog.NewProject("api", other)))

		preview := toPreview(planFor(t, g, t.TempDir(), "api", true), testHome)

		assert.Equal(t, ui.SeverityNotice, preview.Severity)
		assert.False(t, preview.Blocked())
		assert.Contains(t, preview.Message, "name already used by")
	})

	t.Run("nesting inside a repository is explained", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(location, ".git"), 0o755))

		preview := toPreview(planFor(t, g, location, "sub", true), testHome)

		assert.NotContains(t, preview.Facts, "git init")
		assert.Contains(t, preview.Facts[0], "inside ")
	})
}

func TestPreviewFuncRoundTripsTheDraft(t *testing.T) {
	g, _ := newTestGlobals(t)
	location := t.TempDir()

	preview := previewFunc(create.CatalogEnv{Cat: g.Cat}, testHome)(ui.Draft{
		Name:     "api",
		Location: location,
		Git:      true,
	})

	assert.Equal(t, filepath.Join(location, "api"), preview.Path)
	assert.Contains(t, preview.Facts, "git init")
}

func TestRunPlanCreatesAndReports(t *testing.T) {
	g, out := newTestGlobals(t)
	t.Setenv("__PJ_CD_FILE", "")
	location := t.TempDir()

	require.NoError(t, runPlan(g, planFor(t, g, location, "api", false), false, testHome))

	assert.DirExists(t, filepath.Join(location, "api"))
	assert.Equal(t, 1, g.Cat.Count())
	assert.Contains(t, out.String(), "cd "+filepath.Join(location, "api"))
}

func TestRunPlanWritesCdFile(t *testing.T) {
	g, out := newTestGlobals(t)
	cdFile := filepath.Join(t.TempDir(), "cd-target")
	t.Setenv("__PJ_CD_FILE", cdFile)
	location := t.TempDir()

	require.NoError(t, runPlan(g, planFor(t, g, location, "api", false), false, testHome))

	content, err := os.ReadFile(cdFile)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(location, "api"), string(content))
	assert.NotContains(t, out.String(), "Run: cd", "no manual hint when the shell will jump for us")
}

func TestCreateCmdRequiresAdoptFlag(t *testing.T) {
	g, _ := newTestGlobals(t)
	location := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(location, "legacy"), 0o755))

	cmd := CreateCmd{Name: "legacy", At: location, NoInput: true}
	err := cmd.Run(g)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--adopt")
	assert.Zero(t, g.Cat.Count())
}

func TestCreateCmdAdoptsExistingDirectory(t *testing.T) {
	g, _ := newTestGlobals(t)
	t.Setenv("__PJ_CD_FILE", "")
	location := t.TempDir()
	existing := filepath.Join(location, "legacy")
	require.NoError(t, os.Mkdir(existing, 0o755))

	cmd := CreateCmd{Name: "legacy", At: location, NoInput: true, NoGit: true, Adopt: true}
	require.NoError(t, cmd.Run(g))

	assert.Equal(t, 1, g.Cat.Count())
	assert.Equal(t, existing, g.Cat.List()[0].Path)
}

func TestCreateCmdIsFullyNonInteractive(t *testing.T) {
	g, _ := newTestGlobals(t)
	t.Setenv("__PJ_CD_FILE", "")
	location := t.TempDir()

	cmd := CreateCmd{
		Name:    "api",
		At:      location,
		Desc:    "edge router",
		Editor:  "nvim",
		NoGit:   true,
		NoInput: true,
	}
	require.NoError(t, cmd.Run(g))

	require.Equal(t, 1, g.Cat.Count())
	p := g.Cat.List()[0]
	assert.Equal(t, "api", p.Name)
	assert.Equal(t, "edge router", p.Description)
	assert.Equal(t, "nvim", p.Editor)
	assert.NoDirExists(t, filepath.Join(p.Path, ".git"))
}

func TestCreateCmdNeedsWizard(t *testing.T) {
	assert.False(t, (&CreateCmd{Name: "api"}).needsWizard(), "a supplied name skips the wizard")
	assert.False(t, (&CreateCmd{NoInput: true}).needsWizard(), "--no-input never prompts")
}

func TestPluralFiles(t *testing.T) {
	assert.Equal(t, "0 files", pluralFiles(0))
	assert.Equal(t, "1 file", pluralFiles(1))
	assert.Equal(t, "2 files", pluralFiles(2))
}

func TestDraftRequestRoundTrip(t *testing.T) {
	req := create.Request{
		Name:        "api",
		Location:    "/home/luke/dev",
		Description: "edge router",
		Editor:      "nvim",
		Git:         true,
	}

	assert.Equal(t, req, toRequest(toDraft(req)))
}

func TestResolveLocation(t *testing.T) {
	t.Run("empty falls back to the working directory", func(t *testing.T) {
		got, err := resolveLocation("", "/home/luke/dev")

		require.NoError(t, err)
		assert.Equal(t, "/home/luke/dev", got)
	})

	t.Run("relative paths are made absolute", func(t *testing.T) {
		cwd, err := os.Getwd()
		require.NoError(t, err)

		got, err := resolveLocation(".", cwd)

		require.NoError(t, err)
		assert.Equal(t, cwd, got, `--at . must resolve, not blow up as "location is required"`)
	})

	t.Run("tilde is expanded", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		got, err := resolveLocation("~/dev", "/tmp")

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, "dev"), got)
	})

	t.Run("unsupported tilde form is rejected", func(t *testing.T) {
		_, err := resolveLocation("~someoneelse/dev", "/tmp")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --at path")
	})
}

func TestCreateCmdAcceptsRelativeLocation(t *testing.T) {
	g, _ := newTestGlobals(t)
	t.Setenv("__PJ_CD_FILE", "")
	location := t.TempDir()
	t.Chdir(location)

	cmd := CreateCmd{Name: "api", At: ".", NoInput: true, NoGit: true}
	require.NoError(t, cmd.Run(g))

	require.Equal(t, 1, g.Cat.Count())
	assert.Equal(t, filepath.Join(location, "api"), g.Cat.List()[0].Path)
}

func TestCreateCmdErrorsReadCleanly(t *testing.T) {
	g, _ := newTestGlobals(t)
	location := t.TempDir()
	existing := filepath.Join(location, "dup")
	require.NoError(t, os.Mkdir(existing, 0o755))
	require.NoError(t, g.Cat.Add(catalog.NewProject("dup", existing)))

	err := (&CreateCmd{Name: "dup", At: location, NoInput: true}).Run(g)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `already tracked as "dup"`)
	assert.NotContains(t, err.Error(), "plan cannot be executed",
		"internal error wrapping must not leak into user-facing messages")
}
