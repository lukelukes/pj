package proptest

import (
	"pj/internal/catalog"
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

func TestProperty_Filter_SubsetOfList(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(typicalMinProjects, typicalMaxProjects)

		opts := filterOptionsGen().Draw(h.T, "filterOpts")
		filtered := h.Catalog.Filter(opts)
		allProjects := h.Catalog.List()

		assertSubset(h.T, filtered, allProjects)
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

func TestProperty_Filter_ByStatusAndType(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.MustAddProject(WithName("active-go"), WithStatus(catalog.StatusActive), WithTypes(catalog.TypeGo))
		h.MustAddProject(WithName("archived-go"), WithStatus(catalog.StatusArchived), WithTypes(catalog.TypeGo))
		h.MustAddProject(WithName("active-rust"), WithStatus(catalog.StatusActive), WithTypes(catalog.TypeRust))

		filtered := h.Catalog.Filter(catalog.FilterOptions{
			Status: catalog.StatusActive,
			Types:  []catalog.ProjectType{catalog.TypeGo},
		})

		if len(filtered) != 1 {
			h.T.Fatalf("expected 1 project, got %d", len(filtered))
		}
		if filtered[0].Name != "active-go" {
			h.T.Fatalf("expected active-go, got %s", filtered[0].Name)
		}
	})
}

func TestProperty_Sort_Transitivity(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(transitivityMinCount, typicalMaxProjects)

		if h.Catalog.Count() < transitivityMinCount {
			h.T.Skip("need at least 3 projects")
		}

		opts := catalog.FilterOptions{SortBy: catalog.SortByName}
		sorted := h.Catalog.Filter(opts)

		assertSortedBy(h.T, sorted, catalog.SortByName, false)
	})
}

func TestProperty_Sort_Stability(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		h.AddProjects(typicalMinProjects, typicalMaxProjects)

		sortFields := []catalog.SortField{catalog.SortByName, catalog.SortByPath, catalog.SortByAddedAt, catalog.SortByLastAccessed}
		sortBy := rapid.SampledFrom(sortFields).Draw(h.T, "sortBy")

		opts := catalog.FilterOptions{SortBy: sortBy}

		sorted1 := h.Catalog.Filter(opts)
		sorted2 := h.Catalog.Filter(opts)

		assertSameIDs(h.T, sorted1, sorted2)
		for i := range sorted1 {
			if sorted1[i].ID != sorted2[i].ID {
				h.T.Fatalf("sort not stable: position %d differs", i)
			}
		}
	})
}
