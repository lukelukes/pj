package catalog_test

import (
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithTypes_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithTypes(catalog.TypeGo, catalog.TypeRust)

		assert.Empty(t, p1.Types)

		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo, catalog.TypeRust}, p2.Types)
	})

	t.Run("does not share slice memory", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		types := []catalog.ProjectType{catalog.TypeGo, catalog.TypeRust}
		p2 := p1.WithTypes(types...)

		types[0] = catalog.TypePython

		assert.Equal(t, catalog.TypeGo, p2.Types[0])
		assert.Equal(t, catalog.TypeRust, p2.Types[1])
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithTypes(catalog.TypeGo)
		p3 := p2.WithTypes(catalog.TypeRust)

		assert.Empty(t, p1.Types)
		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, p2.Types)
		assert.Equal(t, []catalog.ProjectType{catalog.TypeRust}, p3.Types)
	})
}

func TestWithTags_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithTags("work", "backend")

		assert.Empty(t, p1.Tags)

		assert.Equal(t, []string{"work", "backend"}, p2.Tags)
	})

	t.Run("does not share slice memory", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		tags := []string{"work", "backend"}
		p2 := p1.WithTags(tags...)

		tags[0] = "personal"

		assert.Equal(t, "work", p2.Tags[0])
		assert.Equal(t, "backend", p2.Tags[1])
	})

	t.Run("modifying returned project does not affect original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir).WithTags("a", "b")
		p2 := p1.WithTags("c", "d")

		assert.Equal(t, []string{"a", "b"}, p1.Tags)
		assert.Equal(t, []string{"c", "d"}, p2.Tags)

		p2.Tags[0] = "modified"

		assert.Equal(t, "a", p1.Tags[0])
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithTags("tag1")
		p3 := p2.WithTags("tag2")

		assert.Empty(t, p1.Tags)
		assert.Equal(t, []string{"tag1"}, p2.Tags)
		assert.Equal(t, []string{"tag2"}, p3.Tags)
	})
}

func TestWithStatus_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithStatus(catalog.StatusArchived)

		assert.Equal(t, catalog.StatusActive, p1.Status)

		assert.Equal(t, catalog.StatusArchived, p2.Status)
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithStatus(catalog.StatusArchived)
		p3 := p2.WithStatus(catalog.StatusAbandoned)

		assert.Equal(t, catalog.StatusActive, p1.Status)
		assert.Equal(t, catalog.StatusArchived, p2.Status)
		assert.Equal(t, catalog.StatusAbandoned, p3.Status)
	})
}

func TestWithNotes_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithNotes("Some notes")

		assert.Empty(t, p1.Notes)

		assert.Equal(t, "Some notes", p2.Notes)
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithNotes("First")
		p3 := p2.WithNotes("Second")

		assert.Empty(t, p1.Notes)
		assert.Equal(t, "First", p2.Notes)
		assert.Equal(t, "Second", p3.Notes)
	})
}

func TestWithGitRemote_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithGitRemote("https://github.com/user/repo.git")

		assert.Empty(t, p1.GitRemote)

		assert.Equal(t, "https://github.com/user/repo.git", p2.GitRemote)
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithGitRemote("https://github.com/user/repo1.git")
		p3 := p2.WithGitRemote("https://github.com/user/repo2.git")

		assert.Empty(t, p1.GitRemote)
		assert.Equal(t, "https://github.com/user/repo1.git", p2.GitRemote)
		assert.Equal(t, "https://github.com/user/repo2.git", p3.GitRemote)
	})
}

func TestWithDescription_Immutability(t *testing.T) {
	t.Run("returns new project without modifying original", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithDescription("A test project")

		assert.Empty(t, p1.Description)

		assert.Equal(t, "A test project", p2.Description)
	})

	t.Run("chaining creates independent copies", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithDescription("First description")
		p3 := p2.WithDescription("Second description")

		assert.Empty(t, p1.Description)
		assert.Equal(t, "First description", p2.Description)
		assert.Equal(t, "Second description", p3.Description)
	})
}

