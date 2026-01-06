package proptest

import (
	"pj/internal/catalog"

	"pgregory.net/rapid"
)

const (
	InvProjectHasID           = "INV-1"
	InvLastAccessedAfterAdded = "INV-2"
	InvNameNotEmpty           = "INV-6"
	InvPathAbsolute           = "INV-7"
	InvUnicodePreserved       = "INV-8"
	InvCountEqualsListLen     = "INV-10"
	InvPathUnique             = "INV-11"
	InvAddRemoveRestoresCount = "INV-12"
	InvDuplicatePathRejected  = "INV-13"
	InvUpdatePathIndex        = "INV-14"
	InvGetByPathConsistent    = "INV-15"
	InvModelConsistent        = "INV-16"
	InvEmptySearchReturnsList = "INV-20"
	InvSearchSubsetOfList     = "INV-21"
	InvFilterSubsetOfList     = "INV-22"
	InvSearchCaseInsensitive  = "INV-23"
	InvSearchIsolated         = "INV-24"
	InvSortTransitive         = "INV-25"
	InvSortStable             = "INV-26"
	InvSaveLoadRoundTrip      = "INV-50"
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
	}
}
