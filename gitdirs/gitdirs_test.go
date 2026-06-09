package gitdirs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing"
)

type fakeDirEntry struct {
	name  string
	isDir bool
}

const depth = 2

func (e fakeDirEntry) Name() string               { return e.name }
func (e fakeDirEntry) IsDir() bool                { return e.isDir }
func (e fakeDirEntry) Type() fs.FileMode          { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type fakeFS struct {
	trees    map[string][]fs.DirEntry
	gitRepos map[string]bool
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		trees:    make(map[string][]fs.DirEntry),
		gitRepos: make(map[string]bool),
	}
}

func (f *fakeFS) dir(path string, entries ...fakeDirEntry) {
	f.trees[path] = make([]fs.DirEntry, len(entries))
	for i, e := range entries {
		f.trees[path][i] = e
	}
}

func (f *fakeFS) git(repoPath string) {
	f.gitRepos[repoPath+"/.git"] = true
}

func (f *fakeFS) readDir(name string) ([]fs.DirEntry, error) {
	entries, ok := f.trees[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return entries, nil
}

func (f *fakeFS) statDir(name string) (os.FileInfo, error) {
	if f.gitRepos[name] {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func TestDirs_directRepos(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "alpha", isDir: true},
		fakeDirEntry{name: "beta", isDir: true},
		fakeDirEntry{name: "notes.md", isDir: false},
	)
	fsys.git("/repos/alpha")
	fsys.git("/repos/beta")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	want := map[string]string{
		"alpha": "/repos/alpha",
		"beta":  "/repos/beta",
	}
	assertDirsEqual(t, dirs, want)
}

func TestDirs_nestedRepos(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "my-project", isDir: true},
		fakeDirEntry{name: "other", isDir: true},
	)
	fsys.dir("/repos/my-project",
		fakeDirEntry{name: "frontend", isDir: true},
		fakeDirEntry{name: "backend", isDir: true},
	)
	fsys.dir("/repos/other",
		fakeDirEntry{name: "lib", isDir: true},
	)
	fsys.git("/repos/my-project/frontend")
	fsys.git("/repos/my-project/backend")
	fsys.git("/repos/other/lib")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	want := map[string]string{
		"my-project/frontend": "/repos/my-project/frontend",
		"my-project/backend":  "/repos/my-project/backend",
		"other/lib":           "/repos/other/lib",
	}
	assertDirsEqual(t, dirs, want)
}

func TestDirs_mixedDirectAndNested(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "direct-repo", isDir: true},
		fakeDirEntry{name: "my-project", isDir: true},
		fakeDirEntry{name: "not-a-repo", isDir: true},
		fakeDirEntry{name: "readme.md", isDir: false},
	)
	fsys.dir("/repos/my-project",
		fakeDirEntry{name: "project", isDir: true},
		fakeDirEntry{name: "notes.txt", isDir: false},
	)
	fsys.dir("/repos/not-a-repo",
		fakeDirEntry{name: "empty-dir", isDir: true},
	)
	fsys.git("/repos/direct-repo")
	fsys.git("/repos/my-project/project")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	want := map[string]string{
		"direct-repo":        "/repos/direct-repo",
		"my-project/project": "/repos/my-project/project",
	}
	assertDirsEqual(t, dirs, want)
}

func TestDirs_doesNotDescendBeyondOneLevel(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "wrapper", isDir: true},
	)
	fsys.dir("/repos/wrapper",
		fakeDirEntry{name: "middle", isDir: true},
	)
	fsys.dir("/repos/wrapper/middle",
		fakeDirEntry{name: "deep-repo", isDir: true},
	)
	fsys.git("/repos/wrapper/middle/deep-repo")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	if len(dirs) != 0 {
		t.Fatalf("expected no repos, got %#v", dirs)
	}
}

