package create

import (
	"pj/internal/catalog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanTagsBeforeProbe(t *testing.T) {
	env := newFakeEnv("/dev")
	req := Request{Name: "p", Location: "/dev", Tags: []string{"bad!"}}
	p := BuildPlan(req, env)
	require.ErrorIs(t, p.Blocker, catalog.ErrInvalidTag)
	require.Zero(t, env.statCalls)
	req.Tags = []string{"Work", "lang:GO", "work"}
	p = BuildPlan(req, env)
	require.NoError(t, p.Blocker)
	require.Equal(t, []string{"lang:go", "work"}, p.Tags)
	require.Equal(t, []string{"Work", "lang:GO", "work"}, req.Tags)
}
