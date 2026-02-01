package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyName    = errors.New("project name cannot be empty")
	ErrRelativePath = errors.New("project path must be absolute")
	ErrPathNotExist = errors.New("project path does not exist")
)

type Project struct {
	ID           string    `yaml:"id"`
	Name         string    `yaml:"name"`
	Path         string    `yaml:"path"`
	AddedAt      time.Time `yaml:"added_at"`
	LastAccessed time.Time `yaml:"last_accessed"`
	Description  string    `yaml:"description,omitempty"`
	Editor       string    `yaml:"editor,omitempty"`
}

func NewProject(name, path string) Project {
	now := time.Now()
	return Project{
		ID:           uuid.New().String(),
		Name:         name,
		Path:         path,
		AddedAt:      now,
		LastAccessed: now,
	}
}

func (p Project) WithDescription(description string) Project {
	newP := p
	newP.Description = description
	return newP
}

func (p Project) WithEditor(editor string) Project {
	newP := p
	newP.Editor = editor
	return newP
}

func (p *Project) Touch() {
	p.LastAccessed = time.Now()
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmptyName
	}
	return nil
}

func (p *Project) ValidateAndNormalize() error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}

	if !filepath.IsAbs(p.Path) {
		return fmt.Errorf("%w: got %q", ErrRelativePath, p.Path)
	}

	if _, err := os.Stat(p.Path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrPathNotExist, p.Path)
		}
		return fmt.Errorf("cannot access path %q: %w", p.Path, err)
	}

	return nil
}
