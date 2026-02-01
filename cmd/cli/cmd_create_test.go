package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCreateSummary(t *testing.T) {
	t.Run("shows collapsed name field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
	})

	t.Run("shows collapsed location field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/home/user/projects")
	})

	t.Run("shows both name and location fields", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev"})

		output := out.String()
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/tmp/dev")
	})

	t.Run("shows collapsed description field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Description: "A cool project"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Description")
		assert.Contains(t, output, "A cool project")
	})

	t.Run("omits empty description from output", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Description: ""})

		output := out.String()
		assert.NotContains(t, output, "Description")
	})

	t.Run("shows collapsed editor field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Editor: "vim"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Editor")
		assert.Contains(t, output, "vim")
	})

	t.Run("omits empty editor from output", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Editor: ""})

		output := out.String()
		assert.NotContains(t, output, "Editor")
	})

	t.Run("shows git yes when enabled", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Git: true})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Git")
		assert.Contains(t, output, "Yes")
	})

	t.Run("shows git no when disabled", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Git: false})

		output := out.String()
		assert.Contains(t, output, "Git")
		assert.Contains(t, output, "No")
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
