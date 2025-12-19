package catalog_test

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLCatalog_Add(t *testing.T) {
	t.Run("adds new project successfully", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)

		p := catalog.NewProject("myproject", dir)
		err := cat.Add(p)

		require.NoError(t, err)
		assert.Equal(t, 1, cat.Count())
	})

	t.Run("returns error when project at path already exists", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)

		p1 := catalog.NewProject("project1", dir)
		p2 := catalog.NewProject("project2", dir)

		require.NoError(t, cat.Add(p1))
		err := cat.Add(p2)

		assert.ErrorIs(t, err, catalog.ErrAlreadyExists)
	})
}

func TestYAMLCatalog_Get(t *testing.T) {
	t.Run("retrieves project by ID", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)
		p := catalog.NewProject("myproject", dir)
		require.NoError(t, cat.Add(p))

		got, err := cat.Get(p.ID)

		require.NoError(t, err)
		assert.Equal(t, p.Name, got.Name)
		assert.Equal(t, p.Path, got.Path)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		_, err := cat.Get("nonexistent")

		assert.ErrorIs(t, err, catalog.ErrNotFound)
	})
}

func TestYAMLCatalog_GetByPath(t *testing.T) {
	t.Run("retrieves project by path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)
		p := catalog.NewProject("myproject", dir)
		require.NoError(t, cat.Add(p))

		got, err := cat.GetByPath(dir)

		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("returns error when path not found", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		_, err := cat.GetByPath("/nonexistent/path")

		assert.ErrorIs(t, err, catalog.ErrNotFound)
	})
}

func TestYAMLCatalog_Update(t *testing.T) {
	t.Run("updates existing project", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)
		p := catalog.NewProject("myproject", dir)
		require.NoError(t, cat.Add(p))

		p.Notes = "Updated notes"
		err := cat.Update(p)

		require.NoError(t, err)
		got, _ := cat.Get(p.ID)
		assert.Equal(t, "Updated notes", got.Notes)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)
		p := catalog.NewProject("myproject", dir)

		err := cat.Update(p)

		assert.ErrorIs(t, err, catalog.ErrNotFound)
	})

	t.Run("returns error when updating to existing path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir1 := newTestDir(t)
		dir2 := newTestDir(t)
		p1 := catalog.NewProject("p1", dir1)
		p2 := catalog.NewProject("p2", dir2)
		require.NoError(t, cat.Add(p1))
		require.NoError(t, cat.Add(p2))

		p1.Path = dir2
		err := cat.Update(p1)

		assert.ErrorIs(t, err, catalog.ErrAlreadyExists)
	})

	t.Run("allows updating path to unused path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir1 := newTestDir(t)
		dir2 := newTestDir(t)
		p := catalog.NewProject("myproject", dir1)
		require.NoError(t, cat.Add(p))

		p.Path = dir2
		err := cat.Update(p)

		require.NoError(t, err)
		got, _ := cat.Get(p.ID)
		assert.Equal(t, dir2, got.Path)
	})
}

func TestYAMLCatalog_Remove(t *testing.T) {
	t.Run("removes existing project", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		dir := newTestDir(t)
		p := catalog.NewProject("myproject", dir)
		require.NoError(t, cat.Add(p))

		err := cat.Remove(p.ID)

		require.NoError(t, err)
		assert.Equal(t, 0, cat.Count())
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		err := cat.Remove("nonexistent")

		assert.ErrorIs(t, err, catalog.ErrNotFound)
	})
}

func TestYAMLCatalog_List(t *testing.T) {
	t.Run("returns all projects", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		require.NoError(t, cat.Add(catalog.NewProject("p1", newTestDir(t))))
		require.NoError(t, cat.Add(catalog.NewProject("p2", newTestDir(t))))
		require.NoError(t, cat.Add(catalog.NewProject("p3", newTestDir(t))))

		projects := cat.List()

		assert.Len(t, projects, 3)
	})

	t.Run("returns empty slice when catalog is empty", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		projects := cat.List()

		assert.Empty(t, projects)
	})
}

