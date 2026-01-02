package proptest

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestProperty_NewProject_Invariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validNameGen().Draw(t, "name")
		path := "/" + pathSegmentGen.Draw(t, "path")

		p := catalog.NewProject(name, path)

		if p.ID == "" {
			t.Fatal("INV-1: ID is empty")
		}
		if p.LastAccessed.Before(p.AddedAt) {
			t.Fatalf("INV-2: LastAccessed (%v) before AddedAt (%v)", p.LastAccessed, p.AddedAt)
		}
		if p.Types == nil {
			t.Fatal("INV-4: Types slice is nil")
		}
		if p.Tags == nil {
			t.Fatal("INV-5: Tags slice is nil")
		}
	})
}

func TestProperty_AddTagRemoveTag_Inverse(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		path := "/" + pathSegmentGen.Draw(rt, "path")
		tag := tagGen().Draw(rt, "tag")

		p := catalog.NewProject(name, path)
		originalTags := make([]string, len(p.Tags))
		copy(originalTags, p.Tags)

		if p.HasTag(tag) {
			rt.Skip("tag already exists")
		}

		p.AddTag(tag)
		if !p.HasTag(tag) {
			rt.Fatal("AddTag didn't add the tag")
		}

		p.RemoveTag(tag)
		if p.HasTag(tag) {
			rt.Fatal("RemoveTag didn't remove the tag")
		}

		if len(p.Tags) != len(originalTags) {
			rt.Fatalf("tag count mismatch: expected %d, got %d", len(originalTags), len(p.Tags))
		}
	})
}

func TestProperty_ValidateAndNormalize_Idempotent(t *testing.T) {
	RunBasic(t, func(h *Harness) {
		p := h.GenProject()

		err1 := p.ValidateAndNormalize()
		if err1 != nil {
			h.T.Skip("invalid project")
		}
		tagsAfterFirst := make([]string, len(p.Tags))
		copy(tagsAfterFirst, p.Tags)

		err2 := p.ValidateAndNormalize()
		if err2 != nil {
			h.T.Fatal("second validation failed but first succeeded")
		}

		if len(p.Tags) != len(tagsAfterFirst) {
			h.T.Fatalf("tag count changed: %d → %d", len(tagsAfterFirst), len(p.Tags))
		}
		for i, tag := range p.Tags {
			if tag != tagsAfterFirst[i] {
				h.T.Fatalf("tag[%d] changed: %q → %q", i, tagsAfterFirst[i], tag)
			}
		}
	})
}

func TestProperty_AddTag_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validNameGen().Draw(t, "name")
		path := "/" + pathSegmentGen.Draw(t, "path")
		tag := tagGen().Draw(t, "tag")

		p := catalog.NewProject(name, path)

		p.AddTag(tag)
		countAfterFirst := len(p.Tags)

		p.AddTag(tag)
		countAfterSecond := len(p.Tags)

		if countAfterFirst != countAfterSecond {
			t.Fatalf("AddTag not idempotent: %d → %d", countAfterFirst, countAfterSecond)
		}
	})
}

func TestProperty_WithTypes_PreservesOtherFields(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		name := validNameGen().Draw(rt, "name")
		path := "/" + pathSegmentGen.Draw(rt, "path")

		original := catalog.NewProject(name, path)
		original = original.WithStatus(statusGen().Draw(rt, "status"))
		original = original.WithNotes(rapid.String().Draw(rt, "notes"))

		newTypes := rapid.SliceOfN(projectTypeGen(), minTypes, maxTypes).Draw(rt, "newTypes")
		modified := original.WithTypes(newTypes...)

		assert.Equal(t, original.ID, modified.ID, "ID changed")
		assert.Equal(t, original.Name, modified.Name, "Name changed")
		assert.Equal(t, original.Path, modified.Path, "Path changed")
		assert.Equal(t, original.Status, modified.Status, "Status changed")
		assert.Equal(t, original.Notes, modified.Notes, "Notes changed")
		assert.Equal(t, original.AddedAt, modified.AddedAt, "AddedAt changed")
		assert.Equal(t, original.LastAccessed, modified.LastAccessed, "LastAccessed changed")
	})
}

func TestProperty_WithTypes_IsolatesSlices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validNameGen().Draw(t, "name")
		path := "/" + pathSegmentGen.Draw(t, "path")

		original := catalog.NewProject(name, path)
		original = original.WithTags("tag1", "tag2")

		modified := original.WithTypes(catalog.TypeGo, catalog.TypeRust)

		if len(modified.Tags) > 0 {
			modified.Tags[0] = "mutated"
		}
		if len(modified.Types) > 0 {
			modified.Types[0] = catalog.TypePython
		}

		if original.HasTag("mutated") {
			t.Fatal("modifying copy affected original Tags")
		}
		if original.HasType(catalog.TypePython) {
			t.Fatal("modifying copy affected original Types")
		}
	})
}

func TestProperty_EmptyName_Rejected(t *testing.T) {
	RunBasic(t, func(h *Harness) {
		emptyNames := []string{"", " ", "  ", "\t", "\n", " \t\n "}
		name := rapid.SampledFrom(emptyNames).Draw(h.T, "emptyName")

		path := filepath.Join(h.Dir, "subdir")
		os.MkdirAll(path, 0o755)

		p := catalog.NewProject(name, path)
		err := p.ValidateAndNormalize()

		if err == nil {
			h.T.Fatal("expected error for empty/whitespace name")
		}
		if !strings.Contains(err.Error(), "empty") {
			h.T.Fatalf("expected empty name error, got: %v", err)
		}
	})
}

func TestProperty_RelativePath_Rejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		relativePaths := []string{".", "..", "foo", "foo/bar", "./foo", "../bar"}
		path := rapid.SampledFrom(relativePaths).Draw(t, "relativePath")

		name := validNameGen().Draw(t, "name")
		p := catalog.NewProject(name, path)
		err := p.ValidateAndNormalize()

		if err == nil {
			t.Fatalf("expected error for relative path %q", path)
		}
	})
}

func TestProperty_UnicodeHandling(t *testing.T) {
	RunBasic(t, func(h *Harness) {
		unicodeNames := []string{"项目", "プロジェクト", "проект", "مشروع", "🚀rocket"}
		name := rapid.SampledFrom(unicodeNames).Draw(h.T, "unicodeName")

		path := filepath.Join(h.Dir, "subdir")
		os.MkdirAll(path, 0o755)

		p := catalog.NewProject(name, path)
		p = p.WithTags("标签", "タグ")

		err := p.ValidateAndNormalize()
		if err != nil {
			h.T.Fatalf("Unicode rejected: %v", err)
		}

		if p.Name != name {
			h.T.Fatalf("Unicode name corrupted: %q → %q", name, p.Name)
		}
		if !p.HasTag("标签") {
			h.T.Fatal("Unicode tag not found")
		}
	})
}

func TestProperty_HasTag_ConsistentWithTags(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := validNameGen().Draw(t, "name")
		path := "/" + pathSegmentGen.Draw(t, "path")
		tags := rapid.SliceOfN(tagGen(), minTags, maxTags).Draw(t, "tags")

		p := catalog.NewProject(name, path).WithTags(tags...)

		for _, tag := range p.Tags {
			if !p.HasTag(tag) {
				t.Fatalf("HasTag(%q) returned false but tag is in Tags", tag)
			}
		}

		nonExistent := "definitely-not-a-tag-" + numSuffixGen.Draw(t, "suffix")
		if p.HasTag(nonExistent) {
			t.Fatalf("HasTag(%q) returned true but tag not in Tags", nonExistent)
		}
	})
}
