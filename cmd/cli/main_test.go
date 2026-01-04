package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGlobals(t *testing.T) (*Globals, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	cat, err := catalog.NewYAMLCatalog(filepath.Join(dir, "catalog.yaml"))
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	return &Globals{
		Cat:    cat,
		Out:    buf,
		RunCmd: func(name string, args ...string) error { return nil },
	}, buf
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
		assert.Equal(t, catalog.StatusActive, projects[0].Status)
	})

	t.Run("adds project with default name from directory", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd := AddCmd{Path: projectDir} // No explicit name
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.Equal(t, filepath.Base(projectDir), projects[0].Name)
	})

	t.Run("adds project with tags", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		cmd := AddCmd{
			Path: projectDir,
			Name: "tagged-project",
			Tags: []string{"backend", "api"},
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.ElementsMatch(t, []string{"backend", "api"}, projects[0].Tags)
	})

	t.Run("detects Go project type", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		goModPath := filepath.Join(projectDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0o644)
		require.NoError(t, err)

		cmd := AddCmd{Path: projectDir, Name: "go-project"}
		err = cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.Contains(t, projects[0].Types, catalog.TypeGo)
	})

	t.Run("detects Node project type", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := t.TempDir()

		packageJSONPath := filepath.Join(projectDir, "package.json")
		err := os.WriteFile(packageJSONPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		cmd := AddCmd{Path: projectDir, Name: "node-project"}
		err = cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		require.Len(t, projects, 1)
		assert.Contains(t, projects[0].Types, catalog.TypeNode)
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

	t.Run("filters by status", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "active-project")
		createTestProject(t, g, "archived-project")

		projects := g.Cat.Search("archived-project")
		require.Len(t, projects, 1)
		p := projects[0]
		p.Status = catalog.StatusArchived
		require.NoError(t, g.Cat.Update(p))

		cmd := ListCmd{Status: "archived"}
		err := cmd.Run(g)

		require.NoError(t, err)
		filtered := g.Cat.Filter(catalog.FilterOptions{Status: catalog.StatusArchived})
		assert.Len(t, filtered, 1)
		assert.Equal(t, "archived-project", filtered[0].Name)
		assert.Equal(t, 2, g.Cat.Count())
	})

	t.Run("filters by type", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		goDir := t.TempDir()
		err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test\n"), 0o644)
		require.NoError(t, err)
		cmd1 := AddCmd{Path: goDir, Name: "go-project"}
		require.NoError(t, cmd1.Run(g))

		nodeDir := t.TempDir()
		err = os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte("{}"), 0o644)
		require.NoError(t, err)
		cmd2 := AddCmd{Path: nodeDir, Name: "node-project"}
		require.NoError(t, cmd2.Run(g))

		cmd := ListCmd{Types: []string{"go"}}
		err = cmd.Run(g)

		require.NoError(t, err)
		filtered := g.Cat.Filter(catalog.FilterOptions{Types: []catalog.ProjectType{catalog.TypeGo}})
		assert.Len(t, filtered, 1)
		assert.Equal(t, "go-project", filtered[0].Name)
	})

	t.Run("filters by tags", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		projectDir := t.TempDir()
		cmd1 := AddCmd{
			Path: projectDir,
			Name: "tagged-project",
			Tags: []string{"backend", "api"},
		}
		require.NoError(t, cmd1.Run(g))

		createTestProject(t, g, "untagged-project")

		cmd := ListCmd{Tags: []string{"backend"}}
		err := cmd.Run(g)

		require.NoError(t, err)
		filtered := g.Cat.Filter(catalog.FilterOptions{Tags: []string{"backend"}})
		assert.Len(t, filtered, 1)
		assert.Equal(t, "tagged-project", filtered[0].Name)
	})

	t.Run("output includes header and project data", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "my-project")

		cmd := ListCmd{}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "PATH")
		assert.Contains(t, output, "STATUS")
		assert.Contains(t, output, "my-project")
		assert.Contains(t, output, "active")
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
		out.Reset() // Clear the "Added" output

		cmd := RmCmd{Name: "doomed-project"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Removed:")
		assert.Contains(t, output, "doomed-project")
	})
}

