package proptest

import (
	"fmt"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func rawTags(t *rapid.T) []string {
	tags := tagsGen.Draw(t, "tags")
	var raw []string
	for _, tag := range tags {
		raw = append(raw, " "+strings.ToUpper(tag)+" ", tag)
	}
	return Permute(t, raw)
}

func TestProperty_TagNormalize(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rawTags(t)
		normalized, err := catalog.NormalizeTags(raw)
		require.NoError(t, err)
		again, err := catalog.NormalizeTags(normalized)
		require.NoError(t, err)
		require.Equal(t, normalized, again, InvTagNormalizeIdempotent)
		require.True(t, slices.IsSorted(normalized), InvTagsSortedUnique)
		require.Equal(t, normalized, slices.Compact(slices.Clone(normalized)), InvTagsSortedUnique)
		for _, tag := range raw {
			require.Contains(t, normalized, strings.ToLower(strings.TrimSpace(tag)))
		}
		for _, tag := range normalized {
			require.True(t, slices.ContainsFunc(raw, func(s string) bool { return strings.ToLower(strings.TrimSpace(s)) == tag }))
		}
	})
}

func TestProperty_TagsAddRemove(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := projectGen.Draw(t, "project")
		before := slices.Clone(p.Tags)
		added, err := catalog.NormalizeTags(rawTags(t))
		require.NoError(t, err)
		require.NoError(t, p.AddTags(added...))
		union := slices.Clone(p.Tags)
		require.NoError(t, p.AddTags(added...))
		require.Equal(t, union, p.Tags, InvAddRemoveTagsInverse)
		for _, tag := range append(slices.Clone(before), added...) {
			require.True(t, p.HasTag(tag))
		}
		p.RemoveTags(added...)
		for _, tag := range added {
			require.False(t, p.HasTag(tag), InvAddRemoveTagsInverse)
		}
		for _, tag := range before {
			require.Equal(t, !slices.Contains(added, tag), p.HasTag(tag))
		}
	})
}

func TestProperty_TagSetUnion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := projectsGen.Draw(t, "projects")
		got := catalog.TagSet(ps)
		require.True(t, slices.IsSorted(got), InvTagSetIsUnion)
		require.Equal(t, got, slices.Compact(slices.Clone(got)), InvTagSetIsUnion)
		for _, p := range ps {
			for _, tag := range p.Tags {
				require.Contains(t, got, tag, InvTagSetIsUnion)
			}
		}
		for _, tag := range got {
			require.True(t, slices.ContainsFunc(ps, func(p catalog.Project) bool { return p.HasTag(tag) }), InvTagSetIsUnion)
		}
	})
}

func TestProperty_TagsCatalogNormalization(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		p := h.GenProject()
		for _, save := range []func(catalog.Project) error{h.Catalog.Add, h.Catalog.Update} {
			p.Tags = rawTags(h.T)
			want, err := catalog.NormalizeTags(p.Tags)
			require.NoError(h.T, err)
			require.NoError(h.T, save(p))
			got, err := h.Catalog.Get(p.ID)
			require.NoError(h.T, err)
			require.Equal(h.T, want, got.Tags, InvTagsSortedUnique)
		}
	})
}

func TestProperty_LenientTagsLoad(t *testing.T) {
	RunBasic(t, func(h *Harness) {
		raw := rawTags(h.T)
		want, err := catalog.NormalizeTags(raw)
		require.NoError(h.T, err)
		var yaml strings.Builder
		yaml.WriteString("version: 1\nprojects:\n  - id: p\n    name: p\n    path: /missing\n    tags:\n")
		for _, tag := range raw {
			fmt.Fprintf(&yaml, "      - %q\n", tag)
		}
		path := filepath.Join(h.Dir, "catalog.yaml")
		require.NoError(h.T, os.WriteFile(path, []byte(yaml.String()), 0o600))
		cat, err := catalog.NewYAMLCatalog(path)
		require.NoError(h.T, err)
		require.NoError(h.T, cat.Load())
		got, err := cat.Get("p")
		require.NoError(h.T, err)
		require.Equal(h.T, want, got.Tags, InvLenientLoadEqualsStrict)
	})
}
