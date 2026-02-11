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
	tests := []struct {
		name        string
		xdgDataHome *string
		expected    string
	}{
		{
			name:        "respects XDG_DATA_HOME when set",
			xdgDataHome: new("/custom/data"),
			expected:    "/custom/data/pj/catalog.yaml",
		},
		{
			name:        "falls back to ~/.local/share when XDG_DATA_HOME is empty",
			xdgDataHome: new(""),
			expected:    "~/.local/share/pj/catalog.yaml",
		},
		{
			name:        "falls back to ~/.local/share when XDG_DATA_HOME is not set",
			xdgDataHome: nil,
			expected:    "~/.local/share/pj/catalog.yaml",
		},
		{
			name:        "handles XDG_DATA_HOME with trailing slash",
			xdgDataHome: new("/custom/data/"),
			expected:    "/custom/data/pj/catalog.yaml",
		},
		{
			name:        "handles relative XDG_DATA_HOME path",
			xdgDataHome: new("relative/path"),
			expected:    "relative/path/pj/catalog.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xdgDataHome != nil {
				t.Setenv("XDG_DATA_HOME", *tt.xdgDataHome)
			} else {
				os.Unsetenv("XDG_DATA_HOME")
			}

			got := config.DefaultCatalogPath()

			expected := tt.expected
			if home, err := os.UserHomeDir(); err == nil {
				expected = expandTilde(expected, home)
			}
			assert.Equal(t, expected, got)
		})
	}

	t.Run("falls back to relative path when HOME is unset", func(t *testing.T) {
		os.Unsetenv("HOME")
		os.Unsetenv("XDG_DATA_HOME")
		t.Cleanup(func() {
			if h, err := os.UserHomeDir(); err == nil {
				t.Setenv("HOME", h)
			}
		})

		got := config.DefaultCatalogPath()

		assert.Equal(t, ".local/share/pj/catalog.yaml", got)
	})
}

func expandTilde(path, home string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		return filepath.Join(home, path[2:])
	}
	return path
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

func TestExpandPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name:        "tabs and spaces",
			input:       "\t  \t",
			wantErr:     true,
			errContains: "path cannot be empty",
		},
		{
			name:        "tilde-user syntax",
			input:       "~otheruser/path",
			wantErr:     true,
			errContains: "~username expansion is not supported",
		},
		{
			name:        "tilde-user without path",
			input:       "~otheruser",
			wantErr:     true,
			errContains: "~username expansion is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.ExpandPath(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}
