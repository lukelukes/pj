package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTermErrors(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		err  error
		text string
	}{
		{"bogus=1", ErrUnknownField, "desc, editor, name, path"},
		{"name>x", ErrUnknownOp, "=, !=, ~, !~"},
		{"name=a[b", ErrBadGlob, "a[b"},
		{"name!=[", ErrBadGlob, "["},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			_, err := ParseTerm(tc.raw)
			require.ErrorIs(t, err, tc.err)
			require.ErrorContains(t, err, tc.text)
		})
	}
}

func TestTermMatching(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		project Project
		want    bool
	}{
		{"editor=", Project{}, true},
		{"editor=", Project{Editor: "nvim"}, false},
		{`name=a\b`, Project{Name: `a\b`}, true},
		{"name=a=b~!", Project{Name: "a=b~!"}, true},
		{"name~[", Project{Name: "a[b"}, true},
		{"path=/a/*", Project{Path: "/a/b/c"}, false},
		{"path=/a/*", Project{Path: "/a/b"}, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			term, err := ParseTerm(tc.raw)
			require.NoError(t, err)
			f, err := term.Compile()
			require.NoError(t, err)
			require.Equal(t, tc.want, f(tc.project))
		})
	}
}

func TestCompileRejectsInvalidTerms(t *testing.T) {
	for _, tc := range []struct {
		term Term
		err  error
		text string
	}{
		{Term{}, ErrUnknownField, "desc, editor, name, path"},
		{Term{Field: "bogus", Op: OpEqual}, ErrUnknownField, "bogus"},
		{Term{Field: "name"}, ErrUnknownOp, "=, !=, ~, !~"},
		{Term{Field: "name", Op: "x"}, ErrUnknownOp, `"x"`},
		{Term{Field: "name", Op: OpEqual, Value: "["}, ErrBadGlob, "["},
		{Term{Field: "name", Op: OpNotEqual, Value: "["}, ErrBadGlob, "["},
	} {
		t.Run(tc.term.String(), func(t *testing.T) {
			f, err := tc.term.Compile()
			require.Nil(t, f)
			require.ErrorIs(t, err, tc.err)
			require.ErrorContains(t, err, tc.text)
		})
	}
}
