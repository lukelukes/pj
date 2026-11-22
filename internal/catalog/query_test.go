package catalog

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func v(text string) Value  { return Value{Text: text} }
func qv(text string) Value { return Value{Text: text, Quoted: true} }

func TestParseQuery(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want Query
	}{
		{"api", Query{Terms: []Term{{Values: []Value{v("api")}}}}},
		{"  api \t server ", Query{Terms: []Term{{Values: []Value{v("api")}}, {Values: []Value{v("server")}}}}},
		{`"web server"`, Query{Terms: []Term{{Values: []Value{qv("web server")}}}}},
		{`"a \"b\" \\ \n"`, Query{Terms: []Term{{Values: []Value{qv(`a "b" \ \n`)}}}}},
		{`a\b`, Query{Terms: []Term{{Values: []Value{v(`a\b`)}}}}},
		{"api,rust", Query{Terms: []Term{{Values: []Value{v("api,rust")}}}}},
		{"name:api", Query{Terms: []Term{{Qualifier: "name", Values: []Value{v("api")}}}}},
		{"name:a,b", Query{Terms: []Term{{Qualifier: "name", Values: []Value{v("a"), v("b")}}}}},
		{`desc:"rest api",x`, Query{Terms: []Term{{Qualifier: "desc", Values: []Value{qv("rest api"), v("x")}}}}},
		{"path:a:b", Query{Terms: []Term{{Qualifier: "path", Values: []Value{v("a:b")}}}}},
		{"name:-x", Query{Terms: []Term{{Qualifier: "name", Values: []Value{v("-x")}}}}},
		{"-editor:vim", Query{Terms: []Term{{Negated: true, Qualifier: "editor", Values: []Value{v("vim")}}}}},
		{"NOT editor:vim", Query{Terms: []Term{{Negated: true, Qualifier: "editor", Values: []Value{v("vim")}}}}},
		{"NOT\tapi", Query{Terms: []Term{{Negated: true, Values: []Value{v("api")}}}}},
		{"not api", Query{Terms: []Term{{Values: []Value{v("not")}}, {Values: []Value{v("api")}}}}},
		{"NOTE", Query{Terms: []Term{{Values: []Value{v("NOTE")}}}}},
		{"-NOTE", Query{Terms: []Term{{Negated: true, Values: []Value{v("NOTE")}}}}},
		{"foo-bar:baz", Query{Terms: []Term{{Values: []Value{v("foo-bar:baz")}}}}},
		{":x", Query{Terms: []Term{{Values: []Value{v(":x")}}}}},
		{"sort:name", Query{Sort: &Sort{Key: SortName}}},
		{"sort:name-desc", Query{Sort: &Sort{Key: SortName, Desc: true}}},
		{"sort:opened", Query{Sort: &Sort{Key: SortOpened, Desc: true}}},
		{"sort:modified-asc api", Query{Terms: []Term{{Values: []Value{v("api")}}}, Sort: &Sort{Key: SortModified}}},
		{"is:missing no:desc,editor", Query{Terms: []Term{{Qualifier: "is", Values: []Value{v("missing")}}, {Qualifier: "no", Values: []Value{v("desc"), v("editor")}}}}},
		{"ÉCOLE", Query{Terms: []Term{{Values: []Value{v("ÉCOLE")}}}}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := ParseQuery(tc.raw)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			again, err := ParseQuery(got.String())
			require.NoError(t, err)
			require.Equal(t, got, again)
		})
	}
}

