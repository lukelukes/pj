package detect

import (
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDetect(t *testing.T) {
	require.Equal(t, []string{"lang:js", "lang:ts"}, Detect(fstest.MapFS{"package.json": {}, "tsconfig.json": {}}))
	require.Empty(t, Detect(fstest.MapFS{"go.mod": {Mode: fs.ModeDir}, "nested/package.json": {}}))
	require.Equal(t, []string{"lang:python"}, Detect(fstest.MapFS{"setup.py": {}, "requirements.txt": {}}))
}

func TestProperty_DetectMarkerSubsets(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		files := fstest.MapFS{}
		var want []string
		for _, rule := range rules {
			selected := rapid.SliceOfNDistinct(rapid.SampledFrom(rule.Markers), 0, len(rule.Markers), func(s string) string { return s }).Draw(t, rule.Tag)
			if len(selected) > 0 {
				want = append(want, rule.Tag)
			}
			for _, marker := range selected {
				files[marker] = &fstest.MapFile{}
			}
		}
		slices.Sort(want)
		require.Equal(t, want, Detect(files))
		for _, noise := range rapid.SliceOfN(rapid.StringMatching(`noise-[a-z]{1,10}`), 0, 10).Draw(t, "noise") {
			files[noise] = &fstest.MapFile{}
		}
		require.Equal(t, want, Detect(files), "noise must not change detected tags")
	})
}

func TestDetectKnownMarkers(t *testing.T) {
	for marker, tag := range map[string]string{
		"go.mod": "lang:go", "Cargo.toml": "lang:rust", "package.json": "lang:js", "tsconfig.json": "lang:ts",
		"pyproject.toml": "lang:python", "setup.py": "lang:python", "requirements.txt": "lang:python",
		"Gemfile": "lang:ruby", "pom.xml": "lang:java", "build.gradle": "lang:java", "build.gradle.kts": "lang:java",
		"mix.exs": "lang:elixir", "build.zig": "lang:zig",
	} {
		t.Run(marker, func(t *testing.T) {
			require.Equal(t, []string{tag}, Detect(fstest.MapFS{marker: {}}))
		})
	}
}
