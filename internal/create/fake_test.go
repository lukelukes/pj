package create

import (
	"io/fs"
	"os"
	"time"
)

type fakeInfo struct {
	name  string
	isDir bool
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return 0 }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.isDir }
func (f fakeInfo) Sys() any           { return nil }

type fakeEnv struct {
	dirs      map[string]bool
	files     map[string]bool
	entries   map[string]int
	gitRoots  map[string]string
	names     map[string]Conflict
	paths     map[string]Conflict
	statErrs  map[string]error
	countErr  error
	statCalls int
}

func newFakeEnv(dirs ...string) *fakeEnv {
	e := &fakeEnv{
		dirs:     map[string]bool{},
		files:    map[string]bool{},
		entries:  map[string]int{},
		gitRoots: map[string]string{},
		names:    map[string]Conflict{},
		paths:    map[string]Conflict{},
		statErrs: map[string]error{},
	}
	for _, d := range dirs {
		e.dirs[d] = true
	}
	return e
}

func (e *fakeEnv) withFile(path string) *fakeEnv {
	e.files[path] = true
	return e
}

func (e *fakeEnv) withEntries(path string, n int) *fakeEnv {
	e.dirs[path] = true
	e.entries[path] = n
	return e
}

func (e *fakeEnv) withGitRoot(path, root string) *fakeEnv {
	e.gitRoots[path] = root
	return e
}

func (e *fakeEnv) withTrackedName(name string, c Conflict) *fakeEnv {
	e.names[name] = c
	return e
}

func (e *fakeEnv) withTrackedPath(path string, c Conflict) *fakeEnv {
	e.paths[path] = c
	return e
}

func (e *fakeEnv) failStat(path string, err error) *fakeEnv {
	e.statErrs[path] = err
	return e
}

func (e *fakeEnv) Stat(path string) (fs.FileInfo, error) {
	e.statCalls++
	if err, ok := e.statErrs[path]; ok {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: err}
	}
	if e.dirs[path] {
		return fakeInfo{name: path, isDir: true}, nil
	}
	if e.files[path] {
		return fakeInfo{name: path}, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
}

func (e *fakeEnv) CountEntries(path string) (int, error) {
	if e.countErr != nil {
		return 0, e.countErr
	}
	return e.entries[path], nil
}

func (e *fakeEnv) GitRoot(path string) string { return e.gitRoots[path] }

func (e *fakeEnv) ByName(name string) (Conflict, bool) {
	c, ok := e.names[name]
	return c, ok
}

func (e *fakeEnv) ByPath(path string) (Conflict, bool) {
	c, ok := e.paths[path]
	return c, ok
}
