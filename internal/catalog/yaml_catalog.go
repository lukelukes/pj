package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type catalogFile struct {
	Version  int       `yaml:"version"`
	Projects []Project `yaml:"projects"`
}

type YAMLCatalog struct {
	path     string
	projects map[string]Project
	byPath   map[string]string
	mu       sync.RWMutex
}

func NewYAMLCatalog(path string) (*YAMLCatalog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	return &YAMLCatalog{
		path:     path,
		projects: make(map[string]Project),
		byPath:   make(map[string]string),
	}, nil
}

func (c *YAMLCatalog) Add(p Project) error {
	if err := p.ValidateAndNormalize(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.byPath[p.Path]; exists {
		return ErrAlreadyExists
	}

	c.projects[p.ID] = p
	c.byPath[p.Path] = p.ID
	return nil
}

func (c *YAMLCatalog) Get(id string) (Project, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}

func (c *YAMLCatalog) GetByPath(path string) (Project, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	id, ok := c.byPath[path]
	if !ok {
		return Project{}, ErrNotFound
	}
	return c.projects[id], nil
}

func (c *YAMLCatalog) Update(p Project) error {
	if err := p.ValidateAndNormalize(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.projects[p.ID]
	if !ok {
		return ErrNotFound
	}

	if existing.Path != p.Path {
		if existingID, exists := c.byPath[p.Path]; exists && existingID != p.ID {
			return ErrAlreadyExists
		}
		delete(c.byPath, existing.Path)
		c.byPath[p.Path] = p.ID
	}

	c.projects[p.ID] = p
	return nil
}

func (c *YAMLCatalog) Remove(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.projects[id]
	if !ok {
		return ErrNotFound
	}

	delete(c.projects, id)
	delete(c.byPath, p.Path)
	return nil
}

func (c *YAMLCatalog) List() []Project {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.listUnlocked()
}

func (c *YAMLCatalog) listUnlocked() []Project {
	projects := make([]Project, 0, len(c.projects))
	for _, p := range c.projects {
		projects = append(projects, p)
	}
	return projects
}

func (c *YAMLCatalog) Search(query string) []Project {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if query == "" {
		return c.listUnlocked()
	}

	query = strings.ToLower(query)
	var results []Project

	for _, p := range c.projects {
		if matchesQuery(p, query) {
			results = append(results, p)
		}
	}

	return results
}

func matchesQuery(p Project, query string) bool {
	if strings.Contains(strings.ToLower(p.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Path), query) {
		return true
	}
	for _, tag := range p.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func (c *YAMLCatalog) Filter(opts FilterOptions) []Project {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []Project
	for _, p := range c.projects {
		if matchesFilter(p, opts) {
			results = append(results, p)
		}
	}

	sortProjects(results, opts.SortBy, opts.Descending)
	return results
}

func matchesFilter(p Project, opts FilterOptions) bool {
	if opts.Status != "" && p.Status != opts.Status {
		return false
	}

	if len(opts.Types) > 0 && !slices.ContainsFunc(opts.Types, p.HasType) {
		return false
	}

	for _, tag := range opts.Tags {
		if !p.HasTag(tag) {
			return false
		}
	}

	if opts.Query != "" && !matchesQuery(p, strings.ToLower(opts.Query)) {
		return false
	}

	return true
}

func sortProjects(projects []Project, by SortField, descending bool) {
	if by == "" {
		by = SortByName
	}

	sort.SliceStable(projects, func(i, j int) bool {
		less := compareProjects(projects[i], projects[j], by)
		if descending {
			return !less
		}
		return less
	})
}

func compareProjects(a, b Project, by SortField) bool {
	var less, equal bool

	switch by {
	case SortByPath:
		less, equal = a.Path < b.Path, a.Path == b.Path
	case SortByLastAccessed:
		less = a.LastAccessed.Before(b.LastAccessed)
		equal = a.LastAccessed.Equal(b.LastAccessed)
	case SortByAddedAt:
		less = a.AddedAt.Before(b.AddedAt)
		equal = a.AddedAt.Equal(b.AddedAt)
	case SortByTypes:
		t1, t2 := "", ""
		if len(a.Types) > 0 {
			t1 = string(a.Types[0])
		}
		if len(b.Types) > 0 {
			t2 = string(b.Types[0])
		}
		less, equal = t1 < t2, t1 == t2
	default:
		n1, n2 := strings.ToLower(a.Name), strings.ToLower(b.Name)
		less, equal = n1 < n2, n1 == n2
	}

	// Tiebreaker: use ID for deterministic ordering when primary key is equal
	if equal {
		return a.ID < b.ID
	}
	return less
}

func (c *YAMLCatalog) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.projects)
}

func (c *YAMLCatalog) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	file := catalogFile{
		Version:  1,
		Projects: make([]Project, 0, len(c.projects)),
	}

	for _, p := range c.projects {
		file.Projects = append(file.Projects, p)
	}

	sort.SliceStable(file.Projects, func(i, j int) bool {
		return file.Projects[i].Name < file.Projects[j].Name
	})

	data, err := yaml.Marshal(file)
	if err != nil {
		return err
	}

	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, c.path)
}

func (c *YAMLCatalog) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read catalog file: %w", err)
	}

	var file catalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse catalog file %q: %w", c.path, err)
	}

	c.projects = make(map[string]Project, len(file.Projects))
	c.byPath = make(map[string]string, len(file.Projects))

	for _, p := range file.Projects {
		c.projects[p.ID] = p
		c.byPath[p.Path] = p.ID
	}

	return nil
}
