package proptest

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"pgregory.net/rapid"
)

const (
	minProjects          = 0
	maxProjects          = 20
	typicalMinProjects   = 1
	typicalMaxProjects   = 10
	transitivityMinCount = 3
	minTypes             = 0
	maxTypes             = 3
	minTags              = 0
	maxTags              = 5
	filterMinSlice       = 1
	filterMaxSlice       = 3
	minUnrelatedProjects = 1
	maxUnrelatedProjects = 5
)

type ProjectGenOpt func(*projectGenConfig)

type projectGenConfig struct {
	name   *string
	status *catalog.Status
	types  []catalog.ProjectType
	tags   []string
	notes  *string
}

func WithName(name string) ProjectGenOpt {
	return func(c *projectGenConfig) {
		c.name = &name
	}
}

func WithStatus(status catalog.Status) ProjectGenOpt {
	return func(c *projectGenConfig) {
		c.status = &status
	}
}

func WithTypes(types ...catalog.ProjectType) ProjectGenOpt {
	return func(c *projectGenConfig) {
		c.types = types
	}
}

func GenProject(t *rapid.T, dir string, opts ...ProjectGenOpt) catalog.Project {
	cfg := &projectGenConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var name string
	if cfg.name != nil {
		name = *cfg.name
	} else {
		name = validNameGen().Draw(t, "name")
	}

	subdir := subdirGen.Draw(t, "subdir")
	path := filepath.Join(dir, subdir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	p := catalog.NewProject(name, path)

	if cfg.types != nil {
		p = p.WithTypes(cfg.types...)
	} else {
		p = generateTypes(t, p)
	}

	if cfg.tags != nil {
		p = p.WithTags(cfg.tags...)
	} else {
		p = generateTags(t, p)
	}

	if cfg.status != nil {
		p = p.WithStatus(*cfg.status)
	} else {
		p = p.WithStatus(statusGen().Draw(t, "status"))
	}

	if cfg.notes != nil {
		p = p.WithNotes(*cfg.notes)
	} else if rapid.Bool().Draw(t, "hasNotes") {
		p = p.WithNotes(notesGen.Draw(t, "notes"))
	}

	return p
}

func generateTypes(t *rapid.T, p catalog.Project) catalog.Project {
	numTypes := rapid.IntRange(minTypes, maxTypes).Draw(t, "numTypes")
	types := make([]catalog.ProjectType, numTypes)
	for i := range numTypes {
		types[i] = projectTypeGen().Draw(t, "type")
	}
	p = p.WithTypes(types...)
	return p
}

func generateTags(t *rapid.T, p catalog.Project) catalog.Project {
	numTags := rapid.IntRange(minTags, maxTags).Draw(t, "numTags")
	tags := make([]string, numTags)
	for i := range numTags {
		tags[i] = tagGen().Draw(t, "tag")
	}
	p = p.WithTags(tags...)
	return p
}

type Harness struct {
	T   *rapid.T
	Dir string
}

func (h *Harness) GenProject(opts ...ProjectGenOpt) catalog.Project {
	return GenProject(h.T, h.Dir, opts...)
}

type CatalogHarness struct {
	Harness
	Catalog catalog.Catalog
}

func (h *CatalogHarness) MustAddProject(opts ...ProjectGenOpt) catalog.Project {
	p := h.GenProject(opts...)
	if err := h.Catalog.Add(p); err != nil {
		h.T.Fatalf("failed to add project: %v", err)
	}
	return p
}

func (h *CatalogHarness) AddProjects(minCount, maxCount int) []catalog.Project {
	var added []catalog.Project
	n := rapid.IntRange(minCount, maxCount).Draw(h.T, "numProjects")
	for range n {
		p := h.GenProject()
		if err := h.Catalog.Add(p); err == nil {
			added = append(added, p)
		}
	}
	return added
}

func RunWithCatalog(t *testing.T, fn func(h *CatalogHarness)) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		cat, err := catalog.NewYAMLCatalog(filepath.Join(iterDir, "catalog.yaml"))
		if err != nil {
			rt.Fatalf("failed to create catalog: %v", err)
		}

		harness := &CatalogHarness{
			Harness: Harness{
				T:   rt,
				Dir: iterDir,
			},
			Catalog: cat,
		}

		fn(harness)
	})
}

func RunBasic(t *testing.T, fn func(h *Harness)) {
	tempDir := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		iterDir := filepath.Join(tempDir, iterDirGen.Draw(rt, "iterDir"))
		if err := os.MkdirAll(iterDir, 0o755); err != nil {
			rt.Fatalf("failed to create iter dir: %v", err)
		}

		harness := &Harness{
			T:   rt,
			Dir: iterDir,
		}

		fn(harness)
	})
}
