package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"pj/cmd/cli/render"
	"pj/internal/catalog"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFixedNow = time.Date(2026, 1, 7, 12, 0, 0, 0, time.Local)

func newTestGlobals(t *testing.T) (*Globals, *bytes.Buffer) {
	t.Helper()
	g, buf, _ := newTestGlobalsCore(t)
	return g, buf
}

func newTestGlobalsCore(t *testing.T) (*Globals, *bytes.Buffer, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	cat, err := catalog.NewYAMLCatalog(filepath.Join(dir, "catalog.yaml"))
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	pathMap := make(map[string]string)
	return &Globals{
		Cat:    cat,
		Out:    buf,
		Render: render.NewLipglossRenderer(buf, 80).WithClock(func() time.Time { return testFixedNow }),
		RunCmd: func(name string, args ...string) error { return nil },
	}, buf, pathMap
}

func createTestProject(t *testing.T, g *Globals, name string) string {
	t.Helper()
	projectDir := t.TempDir()
	cmd := AddCmd{Path: projectDir, Name: name}
	require.NoError(t, cmd.Run(g))
	return projectDir
}

func TestAddCmd_Run(t *testing.T) {
	t.Run("adds project successfully", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd := AddCmd{Path: projectDir, Name: "test-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 1, g.Cat.Count())

		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.Equal(t, "test-project", projects[0].Name)
		assert.Equal(t, projectDir, projects[0].Path)
	})

	t.Run("adds project with default name from directory", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd := AddCmd{Path: projectDir}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.Equal(t, filepath.Base(projectDir), projects[0].Name)
	})

	t.Run("returns error for nonexistent path", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := AddCmd{Path: "/nonexistent/path", Name: "fail-project"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path does not exist")
		assert.Equal(t, 0, g.Cat.Count())
	})

	t.Run("returns error for file instead of directory", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(filePath, []byte("test"), 0o644)
		require.NoError(t, err)

		cmd := AddCmd{Path: filePath, Name: "file-project"}
		err = cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
		assert.Equal(t, 0, g.Cat.Count())
	})

	t.Run("returns error for duplicate path", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd1 := AddCmd{Path: projectDir, Name: "project1"}
		err := cmd1.Run(g)
		require.NoError(t, err)

		cmd2 := AddCmd{Path: projectDir, Name: "project2"}
		err = cmd2.Run(g)

		assert.Error(t, err)
		assert.ErrorIs(t, err, catalog.ErrAlreadyExists)
		assert.Equal(t, 1, g.Cat.Count())
	})

	t.Run("outputs confirmation message", func(t *testing.T) {
		g, out := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd := AddCmd{Path: projectDir, Name: "new-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Added:")
		assert.Contains(t, output, "new-project")
	})
}

func TestListCmd_Run(t *testing.T) {
	t.Run("lists empty catalog", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
	})

	t.Run("lists all projects", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "project1")
		createTestProject(t, g, "project2")

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 2, g.Cat.Count())
	})

	t.Run("output includes project name and path", func(t *testing.T) {
		g, out := newTestGlobals(t)
		projectDir := createTestProject(t, g, "my-project")

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "my-project")
		assert.Contains(t, output, projectDir)
	})

	t.Run("names flag outputs only names", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "alpha")
		createTestProject(t, g, "beta")

		cmd := ListCmd{Names: true}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "alpha\n")
		assert.Contains(t, output, "beta\n")
		assert.NotContains(t, output, "  ") // no indentation from card format
	})
}

func TestRmCmd_Run(t *testing.T) {
	t.Run("removes project by name", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := RmCmd{Name: "test-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 0, g.Cat.Count())
	})

	t.Run("removes project by partial name match", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "my-test-project")

		cmd := RmCmd{Name: "test"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 0, g.Cat.Count())
	})

	t.Run("returns error for nonexistent project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := RmCmd{Name: "nonexistent"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("does not remove when multiple matches exist", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")

		cmd := RmCmd{Name: "test"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 2, g.Cat.Count())
	})

	t.Run("removes correct project when exact match exists", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "exact-match")
		createTestProject(t, g, "other-project")

		cmd := RmCmd{Name: "exact-match"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, 1, g.Cat.Count())

		remaining := g.Cat.List()
		require.Len(t, remaining, 1)
		assert.Equal(t, "other-project", remaining[0].Name)
	})

	t.Run("outputs removal confirmation", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "doomed-project")
		out.Reset()

		cmd := RmCmd{Name: "doomed-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Removed:")
		assert.Contains(t, output, "doomed-project")
	})
}