func TestYAMLCatalog_Search(t *testing.T) {
	t.Run("matches project name", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		require.NoError(t, cat.Add(catalog.NewProject("booster", newTestDir(t))))
		require.NoError(t, cat.Add(catalog.NewProject("rocket", newTestDir(t))))

		results := cat.Search("boost")

		assert.Len(t, results, 1)
		assert.Equal(t, "booster", results[0].Name)
	})

	t.Run("matches project path", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		tempDir := t.TempDir()
		workDir := filepath.Join(tempDir, "work", "client")
		personalDir := filepath.Join(tempDir, "personal", "other")
		require.NoError(t, os.MkdirAll(workDir, 0o755))
		require.NoError(t, os.MkdirAll(personalDir, 0o755))

		require.NoError(t, cat.Add(catalog.NewProject("project", workDir)))
		require.NoError(t, cat.Add(catalog.NewProject("other", personalDir)))

		results := cat.Search("work")

		assert.Len(t, results, 1)
		assert.Equal(t, "project", results[0].Name)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		require.NoError(t, cat.Add(catalog.NewProject("MyProject", newTestDir(t))))

		results := cat.Search("myproject")

		assert.Len(t, results, 1)
	})

	t.Run("returns empty when no matches", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		require.NoError(t, cat.Add(catalog.NewProject("project", newTestDir(t))))

		results := cat.Search("nonexistent")

		assert.Empty(t, results)
	})
}

func TestYAMLCatalog_Filter(t *testing.T) {
	t.Run("filters by status", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p1 := catalog.NewProject("active", newTestDir(t))
		p2 := catalog.NewProject("archived", newTestDir(t))
		p2.Status = catalog.StatusArchived
		require.NoError(t, cat.Add(p1))
		require.NoError(t, cat.Add(p2))

		results := cat.Filter(catalog.FilterOptions{Status: catalog.StatusActive})

		assert.Len(t, results, 1)
		assert.Equal(t, "active", results[0].Name)
	})

	t.Run("filters by type", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p1 := catalog.NewProject("goproject", newTestDir(t)).WithTypes(catalog.TypeGo)
		p2 := catalog.NewProject("rustproject", newTestDir(t)).WithTypes(catalog.TypeRust)
		require.NoError(t, cat.Add(p1))
		require.NoError(t, cat.Add(p2))

		results := cat.Filter(catalog.FilterOptions{Types: []catalog.ProjectType{catalog.TypeGo}})

		assert.Len(t, results, 1)
		assert.Equal(t, "goproject", results[0].Name)
	})

	t.Run("filters by tags (all must match)", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)
		p1 := catalog.NewProject("p1", newTestDir(t)).WithTags("work", "frontend")
		p2 := catalog.NewProject("p2", newTestDir(t)).WithTags("work", "backend")
		p3 := catalog.NewProject("p3", newTestDir(t)).WithTags("personal")
		require.NoError(t, cat.Add(p1))
		require.NoError(t, cat.Add(p2))
		require.NoError(t, cat.Add(p3))

		results := cat.Filter(catalog.FilterOptions{Tags: []string{"work", "frontend"}})

		assert.Len(t, results, 1)
		assert.Equal(t, "p1", results[0].Name)
	})
}

func TestYAMLCatalog_Persistence(t *testing.T) {
	t.Run("save and load preserves projects", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		projectDir := filepath.Join(dir, "myproject")
		require.NoError(t, os.Mkdir(projectDir, 0o755))

		cat1, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		p := catalog.NewProject("myproject", projectDir).
			WithTypes(catalog.TypeGo).
			WithTags("work", "tools").
			WithNotes("Important project")
		require.NoError(t, cat1.Add(p))
		require.NoError(t, cat1.Save())

		cat2, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		require.NoError(t, cat2.Load())

		got, err := cat2.Get(p.ID)
		require.NoError(t, err)
		assert.Equal(t, p.Name, got.Name)
		assert.Equal(t, p.Path, got.Path)
		assert.Equal(t, p.Types, got.Types)
		assert.Equal(t, p.Tags, got.Tags)
		assert.Equal(t, p.Notes, got.Notes)
	})

	t.Run("load creates empty catalog if file doesn't exist", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nonexistent.yaml")

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.NoError(t, err)
		assert.Equal(t, 0, cat.Count())
	})
}

