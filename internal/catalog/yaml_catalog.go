package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	return false
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
		Projects: c.listUnlocked(),
	}

	slices.SortStableFunc(file.Projects, func(a, b Project) int {
		return strings.Compare(a.Name, b.Name)
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
