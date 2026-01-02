package proptest

import (
	"errors"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"
)

func TestProperty_Catalog_CountConsistency(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(minProjects, maxProjects)

		count := h.Catalog.Count()
		listLen := len(h.Catalog.List())

		if count != listLen {
			h.T.Fatalf("Count() = %d but len(List()) = %d", count, listLen)
		}
	})
}

func TestProperty_Catalog_PathUniqueness(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(typicalMinProjects, typicalMaxProjects)

		projects := h.Catalog.List()
		assertNoDuplicatePaths(h.T, projects)
	})
}

func TestProperty_AddRemove_CountRestored(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		initialCount := h.Catalog.Count()

		p := h.GenProject()
		err := h.Catalog.Add(p)
		if err != nil {
			h.T.Skip("project rejected (expected for some inputs)")
		}

		if h.Catalog.Count() != initialCount+1 {
			h.T.Fatalf("count after add: expected %d, got %d", initialCount+1, h.Catalog.Count())
		}

		if err := h.Catalog.Remove(p.ID); err != nil {
			h.T.Fatalf("failed to remove: %v", err)
		}

		if h.Catalog.Count() != initialCount {
			h.T.Fatalf("count after remove: expected %d, got %d", initialCount, h.Catalog.Count())
		}
	})
}

func TestProperty_DuplicatePath_Rejected(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		projectPath := filepath.Join(h.Dir, subdirGen.Draw(h.T, "subdir"))
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			h.T.Fatalf("failed to create project path: %v", err)
		}

		p1 := catalog.NewProject(validNameGen().Draw(h.T, "name1"), projectPath)
		if err := h.Catalog.Add(p1); err != nil {
			h.T.Fatalf("failed to add first project: %v", err)
		}

		countBefore := h.Catalog.Count()

		p2 := catalog.NewProject(validNameGen().Draw(h.T, "name2"), projectPath)
		err := h.Catalog.Add(p2)

		if !errors.Is(err, catalog.ErrAlreadyExists) {
			h.T.Fatalf("expected ErrAlreadyExists, got %v", err)
		}

		if h.Catalog.Count() != countBefore {
			h.T.Fatalf("count changed after rejected add: expected %d, got %d", countBefore, h.Catalog.Count())
		}
	})
}

func TestProperty_Update_PathMutation(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		oldPath := filepath.Join(h.Dir, subdirGen.Draw(h.T, "oldSubdir"))
		if err := os.MkdirAll(oldPath, 0o755); err != nil {
			h.T.Fatalf("failed to create old path: %v", err)
		}

		p := catalog.NewProject(validNameGen().Draw(h.T, "name"), oldPath)
		if err := h.Catalog.Add(p); err != nil {
			h.T.Fatalf("failed to add project: %v", err)
		}

		newPath := filepath.Join(h.Dir, subdirGen.Draw(h.T, "newSubdir"))
		if err := os.MkdirAll(newPath, 0o755); err != nil {
			h.T.Fatalf("failed to create new path: %v", err)
		}

		updatedProject := p
		updatedProject.Path = newPath
		if err := h.Catalog.Update(updatedProject); err != nil {
			h.T.Fatalf("failed to update project: %v", err)
		}

		_, err := h.Catalog.GetByPath(oldPath)
		if !errors.Is(err, catalog.ErrNotFound) {
			h.T.Fatalf("expected ErrNotFound for old path, got %v", err)
		}

		found, err := h.Catalog.GetByPath(newPath)
		if err != nil {
			h.T.Fatalf("expected success for new path, got %v", err)
		}
		if found.ID != p.ID {
			h.T.Fatalf("GetByPath returned wrong project: expected %s, got %s", p.ID, found.ID)
		}
	})
}

func TestProperty_GetByPath_ConsistentWithGet(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		p := h.GenProject()
		err := h.Catalog.Add(p)
		if err != nil {
			h.T.Skip("project rejected")
		}

		byID, err1 := h.Catalog.Get(p.ID)
		byPath, err2 := h.Catalog.GetByPath(p.Path)

		if err1 != nil || err2 != nil {
			h.T.Fatalf("Get errors: byID=%v, byPath=%v", err1, err2)
		}

		if byID.ID != byPath.ID {
			h.T.Fatalf("Get and GetByPath returned different projects")
		}
	})
}
