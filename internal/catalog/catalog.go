package catalog

import "errors"

var (
	ErrNotFound      = errors.New("project not found")
	ErrAlreadyExists = errors.New("project already exists at path")
)

type Catalog interface {
	Add(p Project) error
	Get(id string) (Project, error)
	GetByPath(path string) (Project, error)
	Update(p Project) error
	Remove(id string) error
	List() []Project
	Search(query string) []Project
	Count() int
	Save() error
	Load() error
}