func TestOpenCmd_Run(t *testing.T) {
	t.Setenv("EDITOR", "true")

	t.Run("launches editor and updates last accessed", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		var editorCalled bool
		var editorPath string
		g.RunCmd = func(name string, args ...string) error {
			editorCalled = true
			if len(args) > 0 {
				editorPath = args[0]
			}
			return nil
		}
		projectDir := createTestProject(t, g, "test-project")

		cmd := OpenCmd{Name: "test-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.True(t, editorCalled, "editor should be called")
		assert.Equal(t, projectDir, editorPath, "editor should be called with project path")
	})

	t.Run("updates last accessed time", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		initialTime := projects[0].LastAccessed
		time.Sleep(10 * time.Millisecond)

		cmd := OpenCmd{Name: "test-project"}
		err := cmd.Run(g)
		require.NoError(t, err)
		projects = g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.True(t, projects[0].LastAccessed.After(initialTime))
	})

	t.Run("returns error for nonexistent project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := OpenCmd{Name: "nonexistent"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("does not open when multiple matches exist", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		var editorCalled bool
		g.RunCmd = func(name string, args ...string) error {
			editorCalled = true
			return nil
		}
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")

		cmd := OpenCmd{Name: "test"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.False(t, editorCalled, "editor should not be called for ambiguous match")
	})

	t.Run("opens project by partial match", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := createTestProject(t, g, "unique-project")
		var editorPath string
		g.RunCmd = func(name string, args ...string) error {
			if len(args) > 0 {
				editorPath = args[0]
			}
			return nil
		}

		cmd := OpenCmd{Name: "unique"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, projectDir, editorPath)
		projects := g.Cat.Search("unique")
		require.Len(t, projects, 1)
		assert.Equal(t, projectDir, projects[0].Path)
	})
}

func TestEditCmd_Run(t *testing.T) {
	t.Run("returns error for nonexistent project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := EditCmd{Name: "nonexistent", Editor: "nvim"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("does not modify when multiple matches exist", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")

		cmd := EditCmd{Name: "test", Editor: "nvim"}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		for _, p := range projects {
			assert.Empty(t, p.Editor)
		}
	})

	t.Run("outputs update confirmation", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "test-project")
		out.Reset()

		cmd := EditCmd{Name: "test-project", Editor: "nvim"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Updated:")
		assert.Contains(t, output, "test-project")
	})
}

func TestIntegration_MultipleOperations(t *testing.T) {
	g, _ := newTestGlobals(t)

	goDir := t.TempDir()
	cmd1 := AddCmd{Path: goDir, Name: "go-project"}
	require.NoError(t, cmd1.Run(g))

	nodeDir := t.TempDir()
	cmd2 := AddCmd{Path: nodeDir, Name: "node-project"}
	require.NoError(t, cmd2.Run(g))

	assert.Equal(t, 2, g.Cat.Count())
	results := g.Cat.Search("go-project")
	assert.Len(t, results, 1)

	editCmd := EditCmd{
		Name:   "go-project",
		Editor: "nvim",
	}
	require.NoError(t, editCmd.Run(g))

	projects := g.Cat.Search("go-project")
	require.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, "nvim", p.Editor)

	rmCmd := RmCmd{Name: "node-project"}
	require.NoError(t, rmCmd.Run(g))
	assert.Equal(t, 1, g.Cat.Count())

	remaining := g.Cat.List()
	require.Len(t, remaining, 1)
	assert.Equal(t, "go-project", remaining[0].Name)
}

func TestCatalogPersistence(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.yaml")

	cat1, err := catalog.NewYAMLCatalog(catalogPath)
	require.NoError(t, err)
	g1 := &Globals{Cat: cat1, Out: os.Stdout}

	projectDir := t.TempDir()
	cmd := AddCmd{Path: projectDir, Name: "persistent-project"}
	require.NoError(t, cmd.Run(g1))

	cat2, err := catalog.NewYAMLCatalog(catalogPath)
	require.NoError(t, err)
	require.NoError(t, cat2.Load())
	g2 := &Globals{Cat: cat2}

	assert.Equal(t, 1, g2.Cat.Count())
	projects := g2.Cat.List()
	require.Len(t, projects, 1)
	assert.Equal(t, "persistent-project", projects[0].Name)
	assert.Equal(t, projectDir, projects[0].Path)
}