func TestBuilderMethods_Immutability_Parameterized(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name   string
		mutate func(catalog.Project) catalog.Project
		verify func(t *testing.T, original, mutated catalog.Project)
	}{
		{
			name:   "WithTypes",
			mutate: func(p catalog.Project) catalog.Project { return p.WithTypes(catalog.TypeGo) },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Empty(t, orig.Types)
				assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, mut.Types)
			},
		},
		{
			name:   "WithTags",
			mutate: func(p catalog.Project) catalog.Project { return p.WithTags("work") },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Empty(t, orig.Tags)
				assert.Equal(t, []string{"work"}, mut.Tags)
			},
		},
		{
			name:   "WithStatus",
			mutate: func(p catalog.Project) catalog.Project { return p.WithStatus(catalog.StatusArchived) },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Equal(t, catalog.StatusActive, orig.Status)
				assert.Equal(t, catalog.StatusArchived, mut.Status)
			},
		},
		{
			name:   "WithNotes",
			mutate: func(p catalog.Project) catalog.Project { return p.WithNotes("note") },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Empty(t, orig.Notes)
				assert.Equal(t, "note", mut.Notes)
			},
		},
		{
			name:   "WithGitRemote",
			mutate: func(p catalog.Project) catalog.Project { return p.WithGitRemote("https://github.com/test") },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Empty(t, orig.GitRemote)
				assert.Equal(t, "https://github.com/test", mut.GitRemote)
			},
		},
		{
			name:   "WithDescription",
			mutate: func(p catalog.Project) catalog.Project { return p.WithDescription("desc") },
			verify: func(t *testing.T, orig, mut catalog.Project) {
				assert.Empty(t, orig.Description)
				assert.Equal(t, "desc", mut.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" preserves original", func(t *testing.T) {
			original := catalog.NewProject("test", dir)
			mutated := tt.mutate(original)
			tt.verify(t, original, mutated)
		})
	}
}

func TestBuilderPattern_ComplexChaining(t *testing.T) {
	t.Run("complex chaining with all builders", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.
			WithTypes(catalog.TypeGo).
			WithTags("work", "backend").
			WithStatus(catalog.StatusActive).
			WithNotes("Active project").
			WithGitRemote("https://github.com/user/test.git").
			WithDescription("Test description")

		assert.Empty(t, p1.Types)
		assert.Empty(t, p1.Tags)
		assert.Equal(t, catalog.StatusActive, p1.Status)
		assert.Empty(t, p1.Notes)
		assert.Empty(t, p1.GitRemote)
		assert.Empty(t, p1.Description)

		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, p2.Types)
		assert.Equal(t, []string{"work", "backend"}, p2.Tags)
		assert.Equal(t, catalog.StatusActive, p2.Status)
		assert.Equal(t, "Active project", p2.Notes)
		assert.Equal(t, "https://github.com/user/test.git", p2.GitRemote)
		assert.Equal(t, "Test description", p2.Description)
	})

	t.Run("modifying intermediate result does not affect final", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir)
		p2 := p1.WithTags("tag1", "tag2")
		p3 := p2.WithTypes(catalog.TypeGo)

		p2.Tags[0] = "modified"

		assert.Equal(t, "tag1", p3.Tags[0])
		assert.Equal(t, "tag2", p3.Tags[1])
	})
}

func TestBuilderPattern_EdgeCases(t *testing.T) {
	t.Run("WithTypes with empty variadic args creates empty slice", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir).WithTypes(catalog.TypeGo)
		p2 := p1.WithTypes()

		assert.Len(t, p1.Types, 1)

		assert.Empty(t, p2.Types)
	})

	t.Run("WithTags with empty variadic args creates empty slice", func(t *testing.T) {
		dir := t.TempDir()
		p1 := catalog.NewProject("test", dir).WithTags("tag1")
		p2 := p1.WithTags()

		assert.Len(t, p1.Tags, 1)

		assert.Empty(t, p2.Tags)
	})

	t.Run("multiple WithTypes calls create independent slices", func(t *testing.T) {
		dir := t.TempDir()
		base := catalog.NewProject("test", dir)
		p1 := base.WithTypes(catalog.TypeGo)
		p2 := base.WithTypes(catalog.TypeRust)

		assert.Equal(t, []catalog.ProjectType{catalog.TypeGo}, p1.Types)
		assert.Equal(t, []catalog.ProjectType{catalog.TypeRust}, p2.Types)
	})
}
