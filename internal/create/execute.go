package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"strings"
)

var (
	ErrAdoptNotConfirmed = errors.New("target directory already exists; adoption must be confirmed")
	ErrPlanBlocked       = errors.New("plan cannot be executed")
)

const gitignoreContent = `.DS_Store
Thumbs.db

.idea/
.vscode/
*.swp

/dist/
/build/
/out/

/vendor/
/node_modules/
`

type CommandRunner func(name string, args ...string) error

type Service struct {
	Cat      catalog.Catalog
	Run      CommandRunner
	GitFound func() bool
}

type Outcome struct {
	Path       string
	Steps      []string
	Adopted    bool
	CreatedDir bool
}

func (s Service) Execute(p Plan, adopt bool) (Outcome, error) {
	if err := admit(p, adopt); err != nil {
		return Outcome{}, err
	}

	out := Outcome{Path: p.Path, Adopted: p.DirExists}

	if !p.DirExists {
		if err := os.Mkdir(p.Path, 0o755); err != nil {
			return Outcome{}, mkdirError(p.Path, err)
		}
		out.CreatedDir = true
		out.Steps = append(out.Steps, "dir")
	}

	committed := false
	defer func() {
		if !committed && out.CreatedDir {
			os.RemoveAll(p.Path)
		}
	}()

	if p.WillInitGit() {
		steps, err := s.initGit(p.Path)
		if err != nil {
			return Outcome{}, err
		}
		out.Steps = append(out.Steps, steps...)
	}

	if err := s.register(p); err != nil {
		return Outcome{}, err
	}
	out.Steps = append(out.Steps, "catalog")

	committed = true
	return out, nil
}

func admit(p Plan, adopt bool) error {
	switch {
	case p.Blocker != nil:
		return fmt.Errorf("%w: %w", ErrPlanBlocked, p.Blocker)
	case p.PathTaken != nil:
		return fmt.Errorf("%w: %s is already tracked as %q", ErrPlanBlocked, p.Path, p.PathTaken.Name)
	case p.DirExists && !adopt:
		return ErrAdoptNotConfirmed
	}
	return nil
}

func (s Service) initGit(path string) ([]string, error) {
	if s.GitFound != nil && !s.GitFound() {
		return nil, nil
	}
	if err := s.Run("git", "init", path); err != nil {
		return nil, fmt.Errorf("initializing git repository: %w", err)
	}
	steps := []string{"git"}

	gitignore := filepath.Join(path, ".gitignore")
	if _, err := os.Stat(gitignore); err == nil {
		return steps, nil
	}
	if err := os.WriteFile(gitignore, []byte(gitignoreContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing .gitignore: %w", err)
	}
	return append(steps, ".gitignore"), nil
}

func (s Service) register(p Plan) error {
	project := catalog.NewProject(p.Name, p.Path).
		WithDescription(p.Description).
		WithEditor(p.Editor)

	if err := s.Cat.Add(project); err != nil {
		return fmt.Errorf("adding project to catalog: %w", err)
	}
	if err := s.Cat.Save(); err != nil {
		return fmt.Errorf("saving catalog: %w", err)
	}
	return nil
}

func mkdirError(path string, err error) error {
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("permission denied: %s", path)
	}
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("directory already exists: %s", path)
	}
	return fmt.Errorf("creating directory %s: %w", strings.TrimSuffix(path, "/"), err)
}