func TestCatalogPathParsing(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "short flag with space",
			args:     []string{"-c", "/tmp/custom.yaml", "list"},
			expected: "/tmp/custom.yaml",
		},
		{
			name:     "long flag with space",
			args:     []string{"--catalog", "/tmp/custom.yaml", "list"},
			expected: "/tmp/custom.yaml",
		},
		{
			name:     "long flag with equals",
			args:     []string{"--catalog=/tmp/custom.yaml", "list"},
			expected: "/tmp/custom.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := CLI{}

			parser, err := kong.New(&cli,
				kong.Name("pj"),
				kong.Description("Project tracker and launcher"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)
			_, _ = parser.Parse(tc.args)
			assert.Equal(t, tc.expected, cli.CatalogPath)
		})
	}
}

func TestCatalogPathDefault(t *testing.T) {
	cli := CLI{}

	parser, err := kong.New(&cli,
		kong.Name("pj"),
		kong.Description("Project tracker and launcher"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, _ = parser.Parse([]string{"list"})
	assert.Empty(t, cli.CatalogPath)
}

func TestShowCmd_Run(t *testing.T) {
	t.Run("displays project fields", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		out.Reset()
		cmd := ShowCmd{Name: "test-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Name:   test-project")
		assert.Contains(t, output, "Path:")
	})

	t.Run("outputs only path with --path flag", func(t *testing.T) {
		g, out := newTestGlobals(t)
		projectDir := createTestProject(t, g, "test-project")
		out.Reset()

		cmd := ShowCmd{Name: "test-project", Path: true}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Equal(t, projectDir+"\n", output)
	})

	t.Run("returns error for nonexistent project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := ShowCmd{Name: "nonexistent"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("does not show details when multiple matches exist", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")
		out.Reset()

		cmd := ShowCmd{Name: "test"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.NotContains(t, output, "Name:   test-project-1")
		assert.NotContains(t, output, "Name:   test-project-2")
	})
}

func TestInitCmd_Run(t *testing.T) {
	t.Run("outputs valid shell script", func(t *testing.T) {
		g, out := newTestGlobals(t)

		cmd := InitCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "pj shell integration")
		assert.Contains(t, output, "pj()")
		assert.Contains(t, output, "case \"$1\" in")
		assert.Contains(t, output, "cd)")
		assert.Contains(t, output, "command pj show")
	})
}

func TestEditCmd_EditorFlag(t *testing.T) {
	t.Run("sets editor on project", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := EditCmd{Name: "test-project", Editor: "nvim"}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.Equal(t, "nvim", projects[0].Editor)
	})
}

func TestOpenCmd_UsesProjectEditor(t *testing.T) {
	t.Run("uses project-specific editor", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := createTestProject(t, g, "test-project")

		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		p := projects[0]
		p.Editor = "true"
		require.NoError(t, g.Cat.Update(p))

		var editorUsed string
		g.RunCmd = func(name string, args ...string) error {
			editorUsed = name
			return nil
		}

		cmd := OpenCmd{Name: "test-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		assert.Equal(t, "true", editorUsed)
		_ = projectDir
	})
}

func TestOpenCmd_PathNotExist(t *testing.T) {
	t.Setenv("EDITOR", "true")

	t.Run("returns error when path no longer exists", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := createTestProject(t, g, "test-project")

		require.NoError(t, os.RemoveAll(projectDir))

		cmd := OpenCmd{Name: "test-project"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path no longer exists")
		assert.Contains(t, err.Error(), "pj rm")
	})
}

func TestResolveEditor(t *testing.T) {
	t.Run("uses project editor first", func(t *testing.T) {
		p := catalog.Project{Editor: "true"}
		editor, err := resolveEditor(p)
		require.NoError(t, err)
		assert.Equal(t, "true", editor)
	})

	t.Run("falls back to EDITOR env var", func(t *testing.T) {
		t.Setenv("EDITOR", "true")
		p := catalog.Project{}
		editor, err := resolveEditor(p)
		require.NoError(t, err)
		assert.Equal(t, "true", editor)
	})

	t.Run("falls back to vim", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		p := catalog.Project{}
		editor, err := resolveEditor(p)
		if err != nil {
			assert.Contains(t, err.Error(), "not found in PATH")
		} else {
			assert.Equal(t, "vim", editor)
		}
	})

	t.Run("returns error for missing editor", func(t *testing.T) {
		p := catalog.Project{Editor: "nonexistent-editor-12345"}
		_, err := resolveEditor(p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in PATH")
	})
}

