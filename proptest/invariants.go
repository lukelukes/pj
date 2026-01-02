package proptest

import (
	"pj/internal/catalog"

	"pgregory.net/rapid"
)

const (
	// Structural Invariants - Properties of individual projects

	// InvProjectHasID ensures every project has a non-empty ID.
	InvProjectHasID = "INV-1"

	// InvLastAccessedAfterAdded ensures LastAccessed is never before AddedAt.
	InvLastAccessedAfterAdded = "INV-2"

	// InvTypesNotNil ensures the Types slice is never nil (may be empty).
	InvTypesNotNil = "INV-4"

	// InvTagsNotNil ensures the Tags slice is never nil (may be empty).
	InvTagsNotNil = "INV-5"

	// InvNameNotEmpty ensures project names are non-empty after trimming whitespace.
	InvNameNotEmpty = "INV-6"

	// InvPathAbsolute ensures project paths are absolute, not relative.
	InvPathAbsolute = "INV-7"

	// InvUnicodePreserved ensures Unicode characters in names and tags are preserved.
	InvUnicodePreserved = "INV-8"

	// CRUD Invariants - Properties of catalog operations

	// InvCountEqualsListLen ensures Count() always equals len(List()).
	InvCountEqualsListLen = "INV-10"

	// InvPathUnique ensures no two projects share the same path.
	InvPathUnique = "INV-11"

	// InvAddRemoveRestoresCount ensures Add followed by Remove restores the original count.
	InvAddRemoveRestoresCount = "INV-12"

	// InvDuplicatePathRejected ensures adding a project with an existing path returns ErrAlreadyExists.
	InvDuplicatePathRejected = "INV-13"

	// InvUpdatePathIndex ensures updating a project's path updates the path index correctly.
	InvUpdatePathIndex = "INV-14"

	// InvGetByPathConsistent ensures Get(id) and GetByPath(path) return the same project.
	InvGetByPathConsistent = "INV-15"

	// InvModelConsistent ensures real catalog and reference model produce identical results.
	InvModelConsistent = "INV-16"

	// Query Invariants - Properties of search and filter operations

	// InvEmptySearchReturnsList ensures Search("") returns the same projects as List().
	InvEmptySearchReturnsList = "INV-20"

	// InvSearchSubsetOfList ensures Search results are always a subset of List().
	InvSearchSubsetOfList = "INV-21"

	// InvFilterSubsetOfList ensures Filter results are always a subset of List().
	InvFilterSubsetOfList = "INV-22"

	// InvSearchCaseInsensitive ensures search is case-insensitive.
	InvSearchCaseInsensitive = "INV-23"

	// InvSearchIsolated ensures unrelated projects don't affect search results.
	InvSearchIsolated = "INV-24"

	// InvSortTransitive ensures sorted results maintain proper ordering throughout.
	InvSortTransitive = "INV-25"

	// InvSortStable ensures repeated sorts produce identical ordering.
	InvSortStable = "INV-26"

	// Tag Invariants - Properties of tag operations

	// InvAddTagRemoveTagInverse ensures AddTag followed by RemoveTag is an inverse operation.
	InvAddTagRemoveTagInverse = "INV-30"

	// InvAddTagIdempotent ensures AddTag with the same tag twice doesn't duplicate.
	InvAddTagIdempotent = "INV-31"

	// InvHasTagConsistent ensures HasTag returns true iff tag is in Tags slice.
	InvHasTagConsistent = "INV-32"

	// Builder Invariants - Properties of With* methods

	// InvWithTypesPreservesFields ensures WithTypes preserves all other project fields.
	InvWithTypesPreservesFields = "INV-40"

	// InvWithTypesIsolatesSlices ensures modifying the returned project doesn't affect the original.
	InvWithTypesIsolatesSlices = "INV-41"

	// Serialization Invariants - Properties of Save/Load operations

	// InvSaveLoadRoundTrip ensures Save followed by Load preserves all project data.
	InvSaveLoadRoundTrip = "INV-50"

	// Idempotency Invariants - Operations that produce same result when repeated

	// InvValidateNormalizeIdempotent ensures ValidateAndNormalize is idempotent.
	InvValidateNormalizeIdempotent = "INV-60"
)

func verifyStructuralInvariants(t *rapid.T, cat catalog.Catalog) {
	count := cat.Count()
	list := cat.List()
	listLen := len(list)

	if count != listLen {
		t.Fatalf("[%s] violated: Count()=%d but len(List())=%d", InvCountEqualsListLen, count, listLen)
	}

	pathsSeen := make(map[string]bool)
	for _, p := range list {
		if pathsSeen[p.Path] {
			t.Fatalf("[%s] violated: duplicate path %q found in List()", InvPathUnique, p.Path)
		}
		pathsSeen[p.Path] = true

		if p.ID == "" {
			t.Fatalf("[%s] violated: project has empty ID", InvProjectHasID)
		}

		if p.Types == nil {
			t.Fatalf("[%s] violated: project %q has nil Types slice", InvTypesNotNil, p.ID)
		}

		if p.Tags == nil {
			t.Fatalf("[%s] violated: project %q has nil Tags slice", InvTagsNotNil, p.ID)
		}
	}
}
