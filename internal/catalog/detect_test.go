package catalog_test

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		dirs     []string
		path     func(t *testing.T) string
		expected []catalog.ProjectType
	}{
		{
			name:     "detects go project",
			files:    map[string]string{"go.mod": "module test"},
			expected: []catalog.ProjectType{catalog.TypeGo},
		},
		{
			name:     "detects rust project",
			files:    map[string]string{"Cargo.toml": "[package]"},
			expected: []catalog.ProjectType{catalog.TypeRust},
		},
		{
			name:     "detects node project",
			files:    map[string]string{"package.json": "{}"},
			expected: []catalog.ProjectType{catalog.TypeNode},
		},
		{
			name:     "detects python project with pyproject.toml",
			files:    map[string]string{"pyproject.toml": "[project]"},
			expected: []catalog.ProjectType{catalog.TypePython},
		},
		{
			name:     "detects python project with requirements.txt",
			files:    map[string]string{"requirements.txt": "requests"},
			expected: []catalog.ProjectType{catalog.TypePython},
		},
		{
			name:     "detects elixir project",
			files:    map[string]string{"mix.exs": "defmodule"},
			expected: []catalog.ProjectType{catalog.TypeElixir},
		},
		{
			name:     "detects ruby project",
			files:    map[string]string{"Gemfile": "source"},
			expected: []catalog.ProjectType{catalog.TypeRuby},
		},
		{
			name:     "detects java project with pom.xml",
			files:    map[string]string{"pom.xml": "<project>"},
			expected: []catalog.ProjectType{catalog.TypeJava},
		},
		{
			name:     "detects java project with build.gradle",
			files:    map[string]string{"build.gradle": "plugins {}"},
			expected: []catalog.ProjectType{catalog.TypeJava},
		},
		{
			name:     "detects java project with build.gradle.kts",
			files:    map[string]string{"build.gradle.kts": "plugins {}"},
			expected: []catalog.ProjectType{catalog.TypeJava},
		},
		{
			name:     "returns generic for git only",
			dirs:     []string{".git"},
			expected: []catalog.ProjectType{catalog.TypeGeneric},
		},
		{
			name:     "returns unknown for empty directory",
			expected: []catalog.ProjectType{catalog.TypeUnknown},
		},
		{
			name: "detects multiple types when both exist",
			files: map[string]string{
				"go.mod":       "module test",
				"package.json": "{}",
			},
			expected: []catalog.ProjectType{catalog.TypeGo, catalog.TypeNode},
		},
		{
			name: "handles non-existent directory",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "non-existent")
			},
			expected: []catalog.ProjectType{catalog.TypeUnknown},
		},
		{
			name:     "specific type takes precedence over generic .git",
			files:    map[string]string{"go.mod": "module test"},
			dirs:     []string{".git"},
			expected: []catalog.ProjectType{catalog.TypeGo},
		},
		{
			name: "deduplicates python when both markers exist",
			files: map[string]string{
				"pyproject.toml":   "[project]",
				"requirements.txt": "requests",
			},
			expected: []catalog.ProjectType{catalog.TypePython},
		},
		{
			name: "detects project type with multiple unrelated files",
			files: map[string]string{
				"README.md":  "# Test",
				".gitignore": "node_modules",
				"Cargo.toml": "[package]",
			},
			expected: []catalog.ProjectType{catalog.TypeRust},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.path != nil {
				dir = tt.path(t)
			} else {
				dir = t.TempDir()

				for _, dirName := range tt.dirs {
					err := os.Mkdir(filepath.Join(dir, dirName), 0o755)
					require.NoError(t, err)
				}

				for name, content := range tt.files {
					err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
					require.NoError(t, err)
				}
			}

			got := catalog.DetectProjectTypes(dir)
			assert.ElementsMatch(t, tt.expected, got)
		})
	}
}

func TestDetectProjectTypes_Symlinks(t *testing.T) {
	t.Run("handles symlinked project directory", func(t *testing.T) {
		dir := t.TempDir()
		realDir := filepath.Join(dir, "real")
		require.NoError(t, os.Mkdir(realDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(realDir, "go.mod"), []byte("module x"), 0o644))

		linkDir := filepath.Join(dir, "link")
		require.NoError(t, os.Symlink(realDir, linkDir))

		got := catalog.DetectProjectTypes(linkDir)
		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, got)
	})

	t.Run("handles symlinked marker file", func(t *testing.T) {
		dir := t.TempDir()
		realMod := filepath.Join(dir, "real-go.mod")
		require.NoError(t, os.WriteFile(realMod, []byte("module x"), 0o644))
		require.NoError(t, os.Symlink(realMod, filepath.Join(dir, "go.mod")))

		got := catalog.DetectProjectTypes(dir)
		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, got)
	})

	t.Run("handles broken symlink gracefully", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Symlink("/nonexistent/go.mod", filepath.Join(dir, "go.mod")))

		assert.NotPanics(t, func() {
			got := catalog.DetectProjectTypes(dir)
			assert.Equal(t, []catalog.ProjectType{catalog.TypeUnknown}, got)
		})
	})
}