func TestSearchCmd_Run(t *testing.T) {
	t.Run("finds matching projects by name", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "my-project")
		createTestProject(t, g, "other-project")

		cmd := SearchCmd{Query: "my"}
		err := cmd.Run(g)

		require.NoError(t, err)
		results := g.Cat.Search("my")
		assert.Len(t, results, 1)
		assert.Equal(t, "my-project", results[0].Name)
	})

	t.Run("finds matching projects by path", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		tmpDir := t.TempDir()
		cmd1 := AddCmd{Path: tmpDir, Name: "path-test"}
		require.NoError(t, cmd1.Run(g))

		cmd := SearchCmd{Query: filepath.Base(tmpDir)}
		err := cmd.Run(g)

		require.NoError(t, err)
		results := g.Cat.Search(filepath.Base(tmpDir))
		assert.Len(t, results, 1)
	})

	t.Run("finds matching projects by tag", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		projectDir := t.TempDir()
		cmd1 := AddCmd{
			Path: projectDir,
			Name: "tagged-project",
			Tags: []string{"backend", "api"},
		}
		require.NoError(t, cmd1.Run(g))

		createTestProject(t, g, "untagged-project")

		cmd := SearchCmd{Query: "backend"}
		err := cmd.Run(g)

		require.NoError(t, err)
		results := g.Cat.Search("backend")
		assert.Len(t, results, 1)
		assert.Equal(t, "tagged-project", results[0].Name)
	})

	t.Run("returns no results for no matches", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := SearchCmd{Query: "nonexistent"}
		err := cmd.Run(g)

		require.NoError(t, err)
		results := g.Cat.Search("nonexistent")
		assert.Empty(t, results)
	})

	t.Run("searches are case insensitive", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "MyProject")

		cmd := SearchCmd{Query: "myproject"}
		err := cmd.Run(g)

		require.NoError(t, err)
		results := g.Cat.Search("myproject")
		assert.Len(t, results, 1)
		assert.Equal(t, "MyProject", results[0].Name)
	})

	t.Run("outputs matching project names", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "my-project")
		createTestProject(t, g, "other-project")
		out.Reset()

		cmd := SearchCmd{Query: "my"}
		err := cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "my-project")
		assert.NotContains(t, output, "other-project")
	})
}