func TestFindProject(t *testing.T) {
	t.Run("returns exact match", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		project, err := findProject(g.Cat, "test-project")
		require.NoError(t, err)
		assert.Equal(t, "test-project", project.Name)
	})

	t.Run("returns error for no match", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		_, err := findProject(g.Cat, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("returns AmbiguousMatchError for multiple matches", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")

		_, err := findProject(g.Cat, "test")
		require.Error(t, err)

		var ambErr *AmbiguousMatchError
		require.ErrorAs(t, err, &ambErr)
		assert.Equal(t, "test", ambErr.Query)
		assert.Len(t, ambErr.Matches, 2)
	})
}

func TestAmbiguousMatchOutput(t *testing.T) {
	t.Run("displays multiple matches message", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")
		out.Reset()

		cmd := RmCmd{Name: "test"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Multiple projects match")
		assert.Contains(t, output, "test-project-1")
		assert.Contains(t, output, "test-project-2")
	})
}

func TestKongAliases(t *testing.T) {
	testCases := []struct {
		alias   string
		command string
	}{
		{"a", "add"},
		{"ls", "list"},
		{"o", "open"},
		{"e", "edit"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s is alias for %s", tc.alias, tc.command), func(t *testing.T) {
			cli := CLI{}
			parser, err := kong.New(&cli,
				kong.Name("pj"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			require.NotPanics(t, func() {
				_, _ = parser.Parse([]string{tc.alias, "--help"})
			})
		})
	}
}

func TestListCmd_GoldenOutput(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		g, out, _ := newGoldenTestGlobals(t)

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		golden.RequireEqual(t, []byte(out.String()))
	})

	t.Run("single project", func(t *testing.T) {
		g, out, pathMap := newGoldenTestGlobals(t)
		addProjectWithTime(t, g, pathMap, "pj", "Project tracker and launcher CLI",
			time.Date(2026, 1, 7, 10, 0, 0, 0, time.Local))

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		golden.RequireEqual(t, []byte(normalizePaths(out.String(), pathMap)))
	})

	t.Run("multiple projects", func(t *testing.T) {
		g, out, pathMap := newGoldenTestGlobals(t)
		addProjectWithTime(t, g, pathMap, "pj", "Project tracker and launcher CLI",
			time.Date(2026, 1, 7, 10, 0, 0, 0, time.Local))
		addProjectWithTime(t, g, pathMap, "booster", "Go build tool with plugin architecture",
			time.Date(2026, 1, 6, 8, 0, 0, 0, time.Local))

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		golden.RequireEqual(t, []byte(normalizePaths(out.String(), pathMap)))
	})

	t.Run("no description", func(t *testing.T) {
		g, out, pathMap := newGoldenTestGlobals(t)
		addProjectWithTime(t, g, pathMap, "dotfiles", "",
			time.Date(2026, 1, 5, 12, 0, 0, 0, time.Local))

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		golden.RequireEqual(t, []byte(normalizePaths(out.String(), pathMap)))
	})

	t.Run("stale project", func(t *testing.T) {
		g, out, pathMap := newGoldenTestGlobals(t)
		addProjectWithTime(t, g, pathMap, "old-experiment", "Abandoned spike",
			time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local))

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		golden.RequireEqual(t, []byte(normalizePaths(out.String(), pathMap)))
	})
}

func newGoldenTestGlobals(t *testing.T) (*Globals, *bytes.Buffer, map[string]string) {
	t.Helper()
	return newTestGlobalsCore(t)
}

func addProjectWithTime(t *testing.T, g *Globals, pathMap map[string]string, name, description string, mtime time.Time) {
	t.Helper()
	projectDir := t.TempDir()
	require.NoError(t, os.Chtimes(projectDir, mtime, mtime))
	p := catalog.NewProject(name, projectDir)
	p.Description = description
	p.LastAccessed = mtime
	require.NoError(t, g.Cat.Add(p))
	pathMap[projectDir] = "/home/user/projects/" + name
}

func normalizePaths(output string, pathMap map[string]string) string {
	result := output
	for actual, normalized := range pathMap {
		result = strings.ReplaceAll(result, actual, normalized)
	}
	return result
}
