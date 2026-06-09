package gitdirs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type ReadDirFunc func(name string) ([]fs.DirEntry, error)
type StatDirFunc func(name string) (os.FileInfo, error)

func isGitRepo(path string, statDir StatDirFunc) bool {
	_, err := statDir(filepath.Join(path, ".git"))
	return err == nil
}

func dirsRecursive(root string, readDir ReadDirFunc, statDir StatDirFunc, depth, depthLimit int) (map[string]struct{}, error) {
	dirs := make(map[string]struct{})

	if depth > depthLimit {
		return dirs, nil
	}

	if isGitRepo(root, statDir) {
		return map[string]struct{}{
			root: {},
		}, nil
	}

	entries, err := readDir(root)
	if err != nil || len(entries) == 0 {
		return dirs, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		newRoot := filepath.Join(root, entry.Name())
		childGitDirs, err := dirsRecursive(newRoot, readDir, statDir, depth+1, depthLimit)

		if err != nil {
			continue
		}

		for dir := range childGitDirs {
			dirs[dir] = struct{}{}
		}
	}

	return dirs, err
}

func Dirs(root string, readDir ReadDirFunc, statDir StatDirFunc, depth int) (map[string]string, error) {
	gitDirs, err := dirsRecursive(root, readDir, statDir, 0, depth)

	dirs := make(map[string]string)
	for dir := range gitDirs {
		name, _ := filepath.Rel(root, dir)
		dirs[name] = dir
	}

	return dirs, err
}

func ProjectDirs(roots []string, readDir ReadDirFunc, statDir StatDirFunc) (map[string]string, error) {
	merged := make(map[string]string)
	for _, root := range roots {
		dirs, err := Dirs(root, readDir, statDir, 2)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", root, err)
		}
		for name, path := range dirs {
			if _, exists := merged[name]; exists {
				continue
			}
			merged[name] = path
		}
	}
	return merged, nil
}