func TestDirs_directRepoSkipsNestedScan(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "monorepo", isDir: true},
	)
	fsys.dir("/repos/monorepo",
		fakeDirEntry{name: "subpackage", isDir: true},
	)
	fsys.git("/repos/monorepo")
	fsys.git("/repos/monorepo/subpackage")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	want := map[string]string{
		"monorepo": "/repos/monorepo",
	}
	assertDirsEqual(t, dirs, want)
}

func TestDirs_emptyRoot(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos")

	dirs, err := Dirs("/repos", fsys.readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}
	if len(dirs) != 0 {
		t.Fatalf("expected empty map, got %#v", dirs)
	}
}

func TestDirs_rootReadError(t *testing.T) {
	readErr := errors.New("permission denied")
	readDir := func(name string) ([]fs.DirEntry, error) {
		return nil, readErr
	}

	_, err := Dirs("/repos", readDir, func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}, depth)
	if !errors.Is(err, readErr) {
		t.Fatalf("expected %v, got %v", readErr, err)
	}
}

func TestDirs_nestedReadErrorIsIgnored(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/repos",
		fakeDirEntry{name: "unreadable", isDir: true},
		fakeDirEntry{name: "good", isDir: true},
	)
	fsys.git("/repos/good")

	readDir := func(name string) ([]fs.DirEntry, error) {
		if name == "/repos/unreadable" {
			return nil, errors.New("permission denied")
		}
		return fsys.readDir(name)
	}

	dirs, err := Dirs("/repos", readDir, fsys.statDir, depth)
	if err != nil {
		t.Fatalf("Dirs: %v", err)
	}

	want := map[string]string{
		"good": "/repos/good",
	}
	assertDirsEqual(t, dirs, want)
}

func TestProjectDirs_mergesMultipleRoots(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/work",
		fakeDirEntry{name: "work-repo", isDir: true},
	)
	fsys.dir("/personal",
		fakeDirEntry{name: "personal-repo", isDir: true},
	)
	fsys.git("/work/work-repo")
	fsys.git("/personal/personal-repo")

	dirs, err := ProjectDirs([]string{"/work", "/personal"}, fsys.readDir, fsys.statDir)
	if err != nil {
		t.Fatalf("ProjectDirs: %v", err)
	}

	want := map[string]string{
		"work-repo":     "/work/work-repo",
		"personal-repo": "/personal/personal-repo",
	}
	assertDirsEqual(t, dirs, want)
}

func TestProjectDirs_firstRootWinsOnDuplicateName(t *testing.T) {
	fsys := newFakeFS()
	fsys.dir("/first",
		fakeDirEntry{name: "shared-name", isDir: true},
	)
	fsys.dir("/second",
		fakeDirEntry{name: "shared-name", isDir: true},
	)
	fsys.git("/first/shared-name")
	fsys.git("/second/shared-name")

	dirs, err := ProjectDirs([]string{"/first", "/second"}, fsys.readDir, fsys.statDir)
	if err != nil {
		t.Fatalf("ProjectDirs: %v", err)
	}

	want := map[string]string{
		"shared-name": "/first/shared-name",
	}
	assertDirsEqual(t, dirs, want)
}

func TestProjectDirs_propagatesRootError(t *testing.T) {
	readErr := errors.New("permission denied")
	readDir := func(name string) ([]fs.DirEntry, error) {
		if name == "/bad" {
			return nil, readErr
		}
		return nil, nil
	}

	_, err := ProjectDirs([]string{"/good", "/bad"}, readDir, func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, readErr) {
		t.Fatalf("expected wrapped %v, got %v", readErr, err)
	}
	if got := err.Error(); got != fmt.Sprintf("scan /bad: %v", readErr) {
		t.Fatalf("unexpected error message: %q", got)
	}
}

func assertDirsEqual(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d dirs, want %d: got=%#v want=%#v", len(got), len(want), got, want)
	}
	for name, path := range want {
		if gotPath, ok := got[name]; !ok || gotPath != path {
			t.Errorf("dirs[%q] = %q, want %q", name, gotPath, path)
		}
	}
}
