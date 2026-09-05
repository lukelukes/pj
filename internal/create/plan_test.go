package create

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const devDir = "/home/luke/dev"

func request(name string) Request {
	return Request{Name: name, Location: devDir, Git: true}
}

func TestBuildPlanRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want error
	}{
		{"empty name", Request{Location: devDir}, ErrEmptyName},
		{"whitespace name", Request{Name: "   ", Location: devDir}, ErrEmptyName},
		{"tab and newline name", Request{Name: "\t\n", Location: devDir}, ErrEmptyName},
		{"dot name", Request{Name: ".", Location: devDir}, ErrNameDots},
		{"dotdot name", Request{Name: "..", Location: devDir}, ErrNameDots},
		{"name with separator", Request{Name: "a/b", Location: devDir}, ErrNameSeparator},
		{"name escaping upward", Request{Name: "../evil", Location: devDir}, ErrNameSeparator},
		{"empty location", Request{Name: "x"}, ErrEmptyLocation},
		{"dot location", Request{Name: "x", Location: "."}, ErrEmptyLocation},
		{"relative location", Request{Name: "x", Location: "dev/projects"}, ErrRelativeLocation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := BuildPlan(tc.req, newFakeEnv(devDir))

			require.ErrorIs(t, p.Blocker, tc.want)
			assert.True(t, p.Blocked())
			assert.Empty(t, p.Path, "a blocked plan must not expose a target path")
		})
	}
}

func TestBuildPlanRejectsBadInputWithoutTouchingTheFilesystem(t *testing.T) {
	env := newFakeEnv(devDir)

	BuildPlan(Request{Name: "a/b", Location: devDir}, env)

	assert.Zero(t, env.statCalls, "validation must be pure so the wizard can call it on every keystroke")
}

func TestBuildPlanHappyPath(t *testing.T) {
	env := newFakeEnv(devDir)

	p := BuildPlan(request("api-gateway"), env)

	require.NoError(t, p.Blocker)
	assert.Equal(t, "/home/luke/dev/api-gateway", p.Path)
	assert.False(t, p.DirExists)
	assert.False(t, p.NeedsAdopt())
	assert.True(t, p.WillInitGit())
	assert.Empty(t, p.NestedIn())
}

func TestBuildPlanNormalizesInput(t *testing.T) {
	env := newFakeEnv(devDir)

	p := BuildPlan(Request{
		Name:        "  api-gateway  ",
		Location:    devDir + "/",
		Description: "  edge router  ",
		Editor:      "  nvim ",
	}, env)

	require.NoError(t, p.Blocker)
	assert.Equal(t, "api-gateway", p.Name)
	assert.Equal(t, devDir, p.Location)
	assert.Equal(t, "edge router", p.Description)
	assert.Equal(t, "nvim", p.Editor)
}

func TestBuildPlanLocationProblems(t *testing.T) {
	t.Run("missing location", func(t *testing.T) {
		p := BuildPlan(request("x"), newFakeEnv())

		require.ErrorIs(t, p.Blocker, ErrLocationMissing)
	})

	t.Run("location is a file", func(t *testing.T) {
		env := newFakeEnv().withFile(devDir)

		p := BuildPlan(request("x"), env)

		require.ErrorIs(t, p.Blocker, ErrLocationNotDir)
	})

	t.Run("unreadable location is not reported as missing", func(t *testing.T) {
		env := newFakeEnv().failStat(devDir, os.ErrPermission)

		p := BuildPlan(request("x"), env)

		require.Error(t, p.Blocker)
		assert.ErrorIs(t, p.Blocker, os.ErrPermission)
		assert.NotErrorIs(t, p.Blocker, ErrLocationMissing,
			"a permission error must not masquerade as a missing directory")
	})
}

func TestBuildPlanTargetProblems(t *testing.T) {
	t.Run("target is an existing file", func(t *testing.T) {
		env := newFakeEnv(devDir).withFile(devDir + "/x")

		p := BuildPlan(request("x"), env)

		require.ErrorIs(t, p.Blocker, ErrTargetNotDir)
	})

	t.Run("unreadable target blocks instead of silently creating", func(t *testing.T) {
		env := newFakeEnv(devDir).failStat(devDir+"/x", os.ErrPermission)

		p := BuildPlan(request("x"), env)

		require.Error(t, p.Blocker)
		assert.ErrorIs(t, p.Blocker, os.ErrPermission)
	})
}