func TestParseQueryErrors(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		err  error
		text string
	}{
		{"", ErrQuerySyntax, "empty query"},
		{" \t ", ErrQuerySyntax, "empty query"},
		{"foo:bar", ErrUnknownQualifier, "desc, editor, is, name, no, path, sort"},
		{"TAG:go", ErrUnknownQualifier, `"TAG"`},
		{"http://x", ErrUnknownQualifier, `"http"`},
		{"name:", ErrQuerySyntax, `empty value for qualifier "name"`},
		{"name: api", ErrQuerySyntax, "empty value"},
		{"name:a,", ErrQuerySyntax, "empty value"},
		{"name:a,,b", ErrQuerySyntax, "empty value"},
		{"name:,a", ErrQuerySyntax, "empty value"},
		{`name:""`, ErrQuerySyntax, "empty quoted value"},
		{`""`, ErrQuerySyntax, "empty quoted value"},
		{`name:"a"b`, ErrQuerySyntax, "after closing quote"},
		{`"a"b`, ErrQuerySyntax, "after closing quote"},
		{`name:a"b"`, ErrQuerySyntax, "quote inside"},
		{`a"b`, ErrQuerySyntax, "quote inside"},
		{`name:"unterminated`, ErrQuerySyntax, "unterminated"},
		{`"a\"`, ErrQuerySyntax, "unterminated"},
		{"-", ErrQuerySyntax, "'-' must be followed"},
		{"- api", ErrQuerySyntax, "'-' must be followed"},
		{"NOT", ErrQuerySyntax, "NOT must be followed"},
		{"api NOT  ", ErrQuerySyntax, "NOT must be followed"},
		{"NOT NOT api", ErrQuerySyntax, "negated once"},
		{"NOT -api", ErrQuerySyntax, "negated once"},
		{"--api", ErrQuerySyntax, "negated once"},
		{"-NOT api", ErrQuerySyntax, "negated once"},
		{"is:Missing", ErrBadValue, "valid values: missing"},
		{"no:name", ErrBadValue, "valid values: desc, editor"},
		{"sort:name sort:opened", ErrBadSort, "only one sort"},
		{"-sort:name", ErrBadSort, "cannot be negated"},
		{"NOT sort:name", ErrBadSort, "cannot be negated"},
		{"sort:name,added", ErrBadSort, "single value"},
		{"sort:updated", ErrBadSort, "valid sorts"},
		{"sort:name-", ErrBadSort, "valid sorts"},
		{"sort:name-up", ErrBadSort, "valid sorts"},
		{"sort:Name", ErrBadSort, "valid sorts"},
		{"path:~/work/**", ErrBadGlob, "** is not supported"},
		{"name:a[b", ErrBadGlob, "unterminated"},
		{"name:[]", ErrBadGlob, "empty character class"},
		{"name:[z-a]", ErrBadGlob, "reversed"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			q, err := ParseQuery(tc.raw)
			if err == nil {
				_, err = q.Compile(QueryEnv{Home: "/h", Missing: func(Project) bool { return false }})
			}
			require.ErrorIs(t, err, tc.err)
			require.ErrorContains(t, err, tc.text)
		})
	}
}

