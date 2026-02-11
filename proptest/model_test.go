package proptest

import (
	"pj/internal/catalog"
	"slices"

	"pgregory.net/rapid"
)

type StateTracker struct {
	idToPath map[string]string
	pathToID map[string]string
}

func newStateTracker() *StateTracker {
	return &StateTracker{
		idToPath: make(map[string]string),
		pathToID: make(map[string]string),
	}
}

func (s *StateTracker) Add(p catalog.Project) error {
	if _, exists := s.pathToID[p.Path]; exists {
		return catalog.ErrAlreadyExists
	}
	s.idToPath[p.ID] = p.Path
	s.pathToID[p.Path] = p.ID
	return nil
}

func (s *StateTracker) Remove(id string) error {
	path, ok := s.idToPath[id]
	if !ok {
		return catalog.ErrNotFound
	}
	delete(s.idToPath, id)
	delete(s.pathToID, path)
	return nil
}

func (s *StateTracker) Exists(id string) bool {
	_, ok := s.idToPath[id]
	return ok
}

func (s *StateTracker) Update(p catalog.Project) error {
	oldPath, ok := s.idToPath[p.ID]
	if !ok {
		return catalog.ErrNotFound
	}
	if oldPath != p.Path {
		delete(s.pathToID, oldPath)
		s.pathToID[p.Path] = p.ID
	}
	s.idToPath[p.ID] = p.Path
	return nil
}

func (s *StateTracker) IDs() []string {
	ids := make([]string, 0, len(s.idToPath))
	for id := range s.idToPath {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (s *StateTracker) Count() int {
	return len(s.idToPath)
}

func (s *StateTracker) PathExists(path string) bool {
	_, ok := s.pathToID[path]
	return ok
}

type CheckedCatalog struct {
	real  catalog.Catalog
	model *StateTracker
	t     *rapid.T
}

func NewCheckedCatalog(t *rapid.T, cat catalog.Catalog) *CheckedCatalog {
	return &CheckedCatalog{
		real:  cat,
		model: newStateTracker(),
		t:     t,
	}
}

func (c *CheckedCatalog) Model() *StateTracker {
	return c.model
}

func (c *CheckedCatalog) Add(p catalog.Project) error {
	realErr := c.real.Add(p)
	modelErr := c.model.Add(p)
	if (realErr == nil) != (modelErr == nil) {
		c.t.Fatalf("Add divergence: real=%v model=%v", realErr, modelErr)
	}
	verifyStructuralInvariants(c.t, c.real)
	return realErr
}

func (c *CheckedCatalog) Remove(id string) error {
	realErr := c.real.Remove(id)
	modelErr := c.model.Remove(id)
	if (realErr == nil) != (modelErr == nil) {
		c.t.Fatalf("Remove divergence: real=%v model=%v", realErr, modelErr)
	}
	verifyStructuralInvariants(c.t, c.real)
	return realErr
}

func (c *CheckedCatalog) Get(id string) (catalog.Project, error) {
	realProject, realErr := c.real.Get(id)
	modelExists := c.model.Exists(id)
	if (realErr == nil) != modelExists {
		c.t.Fatalf("Get divergence: real err=%v model exists=%v", realErr, modelExists)
	}
	return realProject, realErr
}

func (c *CheckedCatalog) Update(p catalog.Project) error {
	realErr := c.real.Update(p)
	modelErr := c.model.Update(p)
	if (realErr == nil) != (modelErr == nil) {
		c.t.Fatalf("Update divergence: real=%v model=%v", realErr, modelErr)
	}
	verifyStructuralInvariants(c.t, c.real)
	return realErr
}

func (c *CheckedCatalog) GetByPath(path string) (catalog.Project, error) {
	realProject, realErr := c.real.GetByPath(path)
	modelExists := c.model.PathExists(path)
	if (realErr == nil) != modelExists {
		c.t.Fatalf("GetByPath divergence: real err=%v model exists=%v", realErr, modelExists)
	}
	return realProject, realErr
}

func (c *CheckedCatalog) List() []catalog.Project {
	realList := c.real.List()
	verifyStructuralInvariants(c.t, c.real)
	return realList
}

func (c *CheckedCatalog) Search(query string) []catalog.Project {
	realResults := c.real.Search(query)
	allProjects := c.real.List()
	allIDs := make(map[string]bool)
	for _, p := range allProjects {
		allIDs[p.ID] = true
	}
	for _, p := range realResults {
		if !allIDs[p.ID] {
			c.t.Fatalf("Search(%q) returned project %s not in List()", query, p.ID)
		}
	}
	return realResults
}

func (c *CheckedCatalog) Filter(opts catalog.FilterOptions) []catalog.Project {
	realResults := c.real.Filter(opts)
	return realResults
}