func TestBuildPlanAdoption(t *testing.T) {
	t.Run("existing directory needs adoption", func(t *testing.T) {
		env := newFakeEnv(devDir).withEntries(devDir+"/legacy", 12)

		p := BuildPlan(request("legacy"), env)

		require.NoError(t, p.Blocker)
		assert.True(t, p.DirExists)
		assert.Equal(t, 12, p.DirEntries)
		assert.True(t, p.NeedsAdopt())
		assert.False(t, p.Blocked(), "an adoptable directory is a notice, not a blocker")
	})

	t.Run("unreadable directory still reports adoption with zero entries", func(t *testing.T) {
		env := newFakeEnv(devDir).withEntries(devDir+"/legacy", 12)
		env.countErr = os.ErrPermission

		p := BuildPlan(request("legacy"), env)

		require.NoError(t, p.Blocker)
		assert.True(t, p.NeedsAdopt())
		assert.Zero(t, p.DirEntries)
	})

	t.Run("already tracked path blocks rather than adopts", func(t *testing.T) {
		env := newFakeEnv(devDir).
			withEntries(devDir+"/legacy", 3).
			withTrackedPath(devDir+"/legacy", Conflict{Name: "legacy", Path: devDir + "/legacy"})

		p := BuildPlan(request("legacy"), env)

		require.NotNil(t, p.PathTaken)
		assert.True(t, p.Blocked())
		assert.False(t, p.NeedsAdopt(), "a path already in the catalog cannot be adopted twice")
	})
}

func TestBuildPlanNameConflict(t *testing.T) {
	t.Run("duplicate name elsewhere is a notice", func(t *testing.T) {
		env := newFakeEnv(devDir).
			withTrackedName("api", Conflict{Name: "api", Path: "/home/luke/work/api"})

		p := BuildPlan(request("api"), env)

		require.NotNil(t, p.NameTaken)
		assert.Equal(t, "/home/luke/work/api", p.NameTaken.Path)
		assert.False(t, p.Blocked(), "duplicate names are allowed; only duplicate paths are not")
	})

	t.Run("same project is not reported twice", func(t *testing.T) {
		target := devDir + "/api"
		conflict := Conflict{Name: "api", Path: target}
		env := newFakeEnv(devDir).
			withEntries(target, 1).
			withTrackedName("api", conflict).
			withTrackedPath(target, conflict)

		p := BuildPlan(request("api"), env)

		assert.NotNil(t, p.PathTaken)
		assert.Nil(t, p.NameTaken, "the path conflict already explains this; do not double-report")
	})
}

func TestBuildPlanGitNesting(t *testing.T) {
	t.Run("initializes git when not nested", func(t *testing.T) {
		p := BuildPlan(request("x"), newFakeEnv(devDir))

		assert.True(t, p.WillInitGit())
		assert.Empty(t, p.NestedIn())
	})

	t.Run("skips git init inside an existing worktree", func(t *testing.T) {
		env := newFakeEnv(devDir).withGitRoot(devDir+"/x", "/home/luke/dev/monorepo")

		p := BuildPlan(request("x"), env)

		assert.False(t, p.WillInitGit(), "never nest a repository inside another by default")
		assert.Equal(t, "/home/luke/dev/monorepo", p.NestedIn())
	})

	t.Run("git disabled reports no nesting", func(t *testing.T) {
		req := request("x")
		req.Git = false
		env := newFakeEnv(devDir).withGitRoot(devDir+"/x", "/home/luke/dev/monorepo")

		p := BuildPlan(req, env)

		assert.False(t, p.WillInitGit())
		assert.Empty(t, p.NestedIn())
	})
}

func TestBuildPlanNameBoundaries(t *testing.T) {
	env := newFakeEnv(devDir)

	accepted := []string{
		"a", "x.y", "-dash", "_under", "проект", "日本語", "with space",
		strings.Repeat("n", 255),
	}

	for _, name := range accepted {
		t.Run(name, func(t *testing.T) {
			p := BuildPlan(request(name), env)

			require.NoError(t, p.Blocker)
			assert.Equal(t, devDir+"/"+name, p.Path)
		})
	}
}
