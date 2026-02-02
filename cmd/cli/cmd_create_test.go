package main

import (
	"errors"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCreateSummary(t *testing.T) {
	t.Run("shows created header with project name", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "◆ Created my-project")
	})

	t.Run("shows full path on its own line", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "/home/user/projects/my-project")
	})

	t.Run("shows directory created checkmark", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev"})

		output := out.String()
		assert.Contains(t, output, "✓ Directory created")
	})

	t.Run("shows git initialized when git enabled", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Git: true})

		output := out.String()
		assert.Contains(t, output, "✓ Git initialized")
	})

	t.Run("does not show git initialized when git disabled", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Git: false})

		output := out.String()
		assert.NotContains(t, output, "Git initialized")
	})

	t.Run("shows added to catalog checkmark", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev"})

		output := out.String()
		assert.Contains(t, output, "✓ Added to catalog")
	})
}

func TestGitLabel(t *testing.T) {
	assert.Equal(t, "Yes", gitLabel(true))
	assert.Equal(t, "No", gitLabel(false))
}

func TestValidateCreateName(t *testing.T) {
	t.Run("empty string returns Name cannot be empty", func(t *testing.T) {
		err := validateCreateName("")
		assert.EqualError(t, err, "Name cannot be empty")
	})

	t.Run("whitespace-only returns Name cannot be empty", func(t *testing.T) {
		err := validateCreateName("   ")
		assert.EqualError(t, err, "Name cannot be empty")
	})

	t.Run("valid name returns nil", func(t *testing.T) {
		err := validateCreateName("my-project")
		assert.NoError(t, err)
	})
}

func TestCreateProjectDir(t *testing.T) {
	t.Run("creates directory at location/name", func(t *testing.T) {
		location := t.TempDir()
		projectPath, err := createProjectDir(location, "my-project")

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(location, "my-project"), projectPath)
		info, err := os.Stat(projectPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("returns error when directory already exists", func(t *testing.T) {
		location := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(location, "existing"), 0o755))

		_, err := createProjectDir(location, "existing")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Directory already exists")
		assert.Contains(t, err.Error(), filepath.Join(location, "existing"))
	})

	t.Run("returns error when path is not writable", func(t *testing.T) {
		location := t.TempDir()
		require.NoError(t, os.Chmod(location, 0o555))
		t.Cleanup(func() { os.Chmod(location, 0o755) })

		_, err := createProjectDir(location, "blocked")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Permission denied")
		assert.Contains(t, err.Error(), filepath.Join(location, "blocked"))
	})
}

func TestInitGitRepo(t *testing.T) {
	t.Run("creates .git directory when git is available", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := initGitRepo(g, projectPath)

		require.NoError(t, err)
		_, statErr := os.Stat(filepath.Join(projectPath, ".git"))
		assert.NoError(t, statErr)
	})

	t.Run("git not initialized when git is No", func(t *testing.T) {
		projectPath := t.TempDir()

		_, statErr := os.Stat(filepath.Join(projectPath, ".git"))
		assert.True(t, os.IsNotExist(statErr))
	})
}

func TestCreateGitignore(t *testing.T) {
	t.Run("creates .gitignore file", func(t *testing.T) {
		projectPath := t.TempDir()

		err := createGitignore(projectPath)

		require.NoError(t, err)
		_, statErr := os.Stat(filepath.Join(projectPath, ".gitignore"))
		assert.NoError(t, statErr)
	})

	t.Run("contains OS patterns", func(t *testing.T) {
		projectPath := t.TempDir()
		require.NoError(t, createGitignore(projectPath))

		content, err := os.ReadFile(filepath.Join(projectPath, ".gitignore"))
		require.NoError(t, err)

		assert.Contains(t, string(content), ".DS_Store")
		assert.Contains(t, string(content), "Thumbs.db")
	})

	t.Run("contains editor patterns", func(t *testing.T) {
		projectPath := t.TempDir()
		require.NoError(t, createGitignore(projectPath))

		content, err := os.ReadFile(filepath.Join(projectPath, ".gitignore"))
		require.NoError(t, err)

		assert.Contains(t, string(content), ".idea/")
		assert.Contains(t, string(content), ".vscode/")
		assert.Contains(t, string(content), "*.swp")
	})

	t.Run("contains build patterns", func(t *testing.T) {
		projectPath := t.TempDir()
		require.NoError(t, createGitignore(projectPath))

		content, err := os.ReadFile(filepath.Join(projectPath, ".gitignore"))
		require.NoError(t, err)

		assert.Contains(t, string(content), "/dist/")
		assert.Contains(t, string(content), "/build/")
		assert.Contains(t, string(content), "/out/")
	})

	t.Run("contains dependency patterns", func(t *testing.T) {
		projectPath := t.TempDir()
		require.NoError(t, createGitignore(projectPath))

		content, err := os.ReadFile(filepath.Join(projectPath, ".gitignore"))
		require.NoError(t, err)

		assert.Contains(t, string(content), "/vendor/")
		assert.Contains(t, string(content), "/node_modules/")
	})
}