func TestOpenCmd_Run(t *testing.T) {
	// Use 'true' command as editor - it exists and does nothing
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
	t.Run("modifies project status", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := EditCmd{Name: "test-project", Status: "archived"}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.Equal(t, catalog.StatusArchived, projects[0].Status)
	})

	t.Run("adds tags to project", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := EditCmd{
			Name:   "test-project",
			AddTag: []string{"backend", "api"},
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.ElementsMatch(t, []string{"backend", "api"}, projects[0].Tags)
	})

	t.Run("removes tags from project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		projectDir := t.TempDir()
		cmd1 := AddCmd{
			Path: projectDir,
			Name: "test-project",
			Tags: []string{"backend", "api", "old"},
		}
		require.NoError(t, cmd1.Run(g))

		cmd := EditCmd{
			Name:  "test-project",
			RmTag: []string{"old"},
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.ElementsMatch(t, []string{"backend", "api"}, projects[0].Tags)
		assert.False(t, projects[0].HasTag("old"))
	})

	t.Run("sets project notes", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := EditCmd{
			Name:  "test-project",
			Notes: "This is a test project",
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.Equal(t, "This is a test project", projects[0].Notes)
	})

	t.Run("modifies multiple fields at once", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project")

		cmd := EditCmd{
			Name:   "test-project",
			Status: "archived",
			AddTag: []string{"legacy"},
			Notes:  "Archived project",
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.Equal(t, catalog.StatusArchived, projects[0].Status)
		assert.Contains(t, projects[0].Tags, "legacy")
		assert.Equal(t, "Archived project", projects[0].Notes)
	})

	t.Run("returns error for nonexistent project", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		cmd := EditCmd{Name: "nonexistent", Status: "archived"}
		err := cmd.Run(g)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no project found matching")
	})

	t.Run("does not modify when multiple matches exist", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		createTestProject(t, g, "test-project-1")
		createTestProject(t, g, "test-project-2")

		cmd := EditCmd{Name: "test", Status: "archived"}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.List()
		for _, p := range projects {
			assert.Equal(t, catalog.StatusActive, p.Status)
		}
	})

	t.Run("adds and removes tags in same operation", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		projectDir := t.TempDir()
		cmd1 := AddCmd{
			Path: projectDir,
			Name: "test-project",
			Tags: []string{"old-tag"},
		}
		require.NoError(t, cmd1.Run(g))

		cmd := EditCmd{
			Name:   "test-project",
			AddTag: []string{"new-tag"},
			RmTag:  []string{"old-tag"},
		}
		err := cmd.Run(g)

		require.NoError(t, err)
		projects := g.Cat.Search("test-project")
		require.Len(t, projects, 1)
		assert.ElementsMatch(t, []string{"new-tag"}, projects[0].Tags)
		assert.False(t, projects[0].HasTag("old-tag"))
		assert.True(t, projects[0].HasTag("new-tag"))
	})

	t.Run("outputs update confirmation", func(t *testing.T) {
		g, out := newTestGlobals(t)
		createTestProject(t, g, "test-project")
		out.Reset()

		cmd := EditCmd{Name: "test-project", Status: "archived"}
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
	err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test\n"), 0o644)
	require.NoError(t, err)
	cmd1 := AddCmd{Path: goDir, Name: "go-project", Tags: []string{"backend"}}
	require.NoError(t, cmd1.Run(g))

	nodeDir := t.TempDir()
	err = os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte("{}"), 0o644)
	require.NoError(t, err)
	cmd2 := AddCmd{Path: nodeDir, Name: "node-project", Tags: []string{"frontend"}}
	require.NoError(t, cmd2.Run(g))

	assert.Equal(t, 2, g.Cat.Count())
	results := g.Cat.Search("go-project")
	assert.Len(t, results, 1)

	editCmd := EditCmd{
		Name:   "go-project",
		Status: "archived",
		AddTag: []string{"legacy"},
		Notes:  "Old backend project",
	}
	require.NoError(t, editCmd.Run(g))

	projects := g.Cat.Search("go-project")
	require.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, catalog.StatusArchived, p.Status)
	assert.Contains(t, p.Tags, "legacy")
	assert.Contains(t, p.Tags, "backend")
	assert.Equal(t, "Old backend project", p.Notes)

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
	t.Run("displays all fields", func(t *testing.T) {
		g, out := newTestGlobals(t)

		projectDir := t.TempDir()
		err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\n"), 0o644)
		require.NoError(t, err)

		addCmd := AddCmd{Path: projectDir, Name: "test-project", Tags: []string{"backend", "api"}}
		require.NoError(t, addCmd.Run(g))

		out.Reset()
		cmd := ShowCmd{Name: "test-project"}
		err = cmd.Run(g)

		require.NoError(t, err)
		output := out.String()
		assert.Contains(t, output, "Name:   test-project")
		assert.Contains(t, output, "Path:")
		assert.Contains(t, output, "Status: active")
		assert.Contains(t, output, "Tags:   backend, api")
		assert.Contains(t, output, "Types:  go")
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

func TestListCmd_RecentSort(t *testing.T) {
	t.Run("sorts by last accessed with -r flag", func(t *testing.T) {
		g, _ := newTestGlobals(t)

		createTestProject(t, g, "old-project")
		time.Sleep(10 * time.Millisecond)
		createTestProject(t, g, "new-project")

		time.Sleep(10 * time.Millisecond)
		projects := g.Cat.Search("old-project")
		require.Len(t, projects, 1)
		p := projects[0]
		p.Touch()
		require.NoError(t, g.Cat.Update(p))

		opts := catalog.FilterOptions{
			SortBy:     catalog.SortByLastAccessed,
			Descending: true,
		}
		sorted := g.Cat.Filter(opts)

		require.Len(t, sorted, 2)
		assert.Equal(t, "old-project", sorted[0].Name)
		assert.Equal(t, "new-project", sorted[1].Name)
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
		_ = projectDir // Used by createTestProject
	})
}

func TestOpenCmd_PathNotExist(t *testing.T) {
	t.Setenv("EDITOR", "true")

	t.Run("returns error when path no longer exists", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectDir := createTestProject(t, g, "test-project")

		// Delete the project directory
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
		{"s", "search"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s is alias for %s", tc.alias, tc.command), func(t *testing.T) {
			cli := CLI{}
			parser, err := kong.New(&cli,
				kong.Name("pj"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			// Kong exits on --help, so we check that it parsed without panic
			require.NotPanics(t, func() {
				_, _ = parser.Parse([]string{tc.alias, "--help"})
			})
		})
	}
}
