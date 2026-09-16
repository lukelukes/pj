package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeTag(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		err         error
	}{
		{"  Lang:Go ", "lang:go", nil},
		{"c++", "c++", nil},
		{"a_b.c/d-e:1", "a_b.c/d-e:1", nil},
		{"0", "0", nil},
		{"", "", ErrEmptyTag},
		{" ", "", ErrEmptyTag},
		{"lang:", "", ErrInvalidTag},
		{":go", "", ErrInvalidTag},
		{"-a", "", ErrInvalidTag},
		{"é", "", ErrInvalidTag},
		{"a b", "", ErrInvalidTag},
		{"a*", "", ErrInvalidTag},
		{"a?", "", ErrInvalidTag},
		{"a[b]", "", ErrInvalidTag},
		{"!a", "", ErrInvalidTag},
		{"a=b", "", ErrInvalidTag},
		{"a,b", "", ErrInvalidTag},
		{"a~b", "", ErrInvalidTag},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := NormalizeTag(tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeAndParseTags(t *testing.T) {
	raw := []string{" Work ", "lang:GO", "work"}
	got, err := NormalizeTags(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"lang:go", "work"}, got)
	require.Equal(t, []string{" Work ", "lang:GO", "work"}, raw)
	got, err = ParseTags(" Work, lang:GO\t work\ncli ")
	require.NoError(t, err)
	require.Equal(t, []string{"cli", "lang:go", "work"}, got)
	got, err = ParseTags(" , ")
	require.NoError(t, err)
	require.Nil(t, got)
	_, err = NormalizeTags([]string{"ok", ""})
	require.ErrorIs(t, err, ErrEmptyTag)
	p := NewProject("p", "/p").WithTags([]string{"cli"})
	require.ErrorIs(t, p.AddTags("bad!"), ErrInvalidTag)
	require.Equal(t, []string{"cli"}, p.Tags)
}

func TestLoadTagsLenientAndOmitEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version: 1\nprojects:\n  - id: p\n    name: p\n    path: /missing\n    tags: [CLI, ' cli ', 'Bad!', '', '  ']\n  - id: empty\n    name: empty\n    path: /empty\n"), 0o600))
	cat, err := NewYAMLCatalog(path)
	require.NoError(t, err)
	require.NoError(t, cat.Load())
	p, err := cat.Get("p")
	require.NoError(t, err)
	require.Equal(t, []string{"bad!", "cli"}, p.Tags)
	require.NoError(t, cat.Save())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "version: 1")
	require.Equal(t, 1, strings.Count(string(data), "tags:"))
}

func TestCatalogTagsDoNotAlias(t *testing.T) {
	dir := t.TempDir()
	cat, err := NewYAMLCatalog(filepath.Join(dir, "catalog.yaml"))
	require.NoError(t, err)
	p := NewProject("p", dir).WithTags([]string{"cli"})
	require.NoError(t, cat.Add(p))
	p.Tags[0] = "changed"
	for _, read := range []func() Project{
		func() Project { p, err := cat.Get(p.ID); require.NoError(t, err); return p },
		func() Project { p, err := cat.GetByPath(dir); require.NoError(t, err); return p },
		func() Project { return cat.List()[0] },
		func() Project { return cat.Search("p")[0] },
	} {
		got := read()
		require.Equal(t, []string{"cli"}, got.Tags)
		got.Tags[0] = "mutated"
	}
}

func TestTagFilterSetSemantics(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		tags []string
		want bool
	}{
		{"tag=lang:*", []string{"cli", "lang:go"}, true},
		{"tag!=archived", []string{"cli", "archived"}, false},
		{"tag!~GO", []string{"cli", "lang:go"}, false},
		{"tag~GO", []string{"cli", "lang:go"}, true},
		{"tag=", nil, false},
		{"tag~", nil, false},
		{"tag!=", nil, true},
		{"tag!~", nil, true},
	} {
		term, err := ParseTerm(tc.raw)
		require.NoError(t, err)
		require.Equal(t, tc.want, term.Filter()(Project{Tags: tc.tags}), tc.raw)
	}
}