func TestYAMLCatalog_LoadMalformedYAML(t *testing.T) {
	t.Run("returns error for unclosed string", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		malformedYAML := `version: 1
projects:
  - id: abc123
    name: "unclosed string
    path: /home/user/project
`
		require.NoError(t, os.WriteFile(path, []byte(malformedYAML), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse catalog file")
		assert.Contains(t, err.Error(), path)
	})

	t.Run("returns error for invalid indentation", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		malformedYAML := `version: 1
projects:
  - id: abc123
  name: badindent
    path: /home/user/project
`
		require.NoError(t, os.WriteFile(path, []byte(malformedYAML), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse catalog file")
	})

	t.Run("returns error for wrong type - string where array expected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		malformedYAML := `version: 1
projects: "this should be an array"
`
		require.NoError(t, os.WriteFile(path, []byte(malformedYAML), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse catalog file")
	})

	t.Run("returns error for invalid YAML syntax", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		malformedYAML := `version: 1
projects:
  - id: test
    <<: invalid anchor reference
    name: test
`
		require.NoError(t, os.WriteFile(path, []byte(malformedYAML), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse catalog file")
	})

	t.Run("returns error with file path in message", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "my-catalog.yaml")

		malformedYAML := `invalid: [unclosed array`
		require.NoError(t, os.WriteFile(path, []byte(malformedYAML), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.Error(t, err)

		assert.Contains(t, err.Error(), "my-catalog.yaml")
	})

	t.Run("handles empty file gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.NoError(t, err)
		assert.Equal(t, 0, cat.Count())
	})

	t.Run("handles file with only newlines gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "catalog.yaml")

		require.NoError(t, os.WriteFile(path, []byte("\n\n\n"), 0o644))

		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(t, err)
		err = cat.Load()

		require.NoError(t, err)
		assert.Equal(t, 0, cat.Count())
	})
}

func TestYAMLCatalog_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent read operations safely", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		p1 := catalog.NewProject("project1", newTestDir(t)).WithTags("work", "frontend")
		p2 := catalog.NewProject("project2", newTestDir(t)).WithTypes(catalog.TypeGo)
		p3 := catalog.NewProject("project3", newTestDir(t)).WithTags("personal")
		require.NoError(t, cat.Add(p1))
		require.NoError(t, cat.Add(p2))
		require.NoError(t, cat.Add(p3))

		const numGoroutines = 100
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()

				switch id % 4 {
				case 0:
					results := cat.Search("project")
					assert.NotEmpty(t, results)
				case 1:
					projects := cat.List()
					assert.Len(t, projects, 3)
				case 2:
					count := cat.Count()
					assert.Equal(t, 3, count)
				case 3:
					results := cat.Filter(catalog.FilterOptions{Tags: []string{"work"}})
					assert.NotEmpty(t, results)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 3, cat.Count())
		assert.Len(t, cat.List(), 3)
	})

	t.Run("handles mixed concurrent read and write operations safely", func(t *testing.T) {
		cat := newTestYAMLCatalog(t)

		p := catalog.NewProject("initial", newTestDir(t))
		require.NoError(t, cat.Add(p))

		const numGoroutines = 100
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()

				switch id % 5 {
				case 0, 1, 2:

					cat.Search("initial")
				case 3:

					cat.Get(p.ID)
				case 4:

					cat.List()
				}
			}(i)
		}

		wg.Wait()

		assert.GreaterOrEqual(t, cat.Count(), 1)
		_, err := cat.Get(p.ID)
		assert.NoError(t, err)
	})
}

func newTestYAMLCatalog(t *testing.T) *catalog.YAMLCatalog {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-catalog.yaml")
	cat, err := catalog.NewYAMLCatalog(path)
	require.NoError(t, err)
	return cat
}

func newTestDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}
