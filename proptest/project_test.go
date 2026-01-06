package proptest

import (
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"strings"
	"testing"

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

		err := p.ValidateAndNormalize()
		if err != nil {
			h.T.Fatalf("Unicode rejected: %v", err)
		}

		if p.Name != name {
			h.T.Fatalf("Unicode name corrupted: %q → %q", name, p.Name)
		}
	})
}