func TestQueryMatching(t *testing.T) {
	api := Project{ID: "1", Name: "api-server", Path: "/h/work/api", Description: "REST API: why?", Editor: "nvim"}
	web := Project{ID: "2", Name: "Web", Path: "/h/Work/web/ui", Description: "docs/api/service µικρο"}
	gone := Project{ID: "3", Name: "école", Path: "/srv/gone", Editor: "code"}
	all := []Project{api, web, gone}
	env := QueryEnv{Home: "/h", Missing: func(p Project) bool { return p.ID == "3" }}
	for _, tc := range []struct {
		raw  string
		want []string
	}{
		{"api", []string{"1", "2"}},
		{"API", []string{"1", "2"}},
		{"work", nil},
		{"ÉCOLE", []string{"3"}},
		{"desc:ΜΙΚΡΟ", []string{"2"}},
		{`"rest api"`, []string{"1"}},
		{"api-*", []string{"1"}},
		{"NOT api", []string{"3"}},
		{"-api", []string{"3"}},
		{"name:API", []string{"1"}},
		{"name:web", []string{"2"}},
		{"name:a,w", []string{"1", "2"}},
		{"NOT name:api,web", []string{"3"}},
		{"name:API-*", []string{"1"}},
		{"name:[A-Z]eb", []string{"2"}},
		{"name:[^a]*", []string{"2", "3"}},
		{"desc:*api*", []string{"1", "2"}},
		{`desc:"why?"`, []string{"1"}},
		{"desc:why?", nil},
		{`desc:"*"`, nil},
		{"editor:nvim", []string{"1"}},
		{"editor:NVIM", []string{"1"}},
		{"path:work", []string{"1"}},
		{"path:Work", []string{"2"}},
		{"path:~/work", []string{"1"}},
		{"path:~/*/*", []string{"1"}},
		{"path:~/*", nil},
		{"path:~/?ork/*/ui", []string{"2"}},
		{"path:~", []string{"1", "2"}},
		{"path:/h/work/*", []string{"1"}},
		{"path:/H/work/*", nil},
		{"no:desc", []string{"3"}},
		{"no:editor", []string{"2"}},
		{"no:desc,editor", []string{"2", "3"}},
		{"-no:editor", []string{"1", "3"}},
		{"is:missing", []string{"3"}},
		{"NOT is:missing", []string{"1", "2"}},
		{"api -editor:nvim", []string{"2"}},
		{"sort:name", []string{"1", "2", "3"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			q, err := ParseQuery(tc.raw)
			require.NoError(t, err)
			f, err := q.Compile(env)
			require.NoError(t, err)
			var got []string
			for _, p := range Apply(all, f) {
				got = append(got, p.ID)
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCompileNeedsEnvironment(t *testing.T) {
	q, err := ParseQuery("path:~/x")
	require.NoError(t, err)
	_, err = q.Compile(QueryEnv{})
	require.ErrorIs(t, err, ErrNoHome)
	q, err = ParseQuery("path:a~/x")
	require.NoError(t, err)
	_, err = q.Compile(QueryEnv{})
	require.NoError(t, err)
	q, err = ParseQuery("is:missing")
	require.NoError(t, err)
	_, err = q.Compile(QueryEnv{})
	require.ErrorContains(t, err, "snapshot")
	_, err = Term{Qualifier: "bogus", Values: []Value{v("x")}}.Compile(QueryEnv{})
	require.ErrorIs(t, err, ErrUnknownQualifier)
	_, err = Term{Qualifier: "name"}.Compile(QueryEnv{})
	require.ErrorIs(t, err, ErrQuerySyntax)
}

func TestMerge(t *testing.T) {
	a, err := ParseQuery("api sort:name")
	require.NoError(t, err)
	b, err := ParseQuery("-editor:vim")
	require.NoError(t, err)
	merged, err := Merge(a, b)
	require.NoError(t, err)
	require.Equal(t, "api -editor:vim sort:name-asc", merged.String())
	require.True(t, merged.Uses("editor"))
	require.False(t, merged.Uses("is"))
	_, err = Merge(a, a)
	require.ErrorIs(t, err, ErrBadSort)
}

type fakeStat map[string]error

func (f fakeStat) stat(path string) (fs.FileInfo, error) { return nil, f[path] }

func TestMissingSnapshot(t *testing.T) {
	here := Project{ID: "1", Path: "/here"}
	gone := Project{ID: "2", Path: "/gone"}
	locked := Project{ID: "3", Path: "/locked"}
	stat := fakeStat{"/gone": fs.ErrNotExist, "/locked": fs.ErrPermission}
	missing, err := MissingSnapshot([]Project{here, gone}, stat.stat)
	require.NoError(t, err)
	require.False(t, missing(here))
	require.True(t, missing(gone))
	_, err = MissingSnapshot([]Project{here, locked}, stat.stat)
	require.ErrorIs(t, err, fs.ErrPermission)
	require.ErrorContains(t, err, "/locked")

	dir := t.TempDir()
	file := dir + "/file"
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	require.NoError(t, os.Symlink(dir+"/nowhere", dir+"/dangling"))
	onDisk := []Project{{ID: "f", Path: file}, {ID: "d", Path: dir + "/dangling"}}
	missing, err = MissingSnapshot(onDisk, os.Stat)
	require.NoError(t, err)
	require.False(t, missing(onDisk[0]))
	require.True(t, missing(onDisk[1]))
}
