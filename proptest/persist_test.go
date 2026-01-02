package proptest

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"pgregory.net/rapid"
)

func requireNoPanic(rt *rapid.T, description, input string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			rt.Fatalf("%s panicked: %v\nInput: %q", description, r, input)
		}
	}()
	fn()
}

func TestProperty_SaveLoad_RoundTrip(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		added := h.AddProjects(typicalMinProjects, typicalMaxProjects)
		if len(added) == 0 {
			h.T.Skip("no projects added")
		}

		if err := h.Catalog.Save(); err != nil {
			h.T.Fatalf("failed to save: %v", err)
		}

		catalogPath := filepath.Join(h.Dir, "catalog.yaml")
		catalog2, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			h.T.Fatalf("failed to create second catalog: %v", err)
		}
		if err := catalog2.Load(); err != nil {
			h.T.Fatalf("failed to load: %v", err)
		}

		if h.Catalog.Count() != catalog2.Count() {
			h.T.Fatalf("count mismatch after load: %d vs %d", h.Catalog.Count(), catalog2.Count())
		}

		for _, p := range added {
			loaded, err := catalog2.Get(p.ID)
			if err != nil {
				h.T.Fatalf("project %s not found after load", p.ID)
			}
			assertProjectsEqual(h.T, p, loaded)
		}
	})
}

func TestProperty_Load_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		catalogPath := filepath.Join(iterDir, "catalog.yaml")
		if err := os.WriteFile(catalogPath, []byte(""), 0o644); err != nil {
			rt.Fatalf("failed to write empty file: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		err = cat.Load()
		if err != nil {
			rt.Fatalf("Load should succeed on empty file, got: %v", err)
		}

		if cat.Count() != 0 {
			rt.Fatalf("expected 0 projects from empty file, got %d", cat.Count())
		}
	})
}

func TestProperty_Load_MalformedYAML(t *testing.T) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		catalogPath := filepath.Join(iterDir, "catalog.yaml")
		malformed := malformedYAMLGen().Draw(rt, "malformed")
		if err := os.WriteFile(catalogPath, []byte(malformed), 0o644); err != nil {
			rt.Fatalf("failed to write malformed file: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		requireNoPanic(rt, "Load on malformed YAML", malformed, func() {
			_ = cat.Load()
		})
	})
}

func TestProperty_Load_MissingFields(t *testing.T) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		catalogPath := filepath.Join(iterDir, "catalog.yaml")
		content := missingFieldsGen().Draw(rt, "content")
		if err := os.WriteFile(catalogPath, []byte(content), 0o644); err != nil {
			rt.Fatalf("failed to write file: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		requireNoPanic(rt, "Load on missing fields", content, func() {
			_ = cat.Load()
		})
	})
}

func TestProperty_Load_ExtraFields(t *testing.T) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		catalogPath := filepath.Join(iterDir, "catalog.yaml")
		content := extraFieldsGen().Draw(rt, "content")
		if err := os.WriteFile(catalogPath, []byte(content), 0o644); err != nil {
			rt.Fatalf("failed to write file: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		requireNoPanic(rt, "Load on extra fields", content, func() {
			err = cat.Load()
			if err != nil {
				rt.Fatalf("Load should ignore extra fields, got error: %v", err)
			}

			if cat.Count() != 1 {
				rt.Fatalf("expected 1 project, got %d", cat.Count())
			}

			projects := cat.List()
			if projects[0].ID != "test-id" {
				rt.Fatalf("expected ID 'test-id', got %q", projects[0].ID)
			}
			if projects[0].Name != "test-project" {
				rt.Fatalf("expected Name 'test-project', got %q", projects[0].Name)
			}
		})
	})
}

func TestProperty_Load_InvalidTypes(t *testing.T) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		catalogPath := filepath.Join(iterDir, "catalog.yaml")
		content := invalidTypesGen().Draw(rt, "content")
		if err := os.WriteFile(catalogPath, []byte(content), 0o644); err != nil {
			rt.Fatalf("failed to write file: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(catalogPath)
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		requireNoPanic(rt, "Load on invalid types", content, func() {
			_ = cat.Load()
		})
	})
}
