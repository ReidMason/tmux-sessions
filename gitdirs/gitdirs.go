package gitdirs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type ReadDirFunc func(name string) ([]fs.DirEntry, error)
type StatDirFunc func(name string) (os.FileInfo, error)

func Dirs(root string, readDir ReadDirFunc, statDir StatDirFunc) (map[string]string, error) {
	entries, err := readDir(root)
	if err != nil {
		return nil, err
	}

	dirs := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := statDir(filepath.Join(path, ".git")); err == nil {
			dirs[entry.Name()] = path
		}
	}

	return dirs, nil
}

func ProjectDirs(roots []string, readDir ReadDirFunc, statDir StatDirFunc) (map[string]string, error) {
	merged := make(map[string]string)
	for _, root := range roots {
		dirs, err := Dirs(root, readDir, statDir)
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
