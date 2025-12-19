package catalog

import (
	"os"
	"path/filepath"
)

func DetectProjectTypes(path string) []ProjectType {
	markers := []struct {
		file string
		typ  ProjectType
	}{
		{"go.mod", TypeGo},
		{"Cargo.toml", TypeRust},
		{"package.json", TypeNode},
		{"pyproject.toml", TypePython},
		{"requirements.txt", TypePython},
		{"mix.exs", TypeElixir},
		{"Gemfile", TypeRuby},
		{"pom.xml", TypeJava},
		{"build.gradle", TypeJava},
		{"build.gradle.kts", TypeJava},
	}

	var types []ProjectType
	seen := make(map[ProjectType]bool)

	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(path, m.file)); err == nil {
			if !seen[m.typ] {
				seen[m.typ] = true
				types = append(types, m.typ)
			}
		}
	}

	if len(types) > 0 {
		return types
	}

	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return []ProjectType{TypeGeneric}
	}

	return []ProjectType{TypeUnknown}
}
