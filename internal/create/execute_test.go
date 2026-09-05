package create

import (
	"errors"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flakyCatalog struct {
	catalog.Catalog
	addErr  error
	saveErr error
	saves   int
}

func (c *flakyCatalog) Add(p catalog.Project) error {
	if c.addErr != nil {
		return c.addErr
	}
	return c.Catalog.Add(p)
}

func (c *flakyCatalog) Save() error {
	c.saves++
	if c.saveErr != nil {
		return c.saveErr
	}
	return c.Catalog.Save()
}

type recordingRunner struct {
	calls [][]string
	err   error
}

func (r *recordingRunner) run(name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	if r.err != nil {
		return r.err
	}
	if name == "git" && len(args) == 2 && args[0] == "init" {
		return os.MkdirAll(filepath.Join(args[1], ".git"), 0o755)
	}
	return nil
}

func newService(t *testing.T) (Service, *flakyCatalog, *recordingRunner) {
	t.Helper()

	cat, err := catalog.NewYAMLCatalog(filepath.Join(t.TempDir(), "catalog.yaml"))
	require.NoError(t, err)

	flaky := &flakyCatalog{Catalog: cat}
	runner := &recordingRunner{}

	return Service{
		Cat:      flaky,
		Run:      runner.run,
		GitFound: func() bool { return true },
	}, flaky, runner
}

func planIn(t *testing.T, env Env, location, name string, git bool) Plan {
	t.Helper()
	return BuildPlan(Request{Name: name, Location: location, Git: git}, env)
}

func TestExecuteCreatesProject(t *testing.T) {
	svc, flaky, runner := newService(t)
	location := t.TempDir()
	env := CatalogEnv{Cat: flaky}

	out, err := svc.Execute(planIn(t, env, location, "api", true), false)

	require.NoError(t, err)
	assert.Equal(t, []string{"dir", "git", ".gitignore", "catalog"}, out.Steps)
	assert.True(t, out.CreatedDir)
	assert.False(t, out.Adopted)
	assert.Equal(t, [][]string{{"git", "init", out.Path}}, runner.calls)

	assert.DirExists(t, out.Path)
	assert.FileExists(t, filepath.Join(out.Path, ".gitignore"))
	assert.Equal(t, 1, flaky.Count())
}

func TestExecuteWithoutGit(t *testing.T) {
	svc, _, runner := newService(t)
	location := t.TempDir()

	out, err := svc.Execute(planIn(t, CatalogEnv{Cat: svc.Cat}, location, "api", false), false)

	require.NoError(t, err)
	assert.Equal(t, []string{"dir", "catalog"}, out.Steps)
	assert.Empty(t, runner.calls)
	assert.NoDirExists(t, filepath.Join(out.Path, ".git"))
}

func TestExecuteSkipsGitWhenBinaryMissing(t *testing.T) {
	svc, _, runner := newService(t)
	svc.GitFound = func() bool { return false }
	location := t.TempDir()

	out, err := svc.Execute(planIn(t, CatalogEnv{Cat: svc.Cat}, location, "api", true), false)

	require.NoError(t, err)
	assert.Equal(t, []string{"dir", "catalog"}, out.Steps)
	assert.Empty(t, runner.calls, "a missing git binary must not abort the create")
}

func TestExecuteRequiresExplicitAdoption(t *testing.T) {
	svc, flaky, _ := newService(t)
	location := t.TempDir()
	existing := filepath.Join(location, "legacy")
	require.NoError(t, os.Mkdir(existing, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(existing, "keep.txt"), []byte("data"), 0o644))

	plan := planIn(t, CatalogEnv{Cat: flaky}, location, "legacy", true)
	require.True(t, plan.NeedsAdopt())

	_, err := svc.Execute(plan, false)

	require.ErrorIs(t, err, ErrAdoptNotConfirmed)
	assert.FileExists(t, filepath.Join(existing, "keep.txt"))
	assert.Zero(t, flaky.Count(), "a refused adoption must not touch the catalog")
}

func TestExecuteAdoptsWhenConfirmed(t *testing.T) {
	svc, flaky, _ := newService(t)
	location := t.TempDir()
	existing := filepath.Join(location, "legacy")
	require.NoError(t, os.Mkdir(existing, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(existing, ".gitignore"), []byte("mine\n"), 0o644))

	out, err := svc.Execute(planIn(t, CatalogEnv{Cat: flaky}, location, "legacy", true), true)

	require.NoError(t, err)
	assert.True(t, out.Adopted)
	assert.False(t, out.CreatedDir)
	assert.Equal(t, 1, flaky.Count())

	content, err := os.ReadFile(filepath.Join(existing, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, "mine\n", string(content), "adoption must never overwrite existing files")
}

func TestExecuteRollsBackDirectoriesItCreated(t *testing.T) {
	failures := map[string]func(*flakyCatalog){
		"catalog add fails":  func(c *flakyCatalog) { c.addErr = errors.New("add boom") },
		"catalog save fails": func(c *flakyCatalog) { c.saveErr = errors.New("save boom") },
	}

	for name, inject := range failures {
		t.Run(name, func(t *testing.T) {
			svc, flaky, _ := newService(t)
			inject(flaky)
			location := t.TempDir()

			_, err := svc.Execute(planIn(t, CatalogEnv{Cat: flaky}, location, "api", true), false)

			require.Error(t, err)
			assert.NoDirExists(t, filepath.Join(location, "api"),
				"a directory we created must not survive a failed create")
		})
	}
}

func TestExecuteRollsBackWhenGitFails(t *testing.T) {
	svc, _, runner := newService(t)
	runner.err = errors.New("git exploded")
	location := t.TempDir()

	_, err := svc.Execute(planIn(t, CatalogEnv{Cat: svc.Cat}, location, "api", true), false)

	require.Error(t, err)
	assert.NoDirExists(t, filepath.Join(location, "api"))
}

func TestExecuteNeverDeletesAnAdoptedDirectory(t *testing.T) {
	svc, flaky, _ := newService(t)
	flaky.saveErr = errors.New("disk full")
	location := t.TempDir()
	existing := filepath.Join(location, "legacy")
	require.NoError(t, os.Mkdir(existing, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(existing, "precious.txt"), []byte("work"), 0o644))

	_, err := svc.Execute(planIn(t, CatalogEnv{Cat: flaky}, location, "legacy", false), true)

	require.Error(t, err)
	assert.FileExists(t, filepath.Join(existing, "precious.txt"),
		"rollback must only remove what this command created")
}

func TestExecuteRefusesBlockedPlans(t *testing.T) {
	svc, flaky, _ := newService(t)

	t.Run("validation blocker", func(t *testing.T) {
		_, err := svc.Execute(BuildPlan(Request{Name: ""}, CatalogEnv{Cat: flaky}), false)

		require.ErrorIs(t, err, ErrPlanBlocked)
		assert.ErrorIs(t, err, ErrEmptyName)
	})

	t.Run("already tracked path", func(t *testing.T) {
		location := t.TempDir()
		existing := filepath.Join(location, "dup")
		require.NoError(t, os.Mkdir(existing, 0o755))
		require.NoError(t, flaky.Add(catalog.NewProject("dup", existing)))

		_, err := svc.Execute(planIn(t, CatalogEnv{Cat: flaky}, location, "dup", false), true)

		require.ErrorIs(t, err, ErrPlanBlocked)
	})
}

func TestExecuteReportsPermissionDenied(t *testing.T) {
	svc, flaky, _ := newService(t)
	location := t.TempDir()
	require.NoError(t, os.Chmod(location, 0o555))
	t.Cleanup(func() { os.Chmod(location, 0o755) })

	_, err := svc.Execute(planIn(t, CatalogEnv{Cat: flaky}, location, "api", false), false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
