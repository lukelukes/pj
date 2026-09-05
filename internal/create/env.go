package create

import (
	"io/fs"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"strings"
)

type CatalogEnv struct {
	Cat catalog.Catalog
}

func (e CatalogEnv) Stat(path string) (fs.FileInfo, error) { return os.Stat(path) }

func (e CatalogEnv) CountEntries(path string) (int, error) {
	entries, err := os.ReadDir(path)
	return len(entries), err
}

func (e CatalogEnv) GitRoot(path string) string {
	dir := path
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func (e CatalogEnv) ByName(name string) (Conflict, bool) {
	if e.Cat == nil {
		return Conflict{}, false
	}
	for _, p := range e.Cat.List() {
		if strings.EqualFold(p.Name, name) {
			return Conflict{Name: p.Name, Path: p.Path, AddedAt: p.AddedAt}, true
		}
	}
	return Conflict{}, false
}

func (e CatalogEnv) ByPath(path string) (Conflict, bool) {
	if e.Cat == nil {
		return Conflict{}, false
	}
	p, err := e.Cat.GetByPath(path)
	if err != nil {
		return Conflict{}, false
	}
	return Conflict{Name: p.Name, Path: p.Path, AddedAt: p.AddedAt}, true
}
