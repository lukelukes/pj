package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusArchived  Status = "archived"
	StatusAbandoned Status = "abandoned"
)

var (
	ErrEmptyName     = errors.New("project name cannot be empty")
	ErrRelativePath  = errors.New("project path must be absolute")
	ErrPathNotExist  = errors.New("project path does not exist")
	ErrInvalidStatus = errors.New("invalid project status")
)

type ProjectType string

const (
	TypeUnknown ProjectType = "unknown"
	TypeGo      ProjectType = "go"
	TypeRust    ProjectType = "rust"
	TypeNode    ProjectType = "node"
	TypePython  ProjectType = "python"
	TypeElixir  ProjectType = "elixir"
	TypeRuby    ProjectType = "ruby"
	TypeJava    ProjectType = "java"
	TypeGeneric ProjectType = "generic"
)

type Project struct {
	ID           string        `yaml:"id"`
	Name         string        `yaml:"name"`
	Path         string        `yaml:"path"`
	Types        []ProjectType `yaml:"types,omitempty"`
	Tags         []string      `yaml:"tags,omitempty"`
	Status       Status        `yaml:"status"`
	Notes        string        `yaml:"notes,omitempty"`
	AddedAt      time.Time     `yaml:"added_at"`
	LastAccessed time.Time     `yaml:"last_accessed"`
	GitRemote    string        `yaml:"git_remote,omitempty"`
	Description  string        `yaml:"description,omitempty"`
}

func NewProject(name, path string) Project {
	now := time.Now()
	return Project{
		ID:           uuid.New().String(),
		Name:         name,
		Path:         path,
		Types:        []ProjectType{},
		Status:       StatusActive,
		AddedAt:      now,
		LastAccessed: now,
	}
}

func (p Project) WithTypes(types ...ProjectType) Project {
	newP := p
	newP.Types = make([]ProjectType, len(types))
	copy(newP.Types, types)
	newP.Tags = make([]string, len(p.Tags))
	copy(newP.Tags, p.Tags)
	return newP
}

func (p Project) WithTags(tags ...string) Project {
	newP := p
	newP.Tags = make([]string, len(tags))
	copy(newP.Tags, tags)
	newP.Types = make([]ProjectType, len(p.Types))
	copy(newP.Types, p.Types)
	return newP
}

func (p Project) WithStatus(status Status) Project {
	newP := p
	newP.Status = status
	newP.Types = make([]ProjectType, len(p.Types))
	copy(newP.Types, p.Types)
	newP.Tags = make([]string, len(p.Tags))
	copy(newP.Tags, p.Tags)
	return newP
}

func (p Project) WithNotes(notes string) Project {
	newP := p
	newP.Notes = notes
	newP.Types = make([]ProjectType, len(p.Types))
	copy(newP.Types, p.Types)
	newP.Tags = make([]string, len(p.Tags))
	copy(newP.Tags, p.Tags)
	return newP
}

func (p Project) WithGitRemote(remote string) Project {
	newP := p
	newP.GitRemote = remote
	newP.Types = make([]ProjectType, len(p.Types))
	copy(newP.Types, p.Types)
	newP.Tags = make([]string, len(p.Tags))
	copy(newP.Tags, p.Tags)
	return newP
}

func (p Project) WithDescription(description string) Project {
	newP := p
	newP.Description = description
	newP.Types = make([]ProjectType, len(p.Types))
	copy(newP.Types, p.Types)
	newP.Tags = make([]string, len(p.Tags))
	copy(newP.Tags, p.Tags)
	return newP
}

func (p *Project) Touch() {
	p.LastAccessed = time.Now()
}

func (p Project) HasType(t ProjectType) bool {
	return slices.Contains(p.Types, t)
}

func (p Project) HasTag(tag string) bool {
	return slices.Contains(p.Tags, tag)
}

func (p *Project) AddTag(tag string) {
	if !p.HasTag(tag) {
		p.Tags = append(p.Tags, tag)
	}
}

func (p *Project) RemoveTag(tag string) {
	for i, t := range p.Tags {
		if t == tag {
			p.Tags = append(p.Tags[:i], p.Tags[i+1:]...)
			return
		}
	}
}

func (p *Project) ValidateAndNormalize() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrEmptyName
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

	if p.Status != "" && !isValidStatus(p.Status) {
		return fmt.Errorf("%w: %q", ErrInvalidStatus, p.Status)
	}

	cleanTags := make([]string, 0, len(p.Tags))
	for _, tag := range p.Tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			cleanTags = append(cleanTags, trimmed)
		}
	}
	p.Tags = cleanTags

	return nil
}

func isValidStatus(s Status) bool {
	switch s {
	case StatusActive, StatusArchived, StatusAbandoned:
		return true
	default:
		return false
	}
}