func TestInitGitRepoCreatesGitignore(t *testing.T) {
	g, _ := newTestGlobals(t)
	projectPath := t.TempDir()

	err := initGitRepo(g, projectPath)

	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(projectPath, ".gitignore"))
	assert.NoError(t, statErr)
}

func TestRegisterProject(t *testing.T) {
	t.Run("adds project to catalog with correct name and path", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := registerProject(g, createResult{
			Name:     "my-project",
			Location: filepath.Dir(projectPath),
		}, projectPath)

		require.NoError(t, err)
		assert.Equal(t, 1, g.Cat.Count())
		projects := g.Cat.List()
		assert.Equal(t, "my-project", projects[0].Name)
		assert.Equal(t, projectPath, projects[0].Path)
	})

	t.Run("stores description when provided", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := registerProject(g, createResult{
			Name:        "my-project",
			Location:    filepath.Dir(projectPath),
			Description: "A cool project",
		}, projectPath)

		require.NoError(t, err)
		projects := g.Cat.List()
		assert.Equal(t, "A cool project", projects[0].Description)
	})

	t.Run("stores editor when provided", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := registerProject(g, createResult{
			Name:     "my-project",
			Location: filepath.Dir(projectPath),
			Editor:   "nvim",
		}, projectPath)

		require.NoError(t, err)
		projects := g.Cat.List()
		assert.Equal(t, "nvim", projects[0].Editor)
	})

	t.Run("empty description and editor stored as empty", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := registerProject(g, createResult{
			Name:     "my-project",
			Location: filepath.Dir(projectPath),
		}, projectPath)

		require.NoError(t, err)
		projects := g.Cat.List()
		assert.Empty(t, projects[0].Description)
		assert.Empty(t, projects[0].Editor)
	})

	t.Run("persists to catalog file", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		projectPath := t.TempDir()

		err := registerProject(g, createResult{
			Name:     "my-project",
			Location: filepath.Dir(projectPath),
		}, projectPath)

		require.NoError(t, err)
		require.NoError(t, g.Cat.Load())
		assert.Equal(t, 1, g.Cat.Count())
	})
}

func TestPrintCdHint(t *testing.T) {
	t.Run("writes path to cd file when env set", func(t *testing.T) {
		g, out := newTestGlobals(t)
		cdFile := filepath.Join(t.TempDir(), "cd-target")
		t.Setenv("__PJ_CD_FILE", cdFile)

		printCdHint(g, "/home/user/projects/my-project")

		content, err := os.ReadFile(cdFile)
		require.NoError(t, err)
		assert.Equal(t, "/home/user/projects/my-project", string(content))
		assert.Empty(t, out.String())
	})

	t.Run("prints cd command when env not set", func(t *testing.T) {
		g, out := newTestGlobals(t)
		t.Setenv("__PJ_CD_FILE", "")

		printCdHint(g, "/home/user/projects/my-project")

		assert.Contains(t, out.String(), "cd /home/user/projects/my-project")
	})
}

func TestHandleCreateFormError(t *testing.T) {
	t.Run("ErrUserAborted returns nil", func(t *testing.T) {
		err := handleCreateFormError(huh.ErrUserAborted)
		assert.NoError(t, err)
	})

	t.Run("other errors propagate", func(t *testing.T) {
		err := handleCreateFormError(errors.New("unexpected"))
		assert.Error(t, err)
	})
}

func TestExecuteCreateCleansUpOnError(t *testing.T) {
	t.Run("removes directory when catalog add fails", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		projectPath := filepath.Join(location, "doomed")
		require.NoError(t, os.Mkdir(projectPath, 0o755))
		require.NoError(t, g.Cat.Add(catalog.NewProject("doomed", projectPath)))
		require.NoError(t, os.Remove(projectPath))

		err := executeCreate(g, createResult{
			Name:     "doomed",
			Location: location,
		})

		require.Error(t, err)
		_, statErr := os.Stat(projectPath)
		assert.True(t, os.IsNotExist(statErr), "directory should be removed on failure")
	})

	t.Run("catalog unchanged when directory creation fails", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(location, "existing"), 0o755))

		err := executeCreate(g, createResult{
			Name:     "existing",
			Location: location,
		})

		require.Error(t, err)
		assert.Equal(t, 0, g.Cat.Count())
	})

	t.Run("directory preserved on success", func(t *testing.T) {
		g, _ := newTestGlobals(t)
		location := t.TempDir()
		t.Setenv("__PJ_CD_FILE", "")

		err := executeCreate(g, createResult{
			Name:     "keeper",
			Location: location,
		})

		require.NoError(t, err)
		projectPath := filepath.Join(location, "keeper")
		info, statErr := os.Stat(projectPath)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})
}
