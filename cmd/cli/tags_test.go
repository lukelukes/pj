package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/create"
	"pj/internal/ui"
	"pj/proptest"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestTagFlagParsing(t *testing.T) {
	for _, args := range [][]string{
		{"add", "/project", "-t", "cli,work", "--tag", "lang:go"},
		{"create", "project", "-t", "cli,work", "--tag", "lang:go"},
		{"edit", "project", "-t", "cli,work", "--tag", "lang:go"},
		{"list", "-t", "cli,work", "--tag", "lang:go", "-o", "tags"},
		{"tag", "detect", "-t", "cli,work", "--tag", "lang:go", "--apply"},
	} {
		var cli struct {
			Add    AddCmd    `cmd:""`
			Create CreateCmd `cmd:""`
			Edit   EditCmd   `cmd:""`
			List   ListCmd   `cmd:""`
			Tag    TagCmd    `cmd:""`
		}
		parser, err := kong.New(&cli)
		require.NoError(t, err)
		_, err = parser.Parse(args)
		require.NoError(t, err)
		tags := map[string][]string{"add": cli.Add.Tags, "create": cli.Create.Tags, "edit": cli.Edit.Tags, "list": cli.List.Tags, "tag": cli.Tag.Detect.Tags}
		require.Equal(t, []string{"cli", "work", "lang:go"}, tags[args[0]])
	}
	var cli struct {
		Edit EditCmd `cmd:""`
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"edit", "p", "--clear-tags", "--untag", "cli"})
	require.Error(t, err)
}

func TestTagMetadataCommands(t *testing.T) {
	g, out := newTestGlobals(t)
	cmd := AddCmd{Path: t.TempDir(), Name: "p", Tags: []string{"Work", "lang:GO", "work"}}
	require.NoError(t, cmd.Run(g))
	require.NoError(t, g.Cat.Load())
	require.Equal(t, []string{"lang:go", "work"}, g.Cat.List()[0].Tags)
	require.NoError(t, (&EditCmd{Name: "p", Untag: []string{"WORK"}, Tags: []string{"cli", "lang:go"}}).Run(g))
	require.NoError(t, g.Cat.Load())
	require.Equal(t, []string{"cli", "lang:go"}, g.Cat.List()[0].Tags)
	out.Reset()
	require.NoError(t, (&ShowCmd{Name: "p"}).Run(g))
	require.Contains(t, out.String(), "Tags:   cli, lang:go\n")
	out.Reset()
	require.NoError(t, (&ListCmd{Output: OutputTags}).Run(g))
	require.Equal(t, "cli\nlang:go\n", out.String())
	require.NoError(t, (&EditCmd{Name: "p", ClearTags: true, Tags: []string{"fresh"}}).Run(g))
	require.Equal(t, []string{"fresh"}, g.Cat.List()[0].Tags)
	require.ErrorIs(t, (&EditCmd{Name: "p", ClearTags: true, Tags: []string{"bad!"}}).Run(g), catalog.ErrInvalidTag)
	require.ErrorIs(t, (&EditCmd{Name: "p", Untag: []string{"bad!"}}).Run(g), catalog.ErrInvalidTag)
	require.Error(t, (&EditCmd{Name: "p", ClearTags: true, Untag: []string{"fresh"}}).Run(g))
	require.NoError(t, g.Cat.Load())
	require.Equal(t, []string{"fresh"}, g.Cat.List()[0].Tags)
	require.NoError(t, (&EditCmd{Name: "p", ClearTags: true}).Run(g))
	require.NoError(t, g.Cat.Load())
	require.Nil(t, g.Cat.List()[0].Tags)
	out.Reset()
	require.NoError(t, (&ListCmd{Output: OutputJSON}).Run(g))
	var rows []projectJSON
	require.NoError(t, json.Unmarshal(out.Bytes(), &rows))
	require.NotNil(t, rows[0].Tags)
	require.Empty(t, rows[0].Tags)
	require.ErrorIs(t, (&AddCmd{Path: t.TempDir(), Name: "bad", Tags: []string{"!"}}).Run(g), catalog.ErrInvalidTag)
	require.Equal(t, 1, g.Cat.Count())
}

func TestCreateTags(t *testing.T) {
	g, _ := newTestGlobals(t)
	parent := t.TempDir()
	cmd := CreateCmd{Name: "tagged", At: parent, NoGit: true, NoInput: true, Tags: []string{"Work", "lang:GO", "work"}}
	require.NoError(t, cmd.Run(g))
	require.NoError(t, g.Cat.Load())
	require.Equal(t, []string{"lang:go", "work"}, g.Cat.List()[0].Tags)
	cmd.Name, cmd.Tags = "invalid", []string{"bad!"}
	require.ErrorIs(t, cmd.Run(g), catalog.ErrInvalidTag)
	_, err := os.Stat(filepath.Join(parent, "invalid"))
	require.ErrorIs(t, err, os.ErrNotExist)
	req := create.Request{Name: "p", Location: parent, Tags: []string{"cli", "lang:go"}, Git: true}
	require.Equal(t, req, toRequest(toDraft(req)))
	req = toRequest(ui.Draft{Tags: "CLI, lang:go work"})
	require.Equal(t, []string{"CLI", "lang:go", "work"}, req.Tags)
	preview := toPreview(create.Plan{Request: create.Request{Tags: []string{"cli", "lang:go"}}, Path: "/p"}, "")
	require.Contains(t, preview.Facts, "#cli #lang:go")
}

func TestProperty_TagSugar(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		term := proptest.TermGen().Draw(t, "term")
		term.Field, term.Op = "tag", "="
		if len(ps) > 0 && len(ps[0].Tags) > 0 && rapid.Bool().Draw(t, "self") {
			term.Value = ps[0].Tags[0]
		}
		cat := selectionCatalog{projects: ps}
		got, err := (Selector{Tags: []string{term.Value}}).Select(cat)
		require.NoError(t, err)
		want, err := (Selector{Filters: []string{term.String()}}).Select(cat)
		require.NoError(t, err)
		require.Equal(t, want, got, proptest.InvTagSugarEqualsTerm)
	})
}

func TestProperty_TagProjection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ps := proptest.ProjectsGen().Draw(t, "projects")
		var output, shuffled bytes.Buffer
		require.NoError(t, printProjects(&output, ps, OutputTags))
		require.NoError(t, printProjects(&shuffled, proptest.Permute(t, ps), OutputTags))
		require.Equal(t, output.String(), shuffled.String())
		var want strings.Builder
		for _, tag := range catalog.TagSet(ps) {
			want.WriteString(tag + "\n")
		}
		require.Equal(t, want.String(), output.String())
	})
}
