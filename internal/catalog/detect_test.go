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
		name        string
		files       map[string]string
		dirs        []string
		useNonExist bool
		expected    []catalog.ProjectType
		checkLen    bool
		lenCheck    int
		checkMsg    string
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
			checkLen: true,
			lenCheck: 2,
		},
		{
			name:        "handles non-existent directory",
			useNonExist: true,
			expected:    []catalog.ProjectType{catalog.TypeUnknown},
		},
		{
			name:     "specific type takes precedence over generic .git",
			files:    map[string]string{"go.mod": "module test"},
			dirs:     []string{".git"},
			expected: []catalog.ProjectType{catalog.TypeGo},
			checkMsg: "go.mod should result in TypeGo, not TypeGeneric from .git",
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
			checkMsg: "should detect Rust despite other files present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.useNonExist {
				dir = filepath.Join(t.TempDir(), "non-existent")
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

			if tt.checkLen {
				assert.Len(t, got, tt.lenCheck)
				for _, expectedType := range tt.expected {
					assert.Contains(t, got, expectedType)
				}
			} else {
				if tt.checkMsg != "" {
					assert.Equal(t, tt.expected, got, tt.checkMsg)
				} else {
					assert.Equal(t, tt.expected, got)
				}
			}
		})
	}
}
