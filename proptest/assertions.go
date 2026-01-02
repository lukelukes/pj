package proptest

import (
	"pj/internal/catalog"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func assertProjectsEqual(t *rapid.T, expected, actual catalog.Project) {
	t.Helper()
	opts := cmp.Options{
		cmpopts.EquateApproxTime(0),
		cmpopts.EquateEmpty(),
	}
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Fatalf("project mismatch (-want +got):\n%s", diff)
	}
}

func assertSameIDs(t *rapid.T, expected, actual []catalog.Project) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("length mismatch: expected %d, got %d", len(expected), len(actual))
	}
	expectedIDs := make(map[string]bool)
	for _, p := range expected {
		expectedIDs[p.ID] = true
	}
	for _, p := range actual {
		if !expectedIDs[p.ID] {
			t.Fatalf("unexpected ID %s in actual", p.ID)
		}
	}
}

func assertSubset(t *rapid.T, subset, superset []catalog.Project) {
	t.Helper()
	superIDs := make(map[string]bool)
	for _, p := range superset {
		superIDs[p.ID] = true
	}
	for _, p := range subset {
		if !superIDs[p.ID] {
			t.Fatalf("subset contains ID %s not in superset", p.ID)
		}
	}
}

func assertSortedBy(t *rapid.T, projects []catalog.Project, field catalog.SortField, desc bool) {
	t.Helper()
	for i := 0; i < len(projects)-1; i++ {
		a, b := projects[i], projects[i+1]
		var inOrder bool
		switch field {
		case catalog.SortByName:
			inOrder = strings.ToLower(a.Name) <= strings.ToLower(b.Name)
		case catalog.SortByPath:
			inOrder = a.Path <= b.Path
		case catalog.SortByAddedAt:
			inOrder = !a.AddedAt.After(b.AddedAt)
		case catalog.SortByLastAccessed:
			inOrder = !a.LastAccessed.After(b.LastAccessed)
		default:
			inOrder = strings.ToLower(a.Name) <= strings.ToLower(b.Name)
		}
		if desc {
			inOrder = !inOrder || a.ID == b.ID
		}
		if !inOrder {
			t.Fatalf("sort order violated at positions %d, %d", i, i+1)
		}
	}
}

func assertNoDuplicatePaths(t *rapid.T, projects []catalog.Project) {
	t.Helper()
	paths := make(map[string]bool)
	for _, p := range projects {
		if paths[p.Path] {
			t.Fatalf("duplicate path found: %s", p.Path)
		}
		paths[p.Path] = true
	}
}
