package config_test

import (
	"os"
	"path/filepath"
	"pj/internal/config"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCatalogPath(t *testing.T) {
	t.Run("respects XDG_DATA_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data")

		got := config.DefaultCatalogPath()

		assert.Equal(t, "/custom/data/pj/catalog.yaml", got)
	})

	t.Run("falls back to ~/.local/share when XDG_DATA_HOME is empty", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")

		got := config.DefaultCatalogPath()

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		expected := filepath.Join(home, ".local", "share", "pj", "catalog.yaml")
		assert.Equal(t, expected, got)
	})

	t.Run("falls back to ~/.local/share when XDG_DATA_HOME is not set", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")

		got := config.DefaultCatalogPath()

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		expected := filepath.Join(home, ".local", "share", "pj", "catalog.yaml")
		assert.Equal(t, expected, got)
	})

	t.Run("handles XDG_DATA_HOME with trailing slash", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data/")

		got := config.DefaultCatalogPath()

		assert.Equal(t, "/custom/data/pj/catalog.yaml", got)
	})

	t.Run("handles relative XDG_DATA_HOME path", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "relative/path")

		got := config.DefaultCatalogPath()

		assert.Equal(t, "relative/path/pj/catalog.yaml", got)
	})

	t.Run("produces valid path even when HOME is unset", func(t *testing.T) {
		os.Unsetenv("HOME")
		os.Unsetenv("XDG_DATA_HOME")
		t.Cleanup(func() {
			if h, err := os.UserHomeDir(); err == nil {
				t.Setenv("HOME", h)
			}
		})

		got := config.DefaultCatalogPath()

		assert.True(t, filepath.IsAbs(got) || got == ".local/share/pj/catalog.yaml",
			"path should be absolute or have documented fallback, got: %s", got)
	})
}

func TestDefaultProjectsDir(t *testing.T) {
	t.Run("returns ~/projects when it exists", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		projectsDir := filepath.Join(tempHome, "projects")
		err := os.Mkdir(projectsDir, 0o755)
		require.NoError(t, err)

		got := config.DefaultProjectsDir()

		assert.Equal(t, projectsDir, got)
	})

	t.Run("returns ~/ when ~/projects does not exist", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		got := config.DefaultProjectsDir()

		assert.Equal(t, tempHome, got)
	})

	t.Run("returns ~/ when ~/projects exists but is a file", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		projectsFile := filepath.Join(tempHome, "projects")
		err := os.WriteFile(projectsFile, []byte("not a directory"), 0o644)
		require.NoError(t, err)

		got := config.DefaultProjectsDir()

		assert.Equal(t, tempHome, got)
	})

	t.Run("returns ~/ when ~/projects is not readable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission test not reliable on Windows")
		}

		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		projectsDir := filepath.Join(tempHome, "projects")
		err := os.Mkdir(projectsDir, 0o000)
		require.NoError(t, err)
		t.Cleanup(func() { os.Chmod(projectsDir, 0o755) })

		got := config.DefaultProjectsDir()

		assert.Equal(t, tempHome, got)
	})

	t.Run("handles symlink to directory for ~/projects", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		realDir := filepath.Join(tempHome, "real_projects")
		err := os.Mkdir(realDir, 0o755)
		require.NoError(t, err)

		projectsLink := filepath.Join(tempHome, "projects")
		err = os.Symlink(realDir, projectsLink)
		require.NoError(t, err)

		got := config.DefaultProjectsDir()

		assert.Equal(t, projectsLink, got)
	})
}

func TestExpandPath(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		home     string
		expected func(home, cwd string) string
	}{
		{
			name:     "tilde expansion with subpath",
			input:    "~/projects",
			home:     "/home/test",
			expected: func(home, _ string) string { return filepath.Join(home, "projects") },
		},
		{
			name:     "tilde only",
			input:    "~",
			home:     "/home/test",
			expected: func(home, _ string) string { return home },
		},
		{
			name:     "dot expands to current dir",
			input:    ".",
			expected: func(_, cwd string) string { return cwd },
		},
		{
			name:     "relative path becomes absolute",
			input:    "subdir/project",
			expected: func(_, cwd string) string { return filepath.Join(cwd, "subdir/project") },
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path",
			expected: func(_, _ string) string { return "/absolute/path" },
		},
		{
			name:     "tilde with spaces in path",
			input:    "~/my projects/test",
			home:     "/home/test",
			expected: func(home, _ string) string { return filepath.Join(home, "my projects/test") },
		},
		{
			name:     "tilde in middle not expanded",
			input:    "foo/~/bar",
			expected: func(_, cwd string) string { return filepath.Join(cwd, "foo/~/bar") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.home != "" {
				t.Setenv("HOME", tt.home)
			}

			home, _ := os.UserHomeDir()

			result, err := config.ExpandPath(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected(home, cwd), result)
		})
	}
}
