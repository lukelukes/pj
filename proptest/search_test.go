package proptest

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestProperty_EmptySearch_ReturnsList(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(minProjects, typicalMaxProjects)

		list := h.Catalog.List()
		search := h.Catalog.Search("")

		assertSameIDs(h.T, list, search)
	})
}

func TestProperty_Search_SubsetOfList(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(typicalMinProjects, typicalMaxProjects)

		query := shortQueryGen.Draw(h.T, "query")
		searchResults := h.Catalog.Search(query)
		allProjects := h.Catalog.List()

		assertSubset(h.T, searchResults, allProjects)
	})
}

func TestProperty_Search_CaseInsensitive(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(typicalMinProjects, typicalMaxProjects)

		query := shortQueryGen.Draw(h.T, "query")
		lowerResults := h.Catalog.Search(query)
		upperResults := h.Catalog.Search(strings.ToUpper(query))

		assertSameIDs(h.T, lowerResults, upperResults)
	})
}

func TestProperty_Search_UnrelatedProjectsNoEffect(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		knownProject := h.MustAddProject(WithName("searchable"))

		resultsBefore := h.Catalog.Search("searchable")
		if len(resultsBefore) != 1 {
			h.T.Fatalf("expected 1 result for 'searchable', got %d", len(resultsBefore))
		}

		for range rapid.IntRange(minUnrelatedProjects, maxUnrelatedProjects).Draw(h.T, "numUnrelated") {
			unrelatedName := rapid.StringMatching(`[xyz]{5,10}`).Draw(h.T, "unrelatedName")
			_ = h.Catalog.Add(h.GenProject(WithName(unrelatedName)))
		}

		resultsAfter := h.Catalog.Search("searchable")
		if len(resultsAfter) != 1 {
			h.T.Fatalf("expected 1 result for 'searchable' after adding unrelated, got %d", len(resultsAfter))
		}
		if resultsAfter[0].ID != knownProject.ID {
			h.T.Fatalf("search returned wrong project after adding unrelated")
		}
	})
}
