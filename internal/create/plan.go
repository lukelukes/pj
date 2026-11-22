package create

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"pj/internal/catalog"
	"strings"
	"time"
)

var (
	ErrEmptyName        = errors.New("name is required")
	ErrNameSeparator    = errors.New("name cannot contain a path separator")
	ErrNameDots         = errors.New(`name cannot be "." or ".."`)
	ErrEmptyLocation    = errors.New("location is required")
	ErrRelativeLocation = errors.New("location must be an absolute path")
	ErrLocationMissing  = errors.New("location does not exist")
	ErrLocationNotDir   = errors.New("location is not a directory")
	ErrTargetNotDir     = errors.New("target exists and is not a directory")
)

type Request struct {
	Tags        []string
	Name        string
	Location    string
	Description string
	Editor      string
	Git         bool
}

type Conflict struct {
	Name    string
	Path    string
	AddedAt time.Time
}

type Env interface {
	Stat(path string) (fs.FileInfo, error)
	CountEntries(path string) (int, error)
	GitRoot(path string) string
	ByName(name string) (Conflict, bool)
	ByPath(path string) (Conflict, bool)
}

type Plan struct {
	Request
	Path       string
	DirExists  bool
	DirEntries int
	GitRoot    string
	NameTaken  *Conflict
	PathTaken  *Conflict
	Blocker    error
}

func BuildPlan(req Request, env Env) Plan {
	p := Plan{Request: normalize(req)}

	if err := validate(p.Request); err != nil {
		p.Blocker = err
		return p
	}

	tags, err := catalog.NormalizeTags(p.Tags)
	if err != nil {
		p.Blocker = err
		return p
	}
	p.Tags = tags

	p.Path = filepath.Join(p.Location, p.Name)
	p.attachConflicts(env)
	p.probe(env)
	return p
}

func (p *Plan) attachConflicts(env Env) {
	if c, ok := env.ByPath(p.Path); ok {
		p.PathTaken = &c
	}
	if c, ok := env.ByName(p.Name); ok && (p.PathTaken == nil || c.Path != p.Path) {
		p.NameTaken = &c
	}
}

func (p *Plan) probe(env Env) {
	if err := p.probeLocation(env); err != nil {
		p.Blocker = err
		return
	}
	if err := p.probeTarget(env); err != nil {
		p.Blocker = err
		return
	}
	p.GitRoot = env.GitRoot(p.Path)
}

func (p *Plan) probeLocation(env Env) error {
	info, err := env.Stat(p.Location)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrLocationMissing
	case err != nil:
		return fmt.Errorf("cannot use %s: %w", p.Location, err)
	case !info.IsDir():
		return ErrLocationNotDir
	}
	return nil
}

func (p *Plan) probeTarget(env Env) error {
	info, err := env.Stat(p.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("cannot inspect %s: %w", p.Path, err)
	case !info.IsDir():
		return ErrTargetNotDir
	}

	p.DirExists = true
	p.DirEntries, _ = env.CountEntries(p.Path)
	return nil
}

func normalize(req Request) Request {
	req.Name = strings.TrimSpace(req.Name)
	req.Location = filepath.Clean(strings.TrimSpace(req.Location))
	req.Description = strings.TrimSpace(req.Description)
	req.Editor = strings.TrimSpace(req.Editor)
	return req
}

func validate(req Request) error {
	switch {
	case req.Name == "":
		return ErrEmptyName
	case req.Name == "." || req.Name == "..":
		return ErrNameDots
	case strings.ContainsRune(req.Name, filepath.Separator):
		return ErrNameSeparator
	case req.Location == "" || req.Location == ".":
		return ErrEmptyLocation
	case !filepath.IsAbs(req.Location):
		return ErrRelativeLocation
	}
	return nil
}

func (p Plan) Blocked() bool { return p.Blocker != nil || p.PathTaken != nil }

func (p Plan) NeedsAdopt() bool {
	return p.Blocker == nil && p.PathTaken == nil && p.DirExists
}

func (p Plan) WillInitGit() bool {
	return p.Blocker == nil && p.Git && p.GitRoot == ""
}

func (p Plan) NestedIn() string {
	if p.Git && p.GitRoot != "" {
		return p.GitRoot
	}
	return ""
}
