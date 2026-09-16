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
			require.Equal(t, tc.want, term.Filter()(tc.project))
		})
	}
}
